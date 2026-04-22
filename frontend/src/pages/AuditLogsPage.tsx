import { FormEvent, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { listAuditLogs } from "@/api/audit";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export function AuditLogsPage() {
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [moduleFilterInput, setModuleFilterInput] = useState("");
  const [actionFilterInput, setActionFilterInput] = useState("");
  const [moduleFilter, setModuleFilter] = useState("");
  const [actionFilter, setActionFilter] = useState("");

  const logsQuery = useQuery({
    queryKey: ["audit", "logs", page, size, moduleFilter, actionFilter],
    queryFn: () =>
      listAuditLogs({
        page,
        size,
        module: moduleFilter || undefined,
        action: actionFilter || undefined
      })
  });

  function submitFilter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPage(1);
    setModuleFilter(moduleFilterInput.trim());
    setActionFilter(actionFilterInput.trim());
  }

  const total = logsQuery.data?.total ?? 0;
  const maxPage = Math.max(1, Math.ceil(total / size));

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Audit Logs</h2>
        <p className="text-sm text-slate-500">Track key operations and login behavior.</p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <form className="grid gap-2 md:grid-cols-[1fr_1fr_auto]" onSubmit={submitFilter}>
            <Input
              onChange={(event) => setModuleFilterInput(event.target.value)}
              placeholder="module filter"
              value={moduleFilterInput}
            />
            <Input
              onChange={(event) => setActionFilterInput(event.target.value)}
              placeholder="action filter"
              value={actionFilterInput}
            />
            <Button type="submit">Filter</Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Logs</CardTitle>
        </CardHeader>
        <CardContent>
          {logsQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading logs...</p>
          ) : logsQuery.isError ? (
            <p className="text-sm text-rose-600">
              {logsQuery.error instanceof Error ? logsQuery.error.message : "Failed to load logs"}
            </p>
          ) : (
            <div className="space-y-3">
              <div className="overflow-hidden rounded-lg border border-slate-200">
                <table className="w-full text-left text-sm">
                  <thead className="bg-slate-50 text-slate-600">
                    <tr>
                      <th className="px-4 py-3">Time</th>
                      <th className="px-4 py-3">User</th>
                      <th className="px-4 py-3">IP</th>
                      <th className="px-4 py-3">Module</th>
                      <th className="px-4 py-3">Action</th>
                      <th className="px-4 py-3">Target</th>
                      <th className="px-4 py-3">Result</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(logsQuery.data?.items || []).map((item) => (
                      <tr className="border-t border-slate-200" key={item.id}>
                        <td className="px-4 py-3">{new Date(item.created_at).toLocaleString()}</td>
                        <td className="px-4 py-3">{item.username || "-"}</td>
                        <td className="px-4 py-3">{item.ip || "-"}</td>
                        <td className="px-4 py-3">{item.module}</td>
                        <td className="px-4 py-3">{item.action}</td>
                        <td className="px-4 py-3">{item.target_type}:{item.target_id}</td>
                        <td className="px-4 py-3">{item.success ? "success" : "failed"}</td>
                      </tr>
                    ))}
                    {(logsQuery.data?.items || []).length === 0 && (
                      <tr>
                        <td className="px-4 py-8 text-center text-slate-500" colSpan={7}>
                          No audit logs found.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>

              <div className="flex items-center justify-between">
                <p className="text-sm text-slate-600">
                  Total {total} records, page {page} / {maxPage}
                </p>
                <div className="flex gap-2">
                  <Button
                    disabled={page <= 1}
                    onClick={() => setPage((prev) => Math.max(1, prev - 1))}
                    size="sm"
                    variant="ghost"
                  >
                    Prev
                  </Button>
                  <Button
                    disabled={page >= maxPage}
                    onClick={() => setPage((prev) => Math.min(maxPage, prev + 1))}
                    size="sm"
                    variant="ghost"
                  >
                    Next
                  </Button>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
