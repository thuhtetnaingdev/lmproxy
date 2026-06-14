package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ---- Types ----

// Pricing in hundredths-of-a-cent per 1M tokens (1 = $0.0001).
type Pricing struct {
	InputPerMillion    int `json:"input_per_million"`
	OutputPerMillion   int `json:"output_per_million"`
	CacheHitPerMillion int `json:"cache_hit_per_million"`
}

// PricingRow is a full model pricing row for the API.
type PricingRow struct {
	Model              string `json:"model"`
	InputPerMillion    int    `json:"input_per_million"`
	OutputPerMillion   int    `json:"output_per_million"`
	CacheHitPerMillion int    `json:"cache_hit_per_million"`
}

// RequestRecord mirrors a row in the requests table.
// CostCents is in hundredths of a cent ($0.0001 units).
type RequestRecord struct {
	ID               int64     `json:"id"`
	Model            string    `json:"model"`
	RequestedAt      time.Time `json:"requested_at"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CacheHitTokens   int       `json:"cache_hit_tokens"`
	CacheMissTokens  int       `json:"cache_miss_tokens"`
	CostCents        int       `json:"cost_cents"`
	DurationMs       int64     `json:"duration_ms"`
	APITokenID       int64     `json:"api_token_id,omitempty"`
}

// ---- Response DTOs ----

type SummaryResponse struct {
	TotalCostCents        int     `json:"total_cost_cents"`
	TotalCostDisplay      string  `json:"total_cost_display"`
	AvgDailyCostCents     float64 `json:"avg_daily_cost_cents"`
	AvgDailyCostDisplay   string  `json:"avg_daily_cost_display"`
	TotalRequests         int     `json:"total_requests"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	TotalPromptTokens     int     `json:"total_prompt_tokens"`
	TotalCompletionTokens int     `json:"total_completion_tokens"`
	TotalTokens           int     `json:"total_tokens"`
	CacheHitTokens        int     `json:"cache_hit_tokens"`
	CacheMissTokens       int     `json:"cache_miss_tokens"`
	CacheHitCostCents     int     `json:"cache_hit_cost_cents"`
	CacheMissCostCents    int     `json:"cache_miss_cost_cents"`
	CacheSavingsCents     int     `json:"cache_savings_cents"`
	CacheSavingsDisplay   string  `json:"cache_savings_display"`
	AvgDollarsPer1MInput  float64 `json:"avg_dollars_per_1m_input"`
	AvgDollarsPer1MOutput float64 `json:"avg_dollars_per_1m_output"`
	OutputCostCents       int     `json:"output_cost_cents"`
	OutputCostDisplay     string  `json:"output_cost_display"`
}

