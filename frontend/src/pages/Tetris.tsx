import { useEffect, useState, useCallback, useRef } from "react";
import {
  fetchTetris,
  getTetrisBudget,
  setTetrisBudget,
  type TetrisPiece,
  type TetrisResponse,
} from "@/lib/api";
import { centsToDisplay, centsToDollars } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Pause, Play, Settings2, Flame, Trophy, Layers, DollarSign } from "lucide-react";

// ---- Color map ----

const MODEL_COLORS: Record<string, string> = {
  "deepseek-v4-flash": "#3b82f6",
  "deepseek-v4-pro": "#8b5cf6",
  "deepseek-chat": "#10b981",
  "deepseek-reasoner": "#f59e0b",
};
const DEFAULT_COLOR = "#6b7280";

function modelColor(model: string): string {
  for (const [key, color] of Object.entries(MODEL_COLORS)) {
    if (model.includes(key)) return color;
  }
  return DEFAULT_COLOR;
}

// ---- Constants ----

const ROWS = 10;
const POLL_MS = 3000;

// ---- Helpers ----

/** Convert dollars input string → hundredths-of-a-cent. */
function dollarsToCents(dollars: string): number {
  const n = parseFloat(dollars);
  if (isNaN(n) || n <= 0) return 0;
  return Math.round(n * 10000);
}

// ---- Component ----

