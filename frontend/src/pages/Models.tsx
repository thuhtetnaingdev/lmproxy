import { useEffect, useState } from "react";
import { fetchByModel, type ModelBreakdown } from "@/lib/api";
import { formatNumber, formatPercent } from "@/lib/utils";
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

export default function Models() {
  const [models, setModels] = useState<ModelBreakdown[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchByModel().then(setModels).catch(console.error).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="flex items-center justify-center h-64 text-muted-foreground">Loading...</div>;
  if (!models.length) return <div className="flex items-center justify-center h-64 text-muted-foreground">No data yet.</div>;

  const totalCost = models.reduce((s, m) => s + m.cost_cents, 0);

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader><CardTitle className="text-sm">Models Used</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold">{models.length}</p></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm">Total Cost</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold">{(totalCost / 10000).toLocaleString("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 4 })}</p></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm">Total Requests</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold">{formatNumber(models.reduce((s, m) => s + m.requests, 0))}</p></CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Model Breakdown</CardTitle>
          <CardDescription>Per-model token usage and cost</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Model</TableHead>
                <TableHead className="text-right">Requests</TableHead>
                <TableHead className="text-right">Input Tokens</TableHead>
                <TableHead className="text-right">Output Tokens</TableHead>
                <TableHead className="text-right">Cache Hit Rate</TableHead>
                <TableHead className="text-right">Cost</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {models.map((m) => (
                <TableRow key={m.model}>
                  <TableCell className="font-mono text-sm">{m.model}</TableCell>
                  <TableCell className="text-right">{formatNumber(m.requests)}</TableCell>
                  <TableCell className="text-right">{formatNumber(m.prompt_tokens)}</TableCell>
                  <TableCell className="text-right">{formatNumber(m.completion_tokens)}</TableCell>
                  <TableCell className="text-right">{formatPercent(m.cache_hit_rate)}</TableCell>
                  <TableCell className="text-right font-semibold">{m.cost_display}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