type ModelBreakdown struct {
	Model            string  `json:"model"`
	Requests         int     `json:"requests"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CacheHitTokens   int     `json:"cache_hit_tokens"`
	CacheMissTokens  int     `json:"cache_miss_tokens"`
	CostCents        int     `json:"cost_cents"`
	CostDisplay      string  `json:"cost_display"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

type ModelDailyPoint struct {
	Model     string `json:"model"`
	CostCents int    `json:"cost_cents"`
}

type DailyPoint struct {
	Date             string            `json:"date"`
	Requests         int               `json:"requests"`
	PromptTokens     int               `json:"prompt_tokens"`
	CompletionTokens int               `json:"completion_tokens"`
	CacheHitTokens   int               `json:"cache_hit_tokens"`
	CacheMissTokens  int               `json:"cache_miss_tokens"`
	CostCents        int               `json:"cost_cents"`
	CostDisplay      string            `json:"cost_display"`
	ByModel          []ModelDailyPoint `json:"by_model"`
}

type TopDay struct {
	Date         string  `json:"date"`
	CostCents    int     `json:"cost_cents"`
	CostDisplay  string  `json:"cost_display"`
	Requests     int     `json:"requests"`
	CacheHitRate float64 `json:"cache_hit_rate"`
}

type TopDaysResponse struct {
	MostExpensive    []TopDay `json:"most_expensive"`
	LeastExpensive   []TopDay `json:"least_expensive"`
	MostCacheMiss    []TopDay `json:"most_cache_miss"`
	BestCacheHitRate []TopDay `json:"best_cache_hit_rate"`
}

// ---- Auth types ----

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type APIToken struct {
	ID             int64  `json:"id"`
	UserID         int64  `json:"user_id"`
	Name           string `json:"name"`
	Prefix         string `json:"prefix"`
	CostLimitCents int64  `json:"cost_limit_cents"`
	UsageCents     int64  `json:"usage_cents"`
	Token          string `json:"token,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// ---- Store ----

type Store struct {
	DB *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{DB: db} }

// Migrate creates tables if they don't exist.
func (s *Store) Migrate() error {
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS model_pricing (
			model                 TEXT PRIMARY KEY,
			input_per_million     INTEGER NOT NULL,
			output_per_million    INTEGER NOT NULL,
			cache_hit_per_million INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS requests (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			model             TEXT NOT NULL,
			requested_at      TEXT NOT NULL,
			prompt_tokens     INTEGER NOT NULL,
			completion_tokens INTEGER NOT NULL,
			cache_hit_tokens  INTEGER NOT NULL DEFAULT 0,
			cache_miss_tokens INTEGER NOT NULL DEFAULT 0,
			cost_cents        INTEGER NOT NULL,
			duration_ms       INTEGER NOT NULL DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_requests_date ON requests(requested_at);
		CREATE INDEX IF NOT EXISTS idx_requests_model ON requests(model);

		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS api_tokens (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id          INTEGER NOT NULL REFERENCES users(id),
			name             TEXT NOT NULL,
			token_hash       TEXT NOT NULL UNIQUE,
			prefix           TEXT NOT NULL,
			cost_limit_cents INTEGER,  -- NULL = unlimited, in hundredths-of-a-cent
			created_at       TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL
		);
	`)
	// Idempotent upgrades (ignore errors if columns already exist).
	s.DB.Exec(`ALTER TABLE requests ADD COLUMN api_token_id INTEGER DEFAULT NULL`)
	s.DB.Exec(`ALTER TABLE api_tokens ADD COLUMN cost_limit_cents INTEGER DEFAULT NULL`)
	s.DB.Exec(`ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 1`)
	return err
}

// SeedPricing inserts current DeepSeek pricing (idempotent via INSERT OR IGNORE).
func (s *Store) SeedPricing() error {
	rows := []struct {
		model                 string
		input, output, cacheHit int
	}{
		// https://api-docs.deepseek.com/quick_start/pricing
		{"deepseek-v4-flash", 1400, 2800, 28},     // $0.14 input, $0.28 output, $0.0028 cache hit
		{"deepseek-v4-pro", 4350, 8700, 36},      // $0.435 input, $0.87 output, $0.003625 cache hit
	}
	for _, r := range rows {
		_, err := s.DB.Exec(
			`INSERT OR IGNORE INTO model_pricing (model, input_per_million, output_per_million, cache_hit_per_million)
			 VALUES (?, ?, ?, ?)`,
			r.model, r.input, r.output, r.cacheHit,
		)
		if err != nil {
			return fmt.Errorf("seed pricing %s: %w", r.model, err)
		}
	}
	return nil
}

// GetPricing returns the pricing row for a model, or a default if unknown.
func (s *Store) GetPricing(model string) (Pricing, error) {
	var p Pricing
	err := s.DB.QueryRow(
		`SELECT input_per_million, output_per_million, cache_hit_per_million
		 FROM model_pricing WHERE model = ?`, model,
	).Scan(&p.InputPerMillion, &p.OutputPerMillion, &p.CacheHitPerMillion)
	if err == sql.ErrNoRows {
		// Default fallback — assume deepseek-chat rates
		return Pricing{InputPerMillion: 1400, OutputPerMillion: 2800, CacheHitPerMillion: 28}, nil
	}
	return p, err
}

