import { useEffect, useState } from "react";
import { fetchDeepSeekKey, setDeepSeekKey, fetchModels, createModel, saveModel, deleteModel, changePassword, type PricingRow } from "@/lib/api";
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
import { Eye, EyeOff, Save, Plus, Trash2, Edit3, X, Check } from "lucide-react";

export default function Settings() {
  const [key, setKey] = useState("");
  const [masked, setMasked] = useState("");
  const [show, setShow] = useState(false);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  // Password change
  const [currentPw, setCurrentPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [confirmPw, setConfirmPw] = useState("");
  const [pwMsg, setPwMsg] = useState("");
  const [pwSaving, setPwSaving] = useState(false);

  const [models, setModels] = useState<PricingRow[]>([]);
  const [editing, setEditing] = useState<string | null>(null);
  const [newModel, setNewModel] = useState({ model: "", inputMiss: "", inputHit: "", output: "" });
  const [editValues, setEditValues] = useState({ inputMiss: "", inputHit: "", output: "" });
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    fetchDeepSeekKey().then((res) => {
      if (res.masked) setMasked(res.masked);
    }).catch(console.error);
    loadModels();
  }, []);

  function loadModels() {
    fetchModels().then(setModels).catch(console.error);
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault();
    if (newPw !== confirmPw) {
      setPwMsg("Passwords do not match.");
      return;
    }
    if (newPw.length < 8) {
      setPwMsg("New password must be at least 8 characters.");
      return;
    }
    setPwSaving(true);
    setPwMsg("");
    try {
      await changePassword(currentPw, newPw);
      setCurrentPw("");
      setNewPw("");
      setConfirmPw("");
      setPwMsg("Password updated.");
    } catch {
      setPwMsg("Failed — check your current password.");
    } finally {
      setPwSaving(false);
    }
  }

  async function handleSaveKey(e: React.FormEvent) {
    e.preventDefault();
    if (!key.trim()) return;
    setSaving(true);
    setMsg("");
    try {
      const res = await setDeepSeekKey(key.trim());
      setMasked(res.masked);
      setMsg("API key saved.");
    } catch {
      setMsg("Failed to save.");
    } finally {
      setSaving(false);
    }
  }

  async function handleAddModel(e: React.FormEvent) {
    e.preventDefault();
    if (!newModel.model.trim()) return;
    try {
      await createModel(
        newModel.model.trim(),
        toCents(newModel.inputMiss),
        toCents(newModel.output),
        toCents(newModel.inputHit),
      );
      setNewModel({ model: "", inputMiss: "", inputHit: "", output: "" });
      setAdding(false);
      loadModels();
    } catch { /* ignore */ }
  }

  async function handleUpdateModel(model: string) {
    try {
      await saveModel(model, toCents(editValues.inputMiss), toCents(editValues.output), toCents(editValues.inputHit));
      setEditing(null);
      loadModels();
    } catch { /* ignore */ }
  }

  async function handleDeleteModel(model: string) {
    try {
      await deleteModel(model);
      loadModels();
    } catch { /* ignore */ }
  }

  function startEdit(m: PricingRow) {
    setEditing(m.model);
    setEditValues({
      inputMiss: toDollars(m.input_per_million),
      output: toDollars(m.output_per_million),
      inputHit: toDollars(m.cache_hit_per_million),
    });
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Settings</h2>
        <p className="text-sm text-muted-foreground">Configure API key and model pricing.</p>
      </div>

      {/* DeepSeek API Key */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">DeepSeek API Key</CardTitle>
          <CardDescription>
            Your API key is stored in the local database and never leaves your server.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSaveKey} className="space-y-4">
            <div>
              <label className="text-sm font-medium" htmlFor="apikey">API Key</label>
              <div className="flex gap-2 mt-1">
                <input
                  id="apikey"
                  type={show ? "text" : "password"}
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  placeholder="sk-..."
                  className="flex-1 px-3 py-2 border rounded-md text-sm bg-background font-mono"
                />
                <Button type="button" variant="outline" size="icon" onClick={() => setShow(!show)}>
                  {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </Button>
              </div>
              {masked && !key && (
                <p className="text-xs text-muted-foreground mt-1">Current: {masked}</p>
              )}
            </div>
            <div className="flex items-center gap-3">
              <Button type="submit" disabled={saving || !key.trim()}>
                <Save className="h-4 w-4 mr-1" />
                {saving ? "Saving..." : "Save"}
              </Button>
              {msg && <p className={`text-sm ${msg.includes("Failed") ? "text-red-500" : "text-green-600"}`}>{msg}</p>}
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Change Password */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Change Password</CardTitle>
          <CardDescription>Update your admin account password.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleChangePassword} className="space-y-3">
            <div>
              <label className="text-xs font-medium" htmlFor="currentPw">Current Password</label>
              <input
                id="currentPw"
                type="password"
                value={currentPw}
                onChange={(e) => setCurrentPw(e.target.value)}
                className="w-full mt-1 px-3 py-1.5 border rounded text-sm bg-background"
                required
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-medium" htmlFor="newPw">New Password</label>
                <input
                  id="newPw"
                  type="password"
                  value={newPw}
                  onChange={(e) => setNewPw(e.target.value)}
                  className="w-full mt-1 px-3 py-1.5 border rounded text-sm bg-background"
                  required
                />
              </div>
              <div>
                <label className="text-xs font-medium" htmlFor="confirmPw">Confirm New Password</label>
                <input
                  id="confirmPw"
                  type="password"
                  value={confirmPw}
                  onChange={(e) => setConfirmPw(e.target.value)}
                  className="w-full mt-1 px-3 py-1.5 border rounded text-sm bg-background"
                  required
                />
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Button type="submit" disabled={pwSaving}>
                <Save className="h-4 w-4 mr-1" />
                {pwSaving ? "Updating..." : "Update Password"}
              </Button>
              {pwMsg && (
                <p className={`text-sm ${pwMsg.includes("Failed") || pwMsg.includes("do not") ? "text-red-500" : "text-green-600"}`}>
                  {pwMsg}
                </p>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Model Pricing */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-sm">Model Pricing</CardTitle>
            <CardDescription>
              Cost per 1M tokens in USD. Cache-miss is the full input price; cache-hit is the discounted price.
            </CardDescription>
          </div>
          <Button variant="outline" size="sm" onClick={() => setAdding(!adding)}>
            <Plus className="h-4 w-4 mr-1" /> Add Model
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Add form */}
          {adding && (
            <form onSubmit={handleAddModel} className="p-4 border rounded-md space-y-3 bg-muted/30">
              <div>
                <label className="text-xs font-medium">Model Name</label>
                <input
                  placeholder="deepseek-chat"
                  value={newModel.model}
                  onChange={(e) => setNewModel({ ...newModel, model: e.target.value })}
                  className="w-full mt-1 px-2 py-1.5 border rounded text-sm bg-background font-mono"
                  required
                />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="text-xs font-medium">1M Input (Cache Miss)</label>
                  <input
                    type="number" step="any"
                    placeholder="0.14"
                    value={newModel.inputMiss}
                    onChange={(e) => setNewModel({ ...newModel, inputMiss: e.target.value })}
                    className="w-full mt-1 px-2 py-1.5 border rounded text-sm bg-background"
                  />
                </div>
                <div>
                  <label className="text-xs font-medium">1M Input (Cache Hit)</label>
                  <input
                    type="number" step="any"
                    placeholder="0.014"
                    value={newModel.inputHit}
                    onChange={(e) => setNewModel({ ...newModel, inputHit: e.target.value })}
                    className="w-full mt-1 px-2 py-1.5 border rounded text-sm bg-background"
                  />
                </div>
                <div>
                  <label className="text-xs font-medium">1M Output</label>
                  <input
                    type="number" step="any"
                    placeholder="0.28"
                    value={newModel.output}
                    onChange={(e) => setNewModel({ ...newModel, output: e.target.value })}
                    className="w-full mt-1 px-2 py-1.5 border rounded text-sm bg-background"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <Button type="submit" size="sm"><Check className="h-3 w-3 mr-1" /> Save</Button>
                <Button type="button" variant="ghost" size="sm" onClick={() => setAdding(false)}>Cancel</Button>
              </div>
            </form>
          )}

          {/* Table */}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-40">Model</TableHead>
                <TableHead className="text-right">1M Input (Cache Miss)</TableHead>
                <TableHead className="text-right">1M Input (Cache Hit)</TableHead>
                <TableHead className="text-right">1M Output</TableHead>
                <TableHead className="w-20"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {models.map((m) =>
                editing === m.model ? (
                  <TableRow key={m.model}>
                    <TableCell className="font-mono text-xs">{m.model}</TableCell>
                    <TableCell className="text-right">
                      <input
                        type="number" step="any"
                        value={editValues.inputMiss}
                        onChange={(e) => setEditValues({ ...editValues, inputMiss: e.target.value })}
                        className="w-24 px-2 py-1 border rounded text-xs text-right bg-background"
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <input
                        type="number" step="any"
                        value={editValues.inputHit}
                        onChange={(e) => setEditValues({ ...editValues, inputHit: e.target.value })}
                        className="w-24 px-2 py-1 border rounded text-xs text-right bg-background"
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <input
                        type="number" step="any"
                        value={editValues.output}
                        onChange={(e) => setEditValues({ ...editValues, output: e.target.value })}
                        className="w-24 px-2 py-1 border rounded text-xs text-right bg-background"
                      />
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="icon" onClick={() => handleUpdateModel(m.model)}>
                          <Check className="h-3 w-3 text-green-600" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => setEditing(null)}>
                          <X className="h-3 w-3" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  <TableRow key={m.model}>
                    <TableCell className="font-mono text-xs">{m.model}</TableCell>
                    <TableCell className="text-right text-xs tabular-nums">${(m.input_per_million / 10000).toFixed(4)}</TableCell>
                    <TableCell className="text-right text-xs tabular-nums">${(m.cache_hit_per_million / 10000).toFixed(4)}</TableCell>
                    <TableCell className="text-right text-xs tabular-nums">${(m.output_per_million / 10000).toFixed(4)}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="icon" onClick={() => startEdit(m)}>
                          <Edit3 className="h-3 w-3" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => handleDeleteModel(m.model)}>
                          <Trash2 className="h-3 w-3 text-red-500" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              )}
              {models.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-xs text-muted-foreground py-4">
                    No models configured. Defaults are seeded on first run.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

function toCents(dollars: string): number {
  return Math.round(parseFloat(dollars || "0") * 10000);
}

function toDollars(cents: number): string {
  // Strip trailing zeros but keep at least 2 decimal places.
  let s = (cents / 10000).toFixed(6);
  s = s.replace(/0+$/, "").replace(/\.$/, "");
  if (!s.includes(".")) s += ".00";
  const parts = s.split(".");
  if (parts[1] && parts[1].length < 2) s += "0";
  return s;
}
