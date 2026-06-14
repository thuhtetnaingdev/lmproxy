package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Proxy forwards /v1/chat/completions to DeepSeek and records usage.
type Proxy struct {
	Store      *Store
	BaseURL    string
	HTTPClient *http.Client
}

// chatRequest is the minimal fields we need from the incoming body.
type chatRequest struct {
	Model    string `json:"model"`
	Stream   bool   `json:"stream,omitempty"`
}

// chatResponse is the DeepSeek / OpenAI chat completion response.
// We only unmarshal the fields needed for usage tracking.
type chatResponse struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Usage *usage `json:"usage,omitempty"`
}

type usage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
	CacheHitTokens      int `json:"prompt_cache_hit_tokens"`
	CacheMissTokens     int `json:"prompt_cache_miss_tokens"`
}

// HandleModels returns a minimal model list (GET /v1/models).
// OpenAI-compatible SDKs call this to validate the endpoint before sending chat requests.
func (p *Proxy) HandleModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "deepseek-v4-flash", "object": "model", "owned_by": "deepseek"},
			{"id": "deepseek-v4-pro", "object": "model", "owned_by": "deepseek"},
		},
	})
}

// Handle is the HTTP handler for POST /v1/chat/completions.
func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only POST is supported"})
		return
	}

	start := time.Now()

	// Read the incoming body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request body"})
		return
	}
	defer r.Body.Close()

	// Detect if the client asked for streaming.
	clientWantsStream := isStreaming(bodyBytes)

	// Extract the model name for pricing lookup (before any body modifications).
	var req chatRequest
	json.Unmarshal(bodyBytes, &req)
	model := req.Model

	// Look up DeepSeek API key from store.
	deepseekKey, err := p.Store.GetSetting("deepseek_api_key")
	if err != nil || deepseekKey == "" {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "DeepSeek API key not configured — set it in the dashboard Settings"})
		return
	}

	// For streaming: forward stream=true to DeepSeek, read SSE, extract usage
	// from the final chunk, and re-emit all chunks to the client.
	// For non-streaming: strip stream, forward, parse JSON response.
	var respBytes []byte
	var respStatusCode int
	var respHeader http.Header

	if clientWantsStream {
		log.Printf("streaming request detected (model=%s), forwarding SSE", model)
		respBytes, respStatusCode, respHeader, err = p.forwardStream(r, bodyBytes, deepseekKey)
	} else {
		bodyBytes = stripStream(bodyBytes)
		respBytes, respStatusCode, respHeader, err = p.forward(r, bodyBytes, deepseekKey)
	}
	if err != nil {
		log.Printf("proxy error: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
		return
	}

	duration := time.Since(start).Milliseconds()

	// Extract usage — from SSE final chunk or from JSON response.
	if clientWantsStream {
		u := extractUsageFromSSE(respBytes)
		if u != nil {
			p.recordUsage(start, duration, model, u, r)
		} else {
			log.Printf("WARN: could not extract usage from SSE (len=%d, model=%s)", len(respBytes), model)
		}
	} else {
		var cr chatResponse
		if err := json.Unmarshal(respBytes, &cr); err == nil && cr.Usage != nil {
			p.recordUsage(start, duration, model, cr.Usage, r)
		}
	}

	// Write response back to client.
	if clientWantsStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(respStatusCode)
		w.Write(respBytes)
	} else {
		for k, vs := range respHeader {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(respStatusCode)
		w.Write(respBytes)
	}
}

// forward sends a non-streaming request to DeepSeek and returns the full response.
func (p *Proxy) forward(r *http.Request, body []byte, apiKey string) ([]byte, int, http.Header, error) {
	targetURL := p.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, resp.Header, err
}

// forwardStream sends a streaming request to DeepSeek and returns the full SSE body.
func (p *Proxy) forwardStream(r *http.Request, body []byte, apiKey string) ([]byte, int, http.Header, error) {
	targetURL := p.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Printf("stream upstream returned %d: %s", resp.StatusCode, string(b[:min(len(b), 500)]))
	}
	return b, resp.StatusCode, resp.Header, err
}

// recordUsage stores a completed request and logs it.
func (p *Proxy) recordUsage(start time.Time, duration int64, model string, u *usage, r *http.Request) {
	if model == "" {
		return
	}
	pricing, err := p.Store.LookupPricing(model)
	if err != nil {
		log.Printf("pricing lookup failed for model %q: %v", model, err)
	}

	cost := ComputeCost(u.PromptTokens, u.CompletionTokens, u.CacheHitTokens, u.CacheMissTokens, pricing)

	var tokenID int64
	if v := r.Context().Value(ctxTokenID); v != nil {
		tokenID = v.(int64)
	}

	rec := RequestRecord{
		Model:            model,
		RequestedAt:      start,
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		CacheHitTokens:   u.CacheHitTokens,
		CacheMissTokens:  u.CacheMissTokens,
		CostCents:        cost,
		DurationMs:       duration,
		APITokenID:       tokenID,
	}

	if err := p.Store.InsertRequest(rec); err != nil {
		log.Printf("failed to store request: %v", err)
	}

	log.Printf("model=%s prompt=%d comp=%d cache_hit=%d cache_miss=%d cost=%s dur=%dms",
		model, u.PromptTokens, u.CompletionTokens,
		u.CacheHitTokens, u.CacheMissTokens,
		centsToDisplay(cost), duration,
	)
}

// extractUsageFromSSE parses the final SSE data chunk for usage info.
func extractUsageFromSSE(body []byte) *usage {
	s := string(body)
	events := splitSSE(s)

	// Parse the last non-[DONE] event.
	for i := len(events) - 1; i >= 0; i-- {
		e := strings.TrimSpace(events[i])
		if e == "" || e == "[DONE]" || e == "data: [DONE]" {
			continue
		}
		jsonStr := strings.TrimPrefix(e, "data: ")
		jsonStr = strings.TrimPrefix(jsonStr, "data:")
		if jsonStr == "" {
			continue
		}
		var chunk struct {
			Usage *usage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &chunk); err == nil && chunk.Usage != nil {
			return chunk.Usage
		}
		break
	}
	return nil
}

// splitSSE splits a raw SSE body into individual events.
// Handles both \n\n and \r\n\r\n separators.
func splitSSE(s string) []string {
	// Normalize \r\n to \n first.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n\n")
}

// isStreaming checks whether the request body has "stream":true.
func isStreaming(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	v, ok := m["stream"]
	return ok && v == true
}

// stripStream removes or overrides "stream":true in the JSON body so we always
// get a parseable non-streaming response from DeepSeek.
func stripStream(body []byte) []byte {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return body // can't parse, forward as-is
	}
	if v, ok := m["stream"]; ok && v == true {
		m["stream"] = false
		delete(m, "stream_options") // invalid when stream=false
		b, err := json.Marshal(m)
		if err != nil {
			return body
		}
		return b
	}
	return body
}
