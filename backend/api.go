package main

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"strconv"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Guard: typed nil slices → [] so the frontend never sees null.
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		v = reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

// API serves the dashboard endpoints.
type API struct {
	Store *Store
}

// Summary returns aggregate usage stats: total cost, cache hit rate, avg $/1M, etc.
func (a *API) Summary(w http.ResponseWriter, r *http.Request) {
	s, err := a.Store.Summary()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ByModel returns per-model breakdowns with cost and cache hit rate.
func (a *API) ByModel(w http.ResponseWriter, r *http.Request) {
	models, err := a.Store.ByModel()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, models)
}

// Daily returns per-day aggregates with per-model cost breakdown.
func (a *API) Daily(w http.ResponseWriter, r *http.Request) {
	points, err := a.Store.Daily()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, points)
}

// TopDays returns the most/least expensive, most cache-miss, and best cache-hit days.
func (a *API) TopDays(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 5)
	td, err := a.Store.TopDays(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, td)
}

// Recent returns a page of recent requests.
func (a *API) Recent(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 25)
	recs, total, err := a.Store.RecentPaginated(page, perPage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     recs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// Tetris returns game data for Token Tetris.
func (a *API) Tetris(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuth(w, r); !ok {
		return
	}
	budget := 0
	if v, err := a.Store.GetSetting("tetris_daily_budget"); err == nil {
		if n, e := strconv.Atoi(v); e == nil {
			budget = n
		}
	}
	data, err := a.Store.TetrisData(budget)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
