import { useEffect, useState } from "react";
import { fetchRecent, type RequestRecord } from "@/lib/api";
import { centsToDisplay, formatNumber } from "@/lib/utils";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { RefreshCw, ChevronLeft, ChevronRight } from "lucide-react";

const PER_PAGE = 25;

export default function Requests() {
  const [data, setData] = useState<RequestRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);

  function load(p: number) {
    setLoading(true);
    fetchRecent(p, PER_PAGE)
      .then((res) => {
        setData(res.data);
        setTotal(res.total);
        setPage(res.page);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }

  useEffect(() => { load(1); }, []);

  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Recent Requests</h2>
          <p className="text-sm text-muted-foreground">
            {total > 0
              ? `Showing ${(page - 1) * PER_PAGE + 1}–${Math.min(page * PER_PAGE, total)} of ${formatNumber(total)} requests`
              : "No requests yet"}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => load(page)} disabled={loading}>
          <RefreshCw className={`h-4 w-4 mr-1 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">ID</TableHead>
                <TableHead>Time</TableHead>
                <TableHead>Model</TableHead>
                <TableHead className="text-right">Input</TableHead>
                <TableHead className="text-right">Output</TableHead>
                <TableHead className="text-right">Cache Hit</TableHead>
                <TableHead className="text-right">Cache Miss</TableHead>
                <TableHead className="text-right">Cost</TableHead>
                <TableHead className="text-right">Latency</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.length === 0 && !loading && (
                <TableRow>
                  <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                    No requests yet — send traffic through the proxy.
                  </TableCell>
                </TableRow>
              )}
              {data.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="text-xs text-muted-foreground">{r.id}</TableCell>
                  <TableCell className="text-xs whitespace-nowrap">
                    {new Date(r.requested_at).toLocaleString()}
                  </TableCell>
                  <TableCell className="font-mono text-xs">{r.model}</TableCell>
                  <TableCell className="text-right text-xs">{formatNumber(r.prompt_tokens)}</TableCell>
                  <TableCell className="text-right text-xs">{formatNumber(r.completion_tokens)}</TableCell>
                  <TableCell className="text-right text-xs text-green-600">{formatNumber(r.cache_hit_tokens)}</TableCell>
                  <TableCell className="text-right text-xs text-amber-600">{formatNumber(r.cache_miss_tokens)}</TableCell>
                  <TableCell className="text-right text-xs font-semibold">{centsToDisplay(r.cost_cents)}</TableCell>
                  <TableCell className="text-right text-xs text-muted-foreground">{r.duration_ms}ms</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <p className="text-xs text-muted-foreground">
          {total > 0
            ? `Page ${page} of ${totalPages}  ·  ${formatNumber(total)} total`
            : "No requests"}
        </p>
        <div className="flex gap-1">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => load(page - 1)}
          >
            <ChevronLeft className="h-4 w-4 mr-1" /> Prev
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => load(page + 1)}
          >
            Next <ChevronRight className="h-4 w-4 ml-1" />
          </Button>
        </div>
      </div>
    </div>
  );
}
