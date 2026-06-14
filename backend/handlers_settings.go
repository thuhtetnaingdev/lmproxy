package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const settingDeepSeekKey = "deepseek_api_key"

// GET /api/settings/deepseek-key
func (a *API) GetDeepSeekKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	key, err := a.Store.GetSetting(settingDeepSeekKey)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"masked": ""})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"masked": maskKey(key)})
}

// PUT /api/settings/deepseek-key
func (a *API) SetDeepSeekKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}
	if err := a.Store.SetSetting(settingDeepSeekKey, req.Key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved", "masked": maskKey(req.Key)})
}

// ---- Model pricing ----

// GET /api/settings/models
func (a *API) ListModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	models, err := a.Store.ListAllPricing()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, models)
}

// POST /api/settings/models
func (a *API) CreateModel(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	var req struct {
		Model              string `json:"model"`
		InputPerMillion    int    `json:"input_per_million"`
		OutputPerMillion   int    `json:"output_per_million"`
		CacheHitPerMillion int    `json:"cache_hit_per_million"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model is required"})
		return
	}
	if err := a.Store.UpsertPricing(req.Model, req.InputPerMillion, req.OutputPerMillion, req.CacheHitPerMillion); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// PUT /api/settings/models/{model}
func (a *API) UpdateModel(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	model := chi.URLParam(r, "model")
	if model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model is required"})
		return
	}
	var req struct {
		InputPerMillion    int `json:"input_per_million"`
		OutputPerMillion   int `json:"output_per_million"`
		CacheHitPerMillion int `json:"cache_hit_per_million"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := a.Store.UpsertPricing(model, req.InputPerMillion, req.OutputPerMillion, req.CacheHitPerMillion); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DELETE /api/settings/models/{model}
func (a *API) DeleteModel(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	model := chi.URLParam(r, "model")
	if model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model is required"})
		return
	}
	if err := a.Store.DeletePricing(model); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---- Tetris settings ----

const settingTetrisBudget = "tetris_daily_budget"

// GET /api/settings/tetris-budget
func (a *API) GetTetrisBudget(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	budget := 0
	if v, err := a.Store.GetSetting(settingTetrisBudget); err == nil {
		budget, _ = strconv.Atoi(v)
	}
	writeJSON(w, http.StatusOK, map[string]int{"budget": budget})
}

// PUT /api/settings/tetris-budget
func (a *API) SetTetrisBudget(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	var req struct {
		Budget int `json:"budget"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := a.Store.SetSetting(settingTetrisBudget, strconv.Itoa(req.Budget)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "budget": req.Budget})
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
