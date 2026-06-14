.PHONY: dev-backend dev-frontend dev build-backend build-frontend build start clean

# ── Development (two processes, hot reload) ──────────────────

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

dev:
	@echo "Starting backend + frontend (dev mode)..."
	@trap 'kill 0' EXIT; \
		cd backend && go run . & \
		cd frontend && npm run dev & \
		wait

# ── Build ────────────────────────────────────────────────────

build-backend:
	cd backend && go build -o llmproxy .

build-frontend:
	cd frontend && npm run build

build: build-frontend build-backend

# ── Production (single binary, single port) ──────────────────

start: build
	cd backend && STATIC_DIR=../frontend/dist ./llmproxy

# ── Clean ────────────────────────────────────────────────────

clean:
	rm -f backend/llmproxy
	rm -rf frontend/dist
