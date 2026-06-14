import { useEffect, useState } from "react";
import {
  fetchSummary,
  fetchDaily,
  fetchTopDays,
  type SummaryResponse,
  type DailyPoint,
  type TopDaysResponse,
  type ModelDailyPoint,
} from "@/lib/api";
import { centsToDisplay, centsToDollars, formatPercent, formatNumber } from "@/lib/utils";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { DollarSign, Zap, Activity, Percent, TrendingUp, TrendingDown, Database } from "lucide-react";

const COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#8b5cf6", "#ec4899", "#06b6d4"];

export default function Dashboard() {
  const [summary, setSummary] = useState<SummaryResponse | null>(null);
  const [daily, setDaily] = useState<DailyPoint[]>([]);
  const [topDays, setTopDays] = useState<TopDaysResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      fetchSummary(),
      fetchDaily(),
      fetchTopDays(5),
    ]).then(([s, d, t]) => {
      setSummary(s);
      setDaily(d);
      setTopDays(t);
      setLoading(false);
    }).catch(console.error);
  }, []);

  if (loading) {
    return <div className="flex items-center justify-center h-64 text-muted-foreground">Loading...</div>;
  }
  if (!summary) {
    return <div className="flex items-center justify-center h-64 text-muted-foreground">No data yet — send some requests through the proxy.</div>;
  }

  // Build daily chart data with per-model breakdown
  const chartData = daily.map((d) => {
    const point: Record<string, string | number> = { date: d.date.slice(5) }; // MM-DD
    let others = d.cost_cents;
    (d.by_model ?? []).slice(0, 5).forEach((m: ModelDailyPoint) => {
      point[m.model] = m.cost_cents;
      others -= m.cost_cents;
    });
    if (others > 0) point["other"] = others;
    return point;
  });

  // Collect unique model names for chart legend
  const modelNames = new Set<string>();
  daily.forEach((d) => (d.by_model ?? []).forEach((m) => modelNames.add(m.model)));
  const chartModels = [...modelNames].slice(0, 5);

  const summaryCards = [
    { label: "Total Cost", value: summary.total_cost_display, icon: DollarSign, sub: `${formatNumber(summary.total_requests)} requests` },
    { label: "Avg Daily Spend", value: summary.avg_daily_cost_display, icon: TrendingUp, sub: summary.avg_daily_cost_cents > 0 ? "per active day" : "—" },
    { label: "Total Tokens", value: formatNumber(summary.total_tokens), icon: Zap, sub: `${formatNumber(summary.total_prompt_tokens)} in / ${formatNumber(summary.total_completion_tokens)} out` },
    { label: "Cache Hit Rate", value: formatPercent(summary.cache_hit_rate), icon: Percent, sub: `${formatNumber(summary.cache_hit_tokens)} hit / ${formatNumber(summary.cache_miss_tokens)} miss` },
    { label: "Cache Savings", value: summary.cache_savings_display, icon: Database, sub: "vs all cache-miss" },
    { label: "Output Cost", value: summary.output_cost_display, icon: Activity, sub: "completion tokens" },
    { label: "Avg $/1M Input", value: `$${summary.avg_dollars_per_1m_input.toFixed(2)}`, icon: TrendingDown, sub: "effective input rate" },
    { label: "Avg $/1M Output", value: `$${summary.avg_dollars_per_1m_output.toFixed(2)}`, icon: TrendingUp, sub: "effective output rate" },
  ];

  return (
    <div className="space-y-6">
      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {summaryCards.map((card) => (
          <Card key={card.label}>
            <CardHeader className="pb-2">
              <CardDescription className="flex items-center gap-1 text-xs">
                <card.icon className="h-3 w-3" />
                {card.label}
              </CardDescription>
              <CardTitle className="text-xl">{card.value}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">{card.sub}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Cache cost breakdown */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader><CardTitle className="text-sm">Cache Hit Cost</CardTitle></CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{centsToDisplay(summary.cache_hit_cost_cents)}</p>
            <p className="text-xs text-muted-foreground">{formatNumber(summary.cache_hit_tokens)} tokens @ cheap rate</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm">Cache Miss Cost</CardTitle></CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{centsToDisplay(summary.cache_miss_cost_cents)}</p>
            <p className="text-xs text-muted-foreground">{formatNumber(summary.cache_miss_tokens)} tokens @ full rate</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm">Output Cost</CardTitle></CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{summary.output_cost_display}</p>
            <p className="text-xs text-muted-foreground">{formatNumber(summary.total_completion_tokens)} tokens generated</p>
          </CardContent>
        </Card>
      </div>

      {/* Daily cost by model chart */}
      <Card>
        <CardHeader>
          <CardTitle>Daily Cost by Model</CardTitle>
          <CardDescription>Stacked cost breakdown per day</CardDescription>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={350}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} tickFormatter={(v) => `$${centsToDollars(v).toFixed(2)}`} />
              <Tooltip
                contentStyle={{
                  backgroundColor: "hsl(var(--card))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: "0.5rem",
                  color: "hsl(var(--card-foreground))",
                  fontSize: "0.75rem",
                }}
                formatter={(value) => centsToDisplay(Number(value) || 0)}
                labelFormatter={(label) => `Day ${label}`}
              />
              <Legend />
              {chartModels.map((model, i) => (
                <Bar key={model} dataKey={model} stackId="a" fill={COLORS[i % COLORS.length]} />
              ))}
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Top / Bottom days */}
      {topDays && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <DayTable title="🔥 Most Expensive Days" days={topDays.most_expensive} metric="cost" />
          <DayTable title="💸 Least Expensive Days" days={topDays.least_expensive} metric="cost" />
          <DayTable title="❌ Most Cache Miss Days" days={topDays.most_cache_miss} metric="cache" />
          <DayTable title="✅ Best Cache Hit Rate Days" days={topDays.best_cache_hit_rate} metric="cache" />
        </div>
      )}
    </div>
  );
}

function DayTable({ title, days, metric }: { title: string; days: TopDaysResponse["most_expensive"]; metric: "cost" | "cache" }) {
  if (!days || !days.length) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Date</TableHead>
              <TableHead className="text-right">{metric === "cost" ? "Cost" : "Cache Hit Rate"}</TableHead>
              <TableHead className="text-right">Requests</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {days.map((d) => (
              <TableRow key={d.date}>
                <TableCell>{d.date}</TableCell>
                <TableCell className="text-right">
                  {metric === "cost" ? d.cost_display : formatPercent(d.cache_hit_rate)}
                </TableCell>
                <TableCell className="text-right">{d.requests}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
