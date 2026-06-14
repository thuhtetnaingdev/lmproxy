# llmproxy

DeepSeek-native LLM proxy with token usage tracking, cost analytics, and a React dashboard. Drop it in front of any OpenAI-compatible client — **single binary, zero dependencies at runtime.**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react)](https://react.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## Features

- **OpenAI-compatible proxy** — point any OpenAI SDK at `http://localhost:8080/v1`
- **Per-request metering** — prompt tokens, completion tokens, cache hit/miss, latency
- **Cost tracking** — per-model pricing with hundredths-of-a-cent precision
- **Streaming support** — buffers SSE, extracts usage from the final chunk
- **API token management** — generate tokens with optional spending limits
- **JWT + refresh token auth** — 15-minute access tokens, 72-hour refresh rotation
- **React dashboard** — shadcn/ui + Recharts, dark mode
- **Single binary** — Go backend serves the built React app on one port

## Quick start

### One-liner install

```bash
curl -sL https://raw.githubusercontent.com/thuhtetnaingdev/llmproxy/main/install.sh | sh
```

### From source

```bash
git clone https://github.com/thuhtetnaingdev/llmproxy.git
cd llmproxy

# Development (two processes, hot reload)
make dev

# Production (single binary on :8080)
make start
```

Open `http://localhost:8080` — log in with `admin` / `admin`.

## Configuration

| Env var | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Bind address |
| `DEEPSEEK_BASE_URL` | `https://api.deepseek.com` | Upstream API |
| `ADMIN_USERNAME` | `admin` | Dashboard username |
| `ADMIN_PASSWORD` | `admin` | Dashboard password |
| `JWT_SECRET` | auto-generated | Signing key for JWTs |
| `DATABASE_PATH` | `usage.db` | SQLite database path |
| `STATIC_DIR` | `../frontend/dist` | Frontend build directory |

> The DeepSeek API key is configured via the dashboard **Settings** page — not an env var.

## Dashboard

| Page | What it shows |
|---|---|
| **Dashboard** | Total cost, avg daily spend, cache hit rate, daily cost chart (stacked by model), top/bottom days |
| **Models** | Per-model token usage and cost breakdown |
| **Requests** | Paginated request log with latency |
| **Settings** | DeepSeek API key, model pricing CRUD, password change |
| **API Tokens** | Create/delete tokens with cost limits, API reference with code snippets |

## Usage

### OpenAI SDK (Python)

```python
from openai import OpenAI

client = OpenAI(
    api_key="lp_...",                     # from the API Tokens page
    base_url="http://localhost:8080/v1",
)

response = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": "Hello!"}],
)
```

### curl

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer lp_..." \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"Hello!"}]}'
```

## Architecture

```
Client (OpenAI SDK) ──Bearer token──▶ llmproxy (:8080)
                                       ├─ /v1/chat/completions  → DeepSeek API
                                       ├─ /v1/models            → model list
                                       ├─ /api/auth/*           → login, refresh, logout
                                       ├─ /api/usage/*          → analytics
                                       ├─ /api/tokens           → API token CRUD
                                       ├─ /api/settings/*       → key + pricing
                                       └─ /*                    → React SPA
                                                │
                                           SQLite (usage.db)
```

## License

MIT © [Thu Htet Naing](https://github.com/thuhtetnaingdev)