export default function TetrisPage() {
  const [data, setData] = useState<TetrisResponse | null>(null);
  const [budget, setBudget] = useState(0); // hundredths-of-a-cent
  const [budgetInput, setBudgetInput] = useState(""); // dollars string
  const [paused, setPaused] = useState(false);
  const [clearedRows, setClearedRows] = useState(0);
  const [clearingRow, setClearingRow] = useState<number | null>(null);
  const [shake, setShake] = useState(false);
  const [showGameOver, setShowGameOver] = useState(false);
  const prevPieceCount = useRef(0);

  // Fetch budget setting (in hundredths-of-a-cent).
  const loadBudget = useCallback(async () => {
    try {
      const b = await getTetrisBudget();
      setBudget(b.budget);
      // Display as dollars.
      if (b.budget > 0) {
        setBudgetInput(centsToDollars(b.budget).toFixed(2));
      }
    } catch { /* ignore */ }
  }, []);

  // Fetch tetris data.
  const loadData = useCallback(async () => {
    try {
      const d = await fetchTetris();
      setData(d);
      prevPieceCount.current = d.pieces.length;

      // Check for row clears.
      if (d.budget_cents > 0) {
        const rowCap = d.budget_cents / ROWS;
        const totalRows = Math.floor(d.today_cost_cents / rowCap);
        if (totalRows > clearedRows) {
          setClearingRow(totalRows - 1);
          setTimeout(() => setClearingRow(null), 600);
          setClearedRows(totalRows);
        }
      }

      // Game over detection.
      if (d.game_over && !showGameOver) {
        setShake(true);
        setShowGameOver(true);
        setTimeout(() => setShake(false), 800);
      }
    } catch { /* ignore */ }
  }, [clearedRows, showGameOver]);

  // Save budget (input is dollars, store as hundredths-of-a-cent).
  const saveBudget = async () => {
    const cents = dollarsToCents(budgetInput);
    if (cents > 0) {
      await setTetrisBudget(cents);
      setBudget(cents);
    }
  };

  // Initial load.
  useEffect(() => { loadBudget(); }, [loadBudget]);
  useEffect(() => {
    loadData();
    const iv = setInterval(() => {
      if (!paused) loadData();
    }, POLL_MS);
    return () => clearInterval(iv);
  }, [loadData, paused]);

  // Reset game over when new day starts.
  useEffect(() => {
    if (data && !data.game_over && showGameOver) {
      setShowGameOver(false);
      setClearedRows(0);
    }
  }, [data, showGameOver]);

  // ---- Render helpers ----

  const rowCap = budget > 0 ? budget / ROWS : 0;
  const dangerThreshold = 0.8;
  const isDanger = data && budget > 0 && data.percentage >= dangerThreshold * 100;

  // Distribute pieces into rows by cost.
  function buildRows(pieces: TetrisPiece[]): TetrisPiece[][] {
    if (budget <= 0) return [];
    const rows: TetrisPiece[][] = [];
    let currentRow: TetrisPiece[] = [];
    let rowCost = 0;
    for (const p of pieces) {
      if (rowCost + p.cost_cents > rowCap && currentRow.length > 0) {
        rows.push(currentRow);
        currentRow = [];
        rowCost = 0;
      }
      currentRow.push(p);
      rowCost += p.cost_cents;
    }
    if (currentRow.length > 0) rows.push(currentRow);
    while (rows.length < ROWS) rows.unshift([]);
    return rows;
  }

  const rows = data ? buildRows(data.pieces) : [];
  const visibleRows = rows.slice(-ROWS);
  while (visibleRows.length < ROWS) visibleRows.unshift([]);

  function pieceWidth(costCents: number): number {
    if (rowCap <= 0) return 5;
    return Math.max(4, (costCents / rowCap) * 100);
  }

  function rowFillPct(row: TetrisPiece[]): number {
    if (rowCap <= 0) return 0;
    const total = row.reduce((s, p) => s + p.cost_cents, 0);
    return Math.min(100, (total / rowCap) * 100);
  }

  const flatPieces = data?.pieces ?? [];

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold tracking-tight">Token Tetris</h2>
          <p className="text-sm text-muted-foreground">
            Each block is a request — fill rows to clear them. Stay under your daily cost budget!
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setPaused(!paused)}>
            {paused ? <Play className="h-3.5 w-3.5 mr-1" /> : <Pause className="h-3.5 w-3.5 mr-1" />}
            {paused ? "Live" : "Pause"}
          </Button>
        </div>
      </div>

      {/* Budget input (if not set) */}
      {budget === 0 && (
        <div className="bg-card border rounded-lg p-6 text-center space-y-3">
          <DollarSign className="h-8 w-8 mx-auto text-muted-foreground" />
          <p className="text-muted-foreground">Set a daily cost budget to start the game</p>
          <div className="flex items-center justify-center gap-2">
            <span className="text-muted-foreground text-sm">$</span>
            <input
              type="number"
              step="0.01"
              min="0.01"
              value={budgetInput}
              onChange={(e) => setBudgetInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && saveBudget()}
              placeholder="5.00"
              className="w-32 rounded-md border bg-background px-3 py-1.5 text-sm"
            />
            <Button size="sm" onClick={saveBudget}>Set Budget</Button>
          </div>
        </div>
      )}

      {budget > 0 && data && (
        <>
          {/* Stats bar */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
            <Stat icon={<DollarSign />} label="Today" value={centsToDisplay(data.today_cost_cents)} sub="" />
            <Stat icon={<Settings2 />} label="Budget" value={centsToDisplay(budget)} sub={`${data.percentage.toFixed(1)}% used`} />
            <Stat icon={<Flame />} label="Streak" value={String(data.streak)} sub={`best ${data.best_streak}`} />
            <Stat icon={<Layers />} label="Cleared" value={String(clearedRows)} sub={`of ${ROWS} rows`} />
          </div>

          {/* Board */}
          <div
            className={`relative bg-[#0f172a] border-2 rounded-xl overflow-hidden transition-transform ${
              shake ? "animate-shake" : ""
            } ${
              isDanger && !data.game_over ? "border-red-500/50" : data.game_over ? "border-red-600" : "border-slate-700"
            }`}
            style={{ height: Math.min(500, ROWS * 48 + 16) }}
          >
            {/* Danger zone tint */}
            {isDanger && !data.game_over && (
              <div className="absolute inset-0 bg-red-500/5 pointer-events-none z-10" />
            )}

            {/* Game over overlay */}
            {data.game_over && (
              <div className="absolute inset-0 bg-red-950/80 flex flex-col items-center justify-center z-20 rounded-xl">
                <Trophy className="h-10 w-10 text-yellow-400 mb-2" />
                <p className="text-2xl font-black text-red-400 tracking-widest uppercase">Game Over</p>
                <p className="text-sm text-red-300 mt-1">
                  {centsToDisplay(data.today_cost_cents)} / {centsToDisplay(budget)}
                </p>
                <p className="text-xs text-red-400 mt-0.5">
                  Streak: {data.streak} day{data.streak !== 1 ? "s" : ""} · {clearedRows} rows cleared
                </p>
                <p className="text-xs text-muted-foreground mt-3">Resets at midnight</p>
              </div>
            )}

            {/* Row grid */}
            <div className="flex flex-col h-full p-1.5 gap-1">
              {visibleRows.map((row, ri) => {
                const fill = rowFillPct(row);
                const isClearing = clearingRow === visibleRows.length - 1 - ri;
                const isFull = fill >= 100;

                return (
                  <div
                    key={ri}
                    className={`flex-1 flex gap-0.5 rounded-sm transition-all duration-300 relative ${
                      isClearing
                        ? "bg-yellow-400/30 scale-105"
                        : isFull
                          ? "bg-emerald-500/10"
                          : "bg-slate-800/30"
                    }`}
                  >
                    {/* Fill bar background */}
                    <div
                      className="absolute inset-0 rounded-sm transition-all duration-500"
                      style={{
                        width: `${fill}%`,
                        background: isFull
                          ? "linear-gradient(90deg, rgba(16,185,129,0.15), rgba(16,185,129,0.05))"
                          : "",
                      }}
                    />

                    {/* Pieces in this row */}
                    <div className="relative flex items-center w-full px-0.5 gap-0.5">
                      {row.map((p, pi) => (
                        <div
                          key={pi}
                          className="h-[70%] rounded-sm animate-pop-in flex items-center justify-center text-[8px] font-bold text-white/80 transition-all hover:scale-105 hover:brightness-125 cursor-default"
                          style={{
                            width: `${pieceWidth(p.cost_cents)}%`,
                            backgroundColor: modelColor(p.model),
                            minWidth: "8px",
                            animationDelay: `${pi * 30}ms`,
                          }}
                          title={`${p.time} · ${centsToDisplay(p.cost_cents)} · ${p.tokens.toLocaleString()} tok · ${p.model}${p.cache_hit ? " · cache hit" : ""}`}
                        >
                          {p.cost_cents >= rowCap * 0.25
                            ? `$${centsToDollars(p.cost_cents).toFixed(4).replace(/\.?0+$/, "")}`
                            : ""}
                        </div>
                      ))}
                    </div>

                    {/* Row clearing flash */}
                    {isClearing && (
                      <div className="absolute inset-0 bg-yellow-300/50 rounded-sm animate-flash-out" />
                    )}
                  </div>
                );
              })}
            </div>
          </div>

          {/* Legend */}
          <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
            {Object.entries(MODEL_COLORS).map(([model, color]) => (
              <span key={model} className="flex items-center gap-1.5">
                <span className="w-3 h-3 rounded-sm" style={{ backgroundColor: color }} />
                {model}
              </span>
            ))}
            <span className="flex items-center gap-1.5">
              <span className="w-3 h-3 rounded-sm border border-dashed border-emerald-500/50" />
              cache hit
            </span>
          </div>

          {/* Budget adjustment */}
          <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">$</span>
            <input
              type="number"
              step="0.01"
              min="0.01"
              value={budgetInput}
              onChange={(e) => setBudgetInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && saveBudget()}
              placeholder="5.00"
              className="w-28 rounded-md border bg-background px-3 py-1.5 text-sm"
            />
            <Button variant="outline" size="sm" onClick={saveBudget}>Update Budget</Button>
          </div>

          {/* Recent pieces feed */}
          <details className="text-sm">
            <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
              Today's requests ({flatPieces.length})
            </summary>
            <div className="mt-2 max-h-48 overflow-auto space-y-0.5">
              {flatPieces.slice().reverse().map((p, i) => (
                <div key={i} className="flex items-center gap-2 text-xs font-mono text-muted-foreground">
                  <span>{p.time}</span>
                  <span
                    className="w-2.5 h-2.5 rounded-sm shrink-0"
                    style={{ backgroundColor: modelColor(p.model) }}
                  />
                  <span>{centsToDisplay(p.cost_cents)}</span>
                  <span>{p.tokens.toLocaleString()} tok</span>
                  <span className="text-muted-foreground/50">{p.model}</span>
                  {p.cache_hit && <span className="text-emerald-500">⚡</span>}
                </div>
              ))}
            </div>
          </details>
        </>
      )}
    </div>
  );
}

// ---- Stat card ----

function Stat({
  icon,
  label,
  value,
  sub,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub: string;
}) {
  return (
    <div className="bg-card border rounded-lg p-3 flex items-center gap-3">
      <div className="text-muted-foreground shrink-0">{icon}</div>
      <div className="min-w-0">
        <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{label}</p>
        <p className="text-lg font-bold tabular-nums truncate">{value}</p>
        <p className="text-[10px] text-muted-foreground">{sub}</p>
      </div>
    </div>
  );
}
