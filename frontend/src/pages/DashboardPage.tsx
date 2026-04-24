import { useQuery } from "@tanstack/react-query";
import { getDashboardSummary } from "@/api/dashboard";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { describeApiError } from "@/lib/http";

function MetricCard({ title, value, percent }: { title: string; value: string; percent?: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <p className="text-2xl font-semibold text-slate-900">{value}</p>
          {percent !== undefined && (
            <div className="h-2 overflow-hidden rounded-full bg-slate-200">
              <div
                className="h-full rounded-full bg-panel-600 transition-all"
                style={{ width: `${Math.max(0, Math.min(100, percent))}%` }}
              />
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export function DashboardPage() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["dashboard", "summary"],
    queryFn: getDashboardSummary
  });

  if (isLoading) {
    return <p className="text-slate-600">Loading dashboard summary...</p>;
  }

  if (isError) {
    const dashboardError = describeApiError(error, "Failed to load dashboard.");
    return (
      <QueryErrorCard
        title="Failed to load dashboard"
        message={dashboardError.message}
        hint={dashboardError.hint}
        onRetry={() => refetch()}
      />
    );
  }

  if (!data) {
    return <p className="text-slate-600">No dashboard data available.</p>;
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Dashboard</h2>
        <p className="text-sm text-slate-500">Host: {data.hostname || "unknown"}</p>
      </div>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <MetricCard title="System Version" value={data.system_version || "unknown"} />
        <MetricCard title="Kernel Version" value={data.kernel_version || "unknown"} />
        <MetricCard title="Uptime" value={data.uptime || "unknown"} />
        <MetricCard title="CPU Usage" value={`${data.cpu_usage.toFixed(2)}%`} percent={data.cpu_usage} />
        <MetricCard title="Memory Usage" value={`${data.memory_usage.toFixed(2)}%`} percent={data.memory_usage} />
        <MetricCard title="Disk Usage" value={`${data.disk_usage.toFixed(2)}%`} percent={data.disk_usage} />
      </div>
    </div>
  );
}
