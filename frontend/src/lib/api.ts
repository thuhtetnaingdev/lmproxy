// ---- Auth helpers ----

const JWT_KEY = "llmproxy_jwt";
const REFRESH_KEY = "llmproxy_refresh";

export function getJWT(): string | null {
  return localStorage.getItem(JWT_KEY);
}

export function setJWT(token: string) {
  localStorage.setItem(JWT_KEY, token);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY);
}

export function setRefreshToken(token: string) {
  localStorage.setItem(REFRESH_KEY, token);
}

export function clearAuth() {
  localStorage.removeItem(JWT_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

function authHeaders(): Record<string, string> {
  const token = getJWT();
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

// ---- Base fetch with refresh interceptor ----

let refreshPromise: Promise<void> | null = null;

async function tryRefresh(): Promise<boolean> {
  const rt = getRefreshToken();
  if (!rt) return false;
  try {
    const res = await fetch("/api/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    });
    if (!res.ok) return false;
    const data = await res.json();
    setJWT(data.token);
    setRefreshToken(data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const doFetch = () =>
    fetch(url, {
      ...init,
      headers: { ...authHeaders(), ...init?.headers },
    });

  let res = await doFetch();

  if (res.status === 401 && url !== "/api/auth/login" && url !== "/api/auth/refresh") {
    // Try refresh once.
    if (!refreshPromise) {
      refreshPromise = tryRefresh().then((ok) => {
        if (!ok) {
          clearAuth();
          window.location.href = "/login";
        }
      }).finally(() => { refreshPromise = null; });
    }
    await refreshPromise;
    // Retry with new token.
    res = await doFetch();
  }

  if (!res.ok) {
    if (res.status === 401) {
      clearAuth();
      window.location.href = "/login";
    }
    throw new Error(`${url}: ${res.status}`);
  }
  return res.json();
}

// ---- Auth endpoints ----

export interface LoginResponse {
  token: string;
  refresh_token: string;
  username: string;
}

export function login(username: string, password: string): Promise<LoginResponse> {
  return fetchJSON("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
}

export function logout(refreshToken: string): Promise<void> {
  return fetchJSON("/api/auth/logout", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return fetchJSON("/api/auth/password", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });
}

export interface MeResponse {
  id: number;
  username: string;
}

export function fetchMe(): Promise<MeResponse> {
  return fetchJSON("/api/auth/me");
}

// ---- Settings ----

export interface DeepSeekKeyResponse {
  masked: string;
}

export function fetchDeepSeekKey(): Promise<DeepSeekKeyResponse> {
  return fetchJSON("/api/settings/deepseek-key");
}

export function setDeepSeekKey(key: string): Promise<{ status: string; masked: string }> {
  return fetchJSON("/api/settings/deepseek-key", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ key }),
  });
}

// ---- API tokens ----

export interface APIToken {
  id: number;
  user_id: number;
  name: string;
  prefix: string;
  cost_limit_cents: number;
  usage_cents: number;
  token?: string;
  created_at: string;
}

export function fetchTokens(): Promise<APIToken[]> {
  return fetchJSON("/api/tokens");
}

export function createToken(name: string, costLimitCents?: number): Promise<APIToken> {
  return fetchJSON("/api/tokens", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name,
      cost_limit_cents: costLimitCents && costLimitCents > 0 ? costLimitCents : null,
    }),
  });
}

export function deleteToken(id: number): Promise<void> {
  return fetchJSON("/api/tokens/" + id, { method: "DELETE" });
}

// ---- Model pricing ----

export interface PricingRow {
  model: string;
  input_per_million: number;
  output_per_million: number;
  cache_hit_per_million: number;
}

export function fetchModels(): Promise<PricingRow[]> {
  return fetchJSON("/api/settings/models");
}

export function saveModel(model: string, input: number, output: number, cacheHit: number): Promise<void> {
  return fetchJSON("/api/settings/models/" + encodeURIComponent(model), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ input_per_million: input, output_per_million: output, cache_hit_per_million: cacheHit }),
  });
}

export function createModel(model: string, input: number, output: number, cacheHit: number): Promise<void> {
  return fetchJSON("/api/settings/models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model, input_per_million: input, output_per_million: output, cache_hit_per_million: cacheHit }),
  });
}

export function deleteModel(model: string): Promise<void> {
  return fetchJSON("/api/settings/models/" + encodeURIComponent(model), { method: "DELETE" });
}

// ---- Usage endpoints ----

export interface SummaryResponse {
  total_cost_cents: number;
  total_cost_display: string;
  avg_daily_cost_cents: number;
  avg_daily_cost_display: string;
  total_requests: number;
  cache_hit_rate: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_tokens: number;
  cache_hit_tokens: number;
  cache_miss_tokens: number;
  cache_hit_cost_cents: number;
  cache_miss_cost_cents: number;
  cache_savings_cents: number;
  cache_savings_display: string;
  avg_dollars_per_1m_input: number;
  avg_dollars_per_1m_output: number;
  output_cost_cents: number;
  output_cost_display: string;
}

export interface ModelBreakdown {
  model: string;
  requests: number;
  prompt_tokens: number;
  completion_tokens: number;
  cache_hit_tokens: number;
  cache_miss_tokens: number;
  cost_cents: number;
  cost_display: string;
  cache_hit_rate: number;
}

export interface ModelDailyPoint {
  model: string;
  cost_cents: number;
}

export interface DailyPoint {
  date: string;
  requests: number;
  prompt_tokens: number;
  completion_tokens: number;
  cache_hit_tokens: number;
  cache_miss_tokens: number;
  cost_cents: number;
  cost_display: string;
  by_model: ModelDailyPoint[];
}

export interface TopDay {
  date: string;
  cost_cents: number;
  cost_display: string;
  requests: number;
  cache_hit_rate: number;
}

export interface TopDaysResponse {
  most_expensive: TopDay[];
  least_expensive: TopDay[];
  most_cache_miss: TopDay[];
  best_cache_hit_rate: TopDay[];
}

export interface RequestRecord {
  id: number;
  model: string;
  requested_at: string;
  prompt_tokens: number;
  completion_tokens: number;
  cache_hit_tokens: number;
  cache_miss_tokens: number;
  cost_cents: number;
  duration_ms: number;
}

export function fetchSummary(): Promise<SummaryResponse> {
  return fetchJSON("/api/usage/summary");
}

export function fetchByModel(): Promise<ModelBreakdown[]> {
  return fetchJSON("/api/usage/by-model");
}

export function fetchDaily(): Promise<DailyPoint[]> {
  return fetchJSON("/api/usage/daily");
}

export function fetchTopDays(limit = 5): Promise<TopDaysResponse> {
  return fetchJSON(`/api/usage/top-days?limit=${limit}`);
}

export interface RecentResponse {
  data: RequestRecord[];
  total: number;
  page: number;
  per_page: number;
}

export function fetchRecent(page = 1, perPage = 25): Promise<RecentResponse> {
  return fetchJSON(`/api/usage/recent?page=${page}&per_page=${perPage}`);
}