// InsertRequest stores a completed request.
func (s *Store) InsertRequest(r RequestRecord) error {
	_, err := s.DB.Exec(
		`INSERT INTO requests (model, requested_at, prompt_tokens, completion_tokens,
		 cache_hit_tokens, cache_miss_tokens, cost_cents, duration_ms, api_token_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Model, r.RequestedAt.UTC().Format(time.RFC3339),
		r.PromptTokens, r.CompletionTokens,
		r.CacheHitTokens, r.CacheMissTokens,
		r.CostCents, r.DurationMs,
		r.APITokenID,
	)
	return err
}

// ---- Aggregation queries ----

// Summary returns overall usage stats.
func (s *Store) Summary() (SummaryResponse, error) {
	var r SummaryResponse

	row := s.DB.QueryRow(`
		SELECT
			COALESCE(SUM(cost_cents), 0),
			COUNT(*),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(cache_hit_tokens), 0),
			COALESCE(SUM(cache_miss_tokens), 0)
		FROM requests
	`)
	err := row.Scan(&r.TotalCostCents, &r.TotalRequests,
		&r.TotalPromptTokens, &r.TotalCompletionTokens,
		&r.CacheHitTokens, &r.CacheMissTokens)
	if err != nil {
		return r, err
	}

	r.TotalTokens = r.TotalPromptTokens + r.TotalCompletionTokens
	r.TotalCostDisplay = centsToDisplay(r.TotalCostCents)

	// Cache hit rate
	totalPrompt := r.CacheHitTokens + r.CacheMissTokens
	if totalPrompt > 0 {
		r.CacheHitRate = float64(r.CacheHitTokens) / float64(totalPrompt) * 100
	}

	// Cache cost breakdown: how much was spent on cache-hit vs cache-miss input tokens
	r.CacheHitCostCents = estimateCost(r.CacheHitTokens, 28)    // rough: cache-hit rate ($0.0028)
	r.CacheMissCostCents = estimateCost(r.CacheMissTokens, 1400) // rough: cache-miss rate ($0.14)

	// Cache savings: if all cache-hit tokens had been cache-miss instead
	hypotheticalMissCost := estimateCost(r.CacheHitTokens, 1400)
	r.CacheSavingsCents = hypotheticalMissCost - r.CacheHitCostCents
	if r.CacheSavingsCents < 0 {
		r.CacheSavingsCents = 0
	}
	r.CacheSavingsDisplay = centsToDisplay(r.CacheSavingsCents)

	// Output cost: approximated at chat output rate
	r.OutputCostCents = estimateCost(r.TotalCompletionTokens, 2800) // $0.28/1M output
	r.OutputCostDisplay = centsToDisplay(r.OutputCostCents)

	// Avg $/1M tokens
	if r.TotalPromptTokens > 0 {
		// Compute using actual cost data from cache-hit and cache-miss rates
		r.AvgDollarsPer1MInput = float64(r.CacheHitCostCents+r.CacheMissCostCents) / float64(r.CacheHitTokens+r.CacheMissTokens) * 10000 / 100
	}
	if r.TotalCompletionTokens > 0 {
		r.AvgDollarsPer1MOutput = float64(r.OutputCostCents) / float64(r.TotalCompletionTokens) * 10000 / 100
	}

	// Avg daily cost
	var dayCount int
	s.DB.QueryRow(`SELECT COUNT(DISTINCT DATE(requested_at)) FROM requests`).Scan(&dayCount)
	if dayCount > 0 {
		r.AvgDailyCostCents = float64(r.TotalCostCents) / float64(dayCount)
	}
	r.AvgDailyCostDisplay = centsToDisplay(int(r.AvgDailyCostCents))

	return r, nil
}

// ByModel returns per-model aggregates.
func (s *Store) ByModel() ([]ModelBreakdown, error) {
	rows, err := s.DB.Query(`
		SELECT
			model,
			COUNT(*),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(cache_hit_tokens), 0),
			COALESCE(SUM(cache_miss_tokens), 0),
			COALESCE(SUM(cost_cents), 0)
		FROM requests
		GROUP BY model
		ORDER BY SUM(cost_cents) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModelBreakdown
	for rows.Next() {
		var m ModelBreakdown
		if err := rows.Scan(&m.Model, &m.Requests,
			&m.PromptTokens, &m.CompletionTokens,
			&m.CacheHitTokens, &m.CacheMissTokens,
			&m.CostCents); err != nil {
			return nil, err
		}
		m.CostDisplay = centsToDisplay(m.CostCents)
		total := m.CacheHitTokens + m.CacheMissTokens
		if total > 0 {
			m.CacheHitRate = float64(m.CacheHitTokens) / float64(total) * 100
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Daily returns per-day aggregates with per-model breakdown.
func (s *Store) Daily() ([]DailyPoint, error) {
	// Main daily query
	rows, err := s.DB.Query(`
		SELECT
			DATE(requested_at) AS day,
			COUNT(*),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(cache_hit_tokens), 0),
			COALESCE(SUM(cache_miss_tokens), 0),
			COALESCE(SUM(cost_cents), 0)
		FROM requests
		GROUP BY day
		ORDER BY day ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DailyPoint
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Requests, &p.PromptTokens,
			&p.CompletionTokens, &p.CacheHitTokens, &p.CacheMissTokens,
			&p.CostCents); err != nil {
			return nil, err
		}
		p.CostDisplay = centsToDisplay(p.CostCents)
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Per-model breakdown for each day
	if len(points) > 0 {
		modelRows, err := s.DB.Query(`
			SELECT DATE(requested_at) AS day, model, COALESCE(SUM(cost_cents), 0)
			FROM requests
			GROUP BY day, model
			ORDER BY day ASC, SUM(cost_cents) DESC
		`)
		if err != nil {
			return nil, err
		}
		defer modelRows.Close()

		// Build a map: day -> []ModelDailyPoint
		byModel := map[string][]ModelDailyPoint{}
		for modelRows.Next() {
			var day string
			var mp ModelDailyPoint
			if err := modelRows.Scan(&day, &mp.Model, &mp.CostCents); err != nil {
				return nil, err
			}
			byModel[day] = append(byModel[day], mp)
		}
		if err := modelRows.Err(); err != nil {
			return nil, err
		}

		for i := range points {
			points[i].ByModel = byModel[points[i].Date]
			if points[i].ByModel == nil {
				points[i].ByModel = []ModelDailyPoint{}
			}
		}
	}

	// Fill in missing days for the current month (zero values).
	points = fillMonthGaps(points)

	return points, nil
}

// fillMonthGaps adds zero-value entries for every day of the current month
// that doesn't already appear in the points slice.
func fillMonthGaps(points []DailyPoint) []DailyPoint {
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1) // last day of current month

	byDate := map[string]DailyPoint{}
	for _, p := range points {
		byDate[p.Date] = p
	}

	var out []DailyPoint
	for d := firstOfMonth; !d.After(lastOfMonth); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if p, ok := byDate[dateStr]; ok {
			out = append(out, p)
		} else {
			out = append(out, DailyPoint{
				Date:        dateStr,
				CostDisplay: "$0.0000",
				ByModel:     []ModelDailyPoint{},
			})
		}
	}
	return out
}

// TopDays returns the best/worst days by various metrics.
func (s *Store) TopDays(limit int) (TopDaysResponse, error) {
	if limit <= 0 {
		limit = 5
	}
	var r TopDaysResponse

	queries := []struct {
		order   string
		target  *[]TopDay
	}{
		{"SUM(cost_cents) DESC", &r.MostExpensive},
		{"SUM(cost_cents) ASC", &r.LeastExpensive},
		{"SUM(cache_miss_tokens) DESC", &r.MostCacheMiss},
		{"CAST(SUM(cache_hit_tokens) AS REAL) / NULLIF(SUM(cache_hit_tokens)+SUM(cache_miss_tokens), 0) DESC", &r.BestCacheHitRate},
	}

	for _, q := range queries {
		rows, err := s.DB.Query(fmt.Sprintf(`
			SELECT
				DATE(requested_at) AS day,
				COALESCE(SUM(cost_cents), 0),
				COUNT(*),
				COALESCE(SUM(cache_hit_tokens), 0),
				COALESCE(SUM(cache_miss_tokens), 0)
			FROM requests
			GROUP BY day
			ORDER BY %s
			LIMIT ?
		`, q.order), limit)
		if err != nil {
			return r, err
		}

		var list []TopDay
		for rows.Next() {
			var d TopDay
			var hit, miss int
			if err := rows.Scan(&d.Date, &d.CostCents, &d.Requests, &hit, &miss); err != nil {
				rows.Close()
				return r, err
			}
			d.CostDisplay = centsToDisplay(d.CostCents)
			total := hit + miss
			if total > 0 {
				d.CacheHitRate = float64(hit) / float64(total) * 100
			}
			list = append(list, d)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return r, err
		}
		*q.target = list
	}

	return r, nil
}

// Recent returns the most recent N requests.
func (s *Store) Recent(limit int) ([]RequestRecord, error) {
	recs, _, err := s.RecentPaginated(1, limit)
	return recs, err
}

// RecentPaginated returns a page of requests with total count.
func (s *Store) RecentPaginated(page, perPage int) ([]RequestRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	var total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM requests`).Scan(&total)

	rows, err := s.DB.Query(`
		SELECT id, model, requested_at, prompt_tokens, completion_tokens,
		       cache_hit_tokens, cache_miss_tokens, cost_cents, duration_ms, COALESCE(api_token_id, 0)
		FROM requests
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []RequestRecord
	for rows.Next() {
		var r RequestRecord
		var ts string
		if err := rows.Scan(&r.ID, &r.Model, &ts,
			&r.PromptTokens, &r.CompletionTokens,
			&r.CacheHitTokens, &r.CacheMissTokens,
			&r.CostCents, &r.DurationMs, &r.APITokenID); err != nil {
			return nil, 0, err
		}
		r.RequestedAt, _ = time.Parse(time.RFC3339, ts)
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// ---- Auth queries ----

// SeedAdmin creates the admin user if it doesn't exist.
func (s *Store) SeedAdmin(username, passwordHash string) error {
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO users (username, password_hash) VALUES (?, ?)`,
		username, passwordHash,
	)
	return err
}

// GetUserByUsername returns a user by username.
func (s *Store) GetUserByUsername(username string) (User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, username, password_hash FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash)
	return u, err
}

// ---- Settings ----

func (s *Store) GetSetting(key string) (string, error) {
	var v string
	err := s.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	return v, err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}

// ---- API tokens ----

func (s *Store) CreateAPIToken(userID int64, name, tokenHash, prefix string, costLimitCents *int64) (int64, error) {
	res, err := s.DB.Exec(
		`INSERT INTO api_tokens (user_id, name, token_hash, prefix, cost_limit_cents, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, name, tokenHash, prefix, costLimitCents, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListAPITokens(userID int64) ([]APIToken, error) {
	rows, err := s.DB.Query(
		`SELECT id, user_id, name, prefix, COALESCE(cost_limit_cents, 0), created_at FROM api_tokens WHERE user_id = ? ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIToken
	for rows.Next() {
		var t APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Prefix, &t.CostLimitCents, &t.CreatedAt); err != nil {
			return nil, err
		}
		// Compute usage for this token.
		s.DB.QueryRow(`SELECT COALESCE(SUM(cost_cents), 0) FROM requests WHERE api_token_id = ?`, t.ID).Scan(&t.UsageCents)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAPIToken(userID, tokenID int64) error {
	_, err := s.DB.Exec(`DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, tokenID, userID)
	return err
}

// ValidateAPIToken looks up a token by hash and returns the associated user ID.
func (s *Store) ValidateAPIToken(tokenHash string) (int64, error) {
	_, userID, err := s.ValidateAPITokenFull(tokenHash)
	return userID, err
}

// ValidateAPITokenFull looks up a token by hash and returns (tokenID, userID, error).
func (s *Store) ValidateAPITokenFull(tokenHash string) (int64, int64, error) {
	var id, userID int64
	err := s.DB.QueryRow(`SELECT id, user_id FROM api_tokens WHERE token_hash = ?`, tokenHash).Scan(&id, &userID)
	return id, userID, err
}

// ---- Model pricing CRUD ----

func (s *Store) ListAllPricing() ([]PricingRow, error) {
	rows, err := s.DB.Query(`SELECT model, input_per_million, output_per_million, cache_hit_per_million FROM model_pricing ORDER BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PricingRow
	for rows.Next() {
		var r PricingRow
		if err := rows.Scan(&r.Model, &r.InputPerMillion, &r.OutputPerMillion, &r.CacheHitPerMillion); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) UpsertPricing(model string, input, output, cacheHit int) error {
	_, err := s.DB.Exec(
		`INSERT INTO model_pricing (model, input_per_million, output_per_million, cache_hit_per_million)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(model) DO UPDATE SET input_per_million=excluded.input_per_million,
		 output_per_million=excluded.output_per_million, cache_hit_per_million=excluded.cache_hit_per_million`,
		model, input, output, cacheHit,
	)
	return err
}

func (s *Store) DeletePricing(model string) error {
	_, err := s.DB.Exec(`DELETE FROM model_pricing WHERE model = ?`, model)
	return err
}

// ---- Refresh tokens ----

func (s *Store) CreateRefreshToken(userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.DB.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		userID, tokenHash, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

// ConsumeRefreshToken atomically validates and deletes a refresh token, returning the user ID.
func (s *Store) ConsumeRefreshToken(tokenHash string) (int64, error) {
	var userID int64
	var expStr string
	err := s.DB.QueryRow(
		`DELETE FROM refresh_tokens WHERE token_hash = ? RETURNING user_id, expires_at`,
		tokenHash,
	).Scan(&userID, &expStr)
	if err != nil {
		return 0, fmt.Errorf("invalid refresh token")
	}
	exp, err := time.Parse(time.RFC3339, expStr)
	if err != nil || time.Now().After(exp) {
		return 0, fmt.Errorf("token expired")
	}
	return userID, nil
}

func (s *Store) ValidateRefreshToken(tokenHash string) (int64, error) {
	var userID int64
	var expStr string
	err := s.DB.QueryRow(
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&userID, &expStr)
	if err != nil {
		return 0, err
	}
	exp, err := time.Parse(time.RFC3339, expStr)
	if err != nil || time.Now().After(exp) {
		return 0, fmt.Errorf("token expired")
	}
	return userID, nil
}

func (s *Store) DeleteRefreshToken(tokenHash string) error {
	_, err := s.DB.Exec(`DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) DeleteAllRefreshTokens(userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	return err
}

// ---- Password ----

func (s *Store) UpdatePassword(userID int64, newHash string) error {
	_, err := s.DB.Exec(
		`UPDATE users SET password_hash = ?, token_version = token_version + 1 WHERE id = ?`,
		newHash, userID,
	)
	return err
}

// ---- Helpers ----

// ---- Tetris types ----

type TetrisPiece struct {
	Time      string `json:"time"`
	CostCents int    `json:"cost_cents"`
	Tokens    int    `json:"tokens"`
	Model     string `json:"model"`
	CacheHit  bool   `json:"cache_hit"`
}

type TetrisResponse struct {
	TodayCostCents int           `json:"today_cost_cents"`
	BudgetCents    int           `json:"budget_cents"`
	Percentage     float64       `json:"percentage"`
	GameOver       bool          `json:"game_over"`
	Streak         int           `json:"streak"`
	BestStreak     int           `json:"best_streak"`
	Pieces         []TetrisPiece `json:"pieces"`
}

// TetrisData returns today's request pieces and streak info for the game.
// Budget is in hundredths-of-a-cent (same unit as cost_cents in requests).
func (s *Store) TetrisData(budget int) (TetrisResponse, error) {
	today := time.Now().UTC().Format("2006-01-02")

	// Fetch today's requests ordered by time.
	rows, err := s.DB.Query(`
		SELECT requested_at, cost_cents, prompt_tokens + completion_tokens AS tokens, model,
		       CASE WHEN cache_hit_tokens > 0 THEN 1 ELSE 0 END
		FROM requests
		WHERE DATE(requested_at) = ?
		ORDER BY requested_at ASC
	`, today)
	if err != nil {
		return TetrisResponse{}, err
	}
	defer rows.Close()

	var pieces []TetrisPiece
	var todayCostCents int
	for rows.Next() {
		var ts string
		var p TetrisPiece
		var ch int
		if err := rows.Scan(&ts, &p.CostCents, &p.Tokens, &p.Model, &ch); err != nil {
			return TetrisResponse{}, err
		}
		p.CacheHit = ch == 1
		// Extract HH:MM from RFC3339 timestamp.
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			p.Time = t.Format("15:04")
		} else {
			p.Time = ts
		}
		pieces = append(pieces, p)
		todayCostCents += p.CostCents
	}
	if err := rows.Err(); err != nil {
		return TetrisResponse{}, err
	}

	pct := 0.0
	if budget > 0 {
		pct = float64(todayCostCents) / float64(budget) * 100
	}

	streak, best := s.computeStreak(budget, today, todayCostCents)

	return TetrisResponse{
		TodayCostCents: todayCostCents,
		BudgetCents:    budget,
		Percentage:     pct,
		GameOver:       budget > 0 && todayCostCents > budget,
		Streak:         streak,
		BestStreak:     best,
		Pieces:         pieces,
	}, nil
}

// computeStreak walks backwards from today counting consecutive days <= budget.
// Budget and daily totals are in hundredths-of-a-cent.
func (s *Store) computeStreak(budget int, today string, todayCostCents int) (streak, best int) {
	// Count today if not over budget.
	if budget > 0 && todayCostCents <= budget {
		streak = 1
	}

	// Walk backwards day-by-day.
	d, _ := time.Parse("2006-01-02", today)
	for {
		d = d.AddDate(0, 0, -1)
		day := d.Format("2006-01-02")
		var costCents int
		err := s.DB.QueryRow(`
			SELECT COALESCE(SUM(cost_cents), 0)
			FROM requests WHERE DATE(requested_at) = ?
		`, day).Scan(&costCents)
		if err != nil {
			break
		}
		if costCents == 0 && day != today {
			break
		}
		if budget > 0 && costCents <= budget {
			streak++
		} else {
			break
		}
	}

	// Best streak: scan all days with data.
	type dayTotal struct {
		day       string
		costCents int
	}
	var days []dayTotal
	rows, err := s.DB.Query(`
		SELECT DATE(requested_at) AS day, SUM(cost_cents) AS cost_cents
		FROM requests GROUP BY day ORDER BY day ASC
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dt dayTotal
			if rows.Scan(&dt.day, &dt.costCents) == nil {
				days = append(days, dt)
			}
		}

		cur := 0
		for _, dt := range days {
			if budget > 0 && dt.costCents <= budget {
				cur++
				if cur > best {
					best = cur
				}
			} else {
				cur = 0
			}
		}
	}
	if streak > best {
		best = streak
	}
	return
}

// centsToDisplay converts hundredths-of-a-cent (1 = $0.0001) to a dollar string.
func centsToDisplay(c int) string {
	dollars := float64(c) / 10000.0
	return fmt.Sprintf("$%.4f", dollars)
}

// estimateCost estimates cost for a token count at a given rate (hundredths-of-a-cent per 1M).
// Result is in hundredths-of-a-cent.
func estimateCost(tokens int, pricePerMillion int) int {
	// tokens * pricePerMillion / 1_000_000
	return (tokens * pricePerMillion) / 1_000_000
}

// ComputeCost calculates the cost for a single request.
// cacheHitTokens + cacheMissTokens should sum to prompt_tokens.
// Returns cost in hundredths-of-a-cent.
func ComputeCost(promptTokens, completionTokens, cacheHitTokens, cacheMissTokens int, p Pricing) int {
	hitCost := estimateCost(cacheHitTokens, p.CacheHitPerMillion)
	missCost := estimateCost(cacheMissTokens, p.InputPerMillion)
	// If cache breakdown not provided, treat all prompt as cache-miss
	if cacheHitTokens == 0 && cacheMissTokens == 0 {
		missCost = estimateCost(promptTokens, p.InputPerMillion)
	}
	outputCost := estimateCost(completionTokens, p.OutputPerMillion)
	return hitCost + missCost + outputCost
}

// matchModel extracts the short model name for pricing lookup.
// Strips provider prefixes like "deepseek/" or dates like "2025-01-01".
func matchModel(fullModel string) string {
	// Common DeepSeek patterns: "deepseek-chat", "deepseek-reasoner"
	// Also handle "deepseek/deepseek-chat" and dated snapshots.
	m := fullModel
	// Remove vendor prefix
	if idx := strings.Index(m, "/"); idx >= 0 {
		m = m[idx+1:]
	}
	return m
}

// LookupPricing finds the best pricing match for a model string.
func (s *Store) LookupPricing(fullModel string) (Pricing, error) {
	m := matchModel(fullModel)

	// Try exact match first
	p, err := s.GetPricing(m)
	if err == nil {
		return p, nil
	}

	// Prefix match: "deepseek-chat-2025-..." should match "deepseek-chat"
	rows, err := s.DB.Query(`SELECT model, input_per_million, output_per_million, cache_hit_per_million FROM model_pricing`)
	if err != nil {
		return Pricing{}, err
	}
	defer rows.Close()

	var best Pricing
	bestLen := 0
	for rows.Next() {
		var model string
		var pr Pricing
		if err := rows.Scan(&model, &pr.InputPerMillion, &pr.OutputPerMillion, &pr.CacheHitPerMillion); err != nil {
			return Pricing{}, err
		}
		if strings.HasPrefix(m, model) && len(model) > bestLen {
			best = pr
			bestLen = len(model)
		}
	}
	if bestLen > 0 {
		return best, nil
	}
	// Default fallback
	return Pricing{InputPerMillion: 1400, OutputPerMillion: 2800, CacheHitPerMillion: 28}, nil
}
