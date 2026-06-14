import { useEffect, useState } from "react";
import { fetchTokens, createToken, deleteToken, fetchModels, type APIToken, type PricingRow } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Plus, Trash2, Copy, Check, Code, Terminal, Globe } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const API_BASE = window.location.origin.replace("5173", "8080");

export default function Tokens() {
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState("");
  const [costLimit, setCostLimit] = useState("");
  const [creating, setCreating] = useState(false);
  const [newToken, setNewToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [docToken, setDocToken] = useState("<your-token>");
  const [models, setModels] = useState<PricingRow[]>([]);
  const exampleModel = models[0]?.model ?? "deepseek-chat";

  function load() {
    fetchTokens().then(setTokens).catch(console.error).finally(() => setLoading(false));
    fetchModels().then(setModels).catch(console.error);
  }

  useEffect(() => { load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    try {
      const limit = costLimit ? Math.round(parseFloat(costLimit) * 10000) : undefined;
      const t = await createToken(name.trim(), limit);
      setNewToken(t.token ?? null);
      setName("");
      setCostLimit("");
      load();
    } catch {
      // ignore
    } finally {
      setCreating(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteToken(id);
      setTokens((prev) => prev.filter((t) => t.id !== id));
    } catch {
      // ignore
    }
  }

  async function copyToken() {
    if (!newToken) return;
    await navigator.clipboard.writeText(newToken);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-lg font-semibold">API Tokens</h2>
        <p className="text-sm text-muted-foreground">
          Generate bearer tokens for clients to call the proxy at <code className="text-xs bg-muted px-1 rounded">/v1/chat/completions</code>.
        </p>
      </div>

      {/* New token banner */}
      {newToken && (
        <Card className="border-green-500 bg-green-50 dark:bg-green-950">
          <CardContent className="pt-4 space-y-2">
            <p className="text-sm font-semibold text-green-800 dark:text-green-200">
              Token created — copy it now. You won't see it again.
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs bg-background border rounded px-3 py-2 break-all">
                {newToken}
              </code>
              <Button variant="outline" size="icon" onClick={copyToken}>
                {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <Button variant="ghost" size="sm" onClick={() => setNewToken(null)}>
              Dismiss
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Create form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Create Token</CardTitle>
          <CardDescription>Give it a name and optional spending limit.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex gap-2 items-end">
            <div className="flex-1 space-y-1">
              <label className="text-xs font-medium">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. my-app, cli-tool"
                className="w-full px-3 py-2 border rounded-md text-sm bg-background"
                required
              />
            </div>
            <div className="w-36 space-y-1">
              <label className="text-xs font-medium">Cost Limit ($)</label>
              <input
                type="number"
                step="any"
                value={costLimit}
                onChange={(e) => setCostLimit(e.target.value)}
                placeholder="∞ unlimited"
                className="w-full px-3 py-2 border rounded-md text-sm bg-background"
              />
            </div>
            <Button type="submit" disabled={creating || !name.trim()}>
              <Plus className="h-4 w-4 mr-1" />
              Create
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* Token list */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Existing Tokens</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Prefix</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Usage</TableHead>
                <TableHead className="text-right">Limit</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {!loading && tokens.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-4">
                    No tokens yet.
                  </TableCell>
                </TableRow>
              )}
              {tokens.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-medium text-sm">{t.name}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{t.prefix}...</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(t.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right text-xs tabular-nums">
                    ${(t.usage_cents / 10000).toFixed(4)}
                  </TableCell>
                  <TableCell className="text-right text-xs tabular-nums">
                    {t.cost_limit_cents > 0
                      ? `$${(t.cost_limit_cents / 10000).toFixed(2)}`
                      : "∞"}
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="icon" onClick={() => handleDelete(t.id)}>
                      <Trash2 className="h-4 w-4 text-red-500" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Documentation */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Globe className="h-4 w-4" />
            API Reference
          </CardTitle>
          <CardDescription>
            The proxy is OpenAI-compatible — use any OpenAI SDK by pointing <code className="text-xs bg-muted px-1 rounded">base_url</code> here.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Endpoint */}
          <div>
            <p className="text-sm font-medium mb-1">Endpoint</p>
            <code className="inline-block text-xs bg-muted px-3 py-2 rounded w-full break-all">
              {API_BASE}/v1/chat/completions
            </code>
          </div>

          {/* Token selector for docs */}
          {tokens.length > 0 && (
            <div className="flex items-center gap-2">
              <label className="text-xs text-muted-foreground whitespace-nowrap">Example token:</label>
              <select
                className="text-xs border rounded px-2 py-1 bg-background"
                value={docToken}
                onChange={(e) => setDocToken(e.target.value)}
              >
                <option value="<your-token>">&lt;your-token&gt;</option>
                {tokens.map((t) => (
                  <option key={t.id} value={t.prefix + "..."}>
                    {t.name} ({t.prefix}...)
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Examples */}
          <Tabs defaultValue="curl">
            <TabsList>
              <TabsTrigger value="curl">
                <Terminal className="h-3 w-3 mr-1" /> curl
              </TabsTrigger>
              <TabsTrigger value="python">
                <Code className="h-3 w-3 mr-1" /> Python
              </TabsTrigger>
              <TabsTrigger value="openai">
                <Code className="h-3 w-3 mr-1" /> OpenAI SDK
              </TabsTrigger>
            </TabsList>

            <TabsContent value="curl" className="mt-2">
              <pre className="text-xs bg-muted p-3 rounded overflow-x-auto">
{`curl ${API_BASE}/v1/chat/completions \\
  -H "Authorization: Bearer ${docToken}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${exampleModel}",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`}
              </pre>
            </TabsContent>

            <TabsContent value="python" className="mt-2">
              <pre className="text-xs bg-muted p-3 rounded overflow-x-auto">
{`import requests

resp = requests.post(
    "${API_BASE}/v1/chat/completions",
    headers={
        "Authorization": "Bearer ${docToken}",
        "Content-Type": "application/json",
    },
    json={
        "model": "${exampleModel}",
        "messages": [{"role": "user", "content": "Hello!"}],
    },
)
print(resp.json())`}
              </pre>
            </TabsContent>

            <TabsContent value="openai" className="mt-2">
              <pre className="text-xs bg-muted p-3 rounded overflow-x-auto">
{`from openai import OpenAI

client = OpenAI(
    api_key="${docToken}",
    base_url="${API_BASE}/v1",
)

response = client.chat.completions.create(
    model="${exampleModel}",
    messages=[{"role": "user", "content": "Hello!"}],
)
print(response.choices[0].message.content)`}
              </pre>
            </TabsContent>
          </Tabs>

          {/* Supported models */}
          <div>
            <p className="text-sm font-medium mb-1">Supported Models</p>
            {models.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {models.map((m) => (
                  <code key={m.model} className="text-xs bg-muted px-2 py-1 rounded">{m.model}</code>
                ))}
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">No models configured — add them in Settings.</p>
            )}
            <p className="text-xs text-muted-foreground mt-1">
              Model snapshots (e.g. <code>deepseek-chat-2025-01-01</code>) are auto-matched by prefix.
            </p>
          </div>

          {/* Pricing table */}
          {models.length > 0 && (
            <div className="text-xs text-muted-foreground space-y-1">
              <p className="font-medium text-foreground">Pricing (per 1M tokens)</p>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Model</TableHead>
                    <TableHead className="text-right">Input (Cache Miss)</TableHead>
                    <TableHead className="text-right">Input (Cache Hit)</TableHead>
                    <TableHead className="text-right">Output</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {models.map((m) => (
                    <TableRow key={m.model}>
                      <TableCell className="font-mono">{m.model}</TableCell>
                      <TableCell className="text-right">${(m.input_per_million / 10000).toFixed(4)}</TableCell>
                      <TableCell className="text-right">${(m.cache_hit_per_million / 10000).toFixed(4)}</TableCell>
                      <TableCell className="text-right">${(m.output_per_million / 10000).toFixed(4)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
