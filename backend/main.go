package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "modernc.org/sqlite"
)

func main() {
	dsn := os.Getenv("DATABASE_PATH")
	if dsn == "" {
		dsn = "usage.db"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	if err := store.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := store.SeedPricing(); err != nil {
		log.Fatalf("seed pricing: %v", err)
	}

	// Seed admin user
	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin"
	}
	adminHash, err := HashPassword(adminPass)
	if err != nil {
		log.Fatalf("hash admin password: %v", err)
	}
	if err := store.SeedAdmin(adminUser, adminHash); err != nil {
		log.Fatalf("seed admin: %v", err)
	}
	log.Printf("admin user %q seeded", adminUser)

	proxy := &Proxy{
		Store:      store,
		BaseURL:    os.Getenv("DEEPSEEK_BASE_URL"),
		HTTPClient: &http.Client{},
	}
	if proxy.BaseURL == "" {
		proxy.BaseURL = "https://api.deepseek.com"
	}
	// API key is read from the store at request time (per-request lookup)

	api := &API{Store: store}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	// Public routes (no auth)
	loginLimiter := newLoginRateLimiter()
	r.Post("/api/auth/login", rateLimitLogin(loginLimiter, api.Login))
	r.Post("/api/auth/refresh", api.Refresh)
	r.Post("/api/auth/logout", api.Logout)

	// Proxy route — protected by bearer token (JWT or API token)
	r.Group(func(r chi.Router) {
		r.Use(ProxyAuth(store))
		r.Get("/v1/models", proxy.HandleModels)
		r.Post("/v1/chat/completions", proxy.Handle)
	})

	// Dashboard API — protected by JWT
	r.Group(func(r chi.Router) {
		r.Use(JWTAuth(store))

		r.Get("/api/auth/me", api.Me)
		r.Put("/api/auth/password", api.ChangePassword)

		r.Get("/api/usage/summary", api.Summary)
		r.Get("/api/usage/by-model", api.ByModel)
		r.Get("/api/usage/daily", api.Daily)
		r.Get("/api/usage/top-days", api.TopDays)
		r.Get("/api/usage/recent", api.Recent)
		r.Get("/api/usage/tetris", api.Tetris)

		r.Get("/api/tokens", api.ListTokens)
		r.Post("/api/tokens", api.CreateToken)
		r.Delete("/api/tokens/{id}", api.DeleteToken)

		r.Get("/api/settings/deepseek-key", api.GetDeepSeekKey)
		r.Put("/api/settings/deepseek-key", api.SetDeepSeekKey)

		r.Get("/api/settings/models", api.ListModels)
		r.Post("/api/settings/models", api.CreateModel)
		r.Put("/api/settings/models/{model}", api.UpdateModel)
		r.Delete("/api/settings/models/{model}", api.DeleteModel)

		r.Get("/api/settings/tetris-budget", api.GetTetrisBudget)
		r.Put("/api/settings/tetris-budget", api.SetTetrisBudget)
	})

	// Serve frontend static files (SPA fallback to index.html).
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../frontend/dist"
	}
	serveSPA(r, staticDir)

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("llmproxy listening on %s, proxying to %s", addr, proxy.BaseURL)
	log.Printf("serving static files from %s", staticDir)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// serveSPA serves a React/Vite SPA from dir, falling back to index.html.
func serveSPA(r chi.Router, dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		log.Printf("WARN: cannot resolve static dir %s: %v", dir, err)
		return
	}
	root := http.Dir(abs)
	fileServer := http.FileServer(root)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try the exact file first.
		path := filepath.Clean(r.URL.Path)
		f, err := root.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback: serve index.html for any unmatched path.
		http.ServeFile(w, r, filepath.Join(abs, "index.html"))
	})
}
