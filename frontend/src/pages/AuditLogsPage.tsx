import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  AuditExportFormat,
  AuditExportParams,
  exportAuditLogs,
  listAuditLogs
} from "@/api/audit";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
import { AuditLog } from "@/types/audit";

type SuccessFilter = "all" | "true" | "false";

interface AuditFilterInputs {
  startTime: string;
  endTime: string;
  userID: string;
  username: string;
  module: string;
  action: string;
  targetType: string;
  targetID: string;
  success: SuccessFilter;
  resultCode: string;
  requestID: string;
  traceID: string;
}

const emptyFilters: AuditFilterInputs = {
  startTime: "",
  endTime: "",
  userID: "",
  username: "",
  module: "",
  action: "",
  targetType: "",
  targetID: "",
  success: "all",
  resultCode: "",
  requestID: "",
  traceID: ""
};

function buildAuditParams(filters: AuditFilterInputs): AuditExportParams {
  const params: AuditExportParams = {};
  const userID = Number(filters.userID.trim());

  if (filters.startTime) {
    params.start_time = filters.startTime;
  }
  if (filters.endTime) {
    params.end_time = filters.endTime;
  }
  if (Number.isFinite(userID) && userID > 0) {
    params.user_id = userID;
  }
  if (filters.username.trim()) {
    params.username = filters.username.trim();
  }
  if (filters.module.trim()) {
    params.module = filters.module.trim();
  }
  if (filters.action.trim()) {
    params.action = filters.action.trim();
  }
  if (filters.targetType.trim()) {
    params.target_type = filters.targetType.trim();
  }
  if (filters.targetID.trim()) {
    params.target_id = filters.targetID.trim();
  }
  if (filters.success !== "all") {
    params.success = filters.success === "true";
  }
  if (filters.resultCode.trim()) {
    params.result_code = filters.resultCode.trim();
  }
  if (filters.requestID.trim()) {
    params.request_id = filters.requestID.trim();
  }
  if (filters.traceID.trim()) {
    params.trace_id = filters.traceID.trim();
  }

  return params;
}

function taskIDFor(log: AuditLog) {
  if ((log.target_type === "task" || log.target_type === "tasks") && /^\d+$/.test(log.target_id)) {
    return log.target_id;
  }
  return "";
}

function targetLabel(log: AuditLog) {
  if (!log.target_type && !log.target_id) {
    return "-";
  }
  if (!log.target_id) {
    return log.target_type;
  }
  return `${log.target_type}:${log.target_id}`;
}

function downloadAuditBlob(blob: Blob, format: AuditExportFormat) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  anchor.href = url;
  anchor.download = `snowpanel-audit-${stamp}.${format}`;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

function DetailLine({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase text-slate-500">{label}</dt>
      <dd className="mt-1 break-words text-sm text-slate-800">{value || "-"}</dd>
    </div>
  );
}

export function AuditLogsPage() {
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [filterInputs, setFilterInputs] = useState<AuditFilterInputs>(emptyFilters);
  const [appliedFilters, setAppliedFilters] = useState<AuditFilterInputs>(emptyFilters);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [exportingFormat, setExportingFormat] = useState<AuditExportFormat | null>(null);
  const [feedback, setFeedback] = useState("");
  const queryParams = buildAuditParams(appliedFilters);

  const logsQuery = useQuery({
    queryKey: ["audit", "logs", page, size, queryParams],
    queryFn: () =>
      listAuditLogs({
        page,
        size,
        ...queryParams
      })
  });
  const logsLoadError = logsQuery.isError
    ? describeApiError(logsQuery.error, "Failed to load logs.")
    : null;

  function updateFilter<Key extends keyof AuditFilterInputs>(key: Key, value: AuditFilterInputs[Key]) {
    setFilterInputs((prev) => ({
      ...prev,
      [key]: value
    }));
  }

  function submitFilter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPage(1);
    setSelectedLog(null);
    setAppliedFilters({ ...filterInputs });
  }

  function clearFilter() {
    setPage(1);
    setSelectedLog(null);
    setFilterInputs(emptyFilters);
    setAppliedFilters(emptyFilters);
  }

  async function copyRequestID(requestID: string) {
    if (!requestID) {
      return;
    }
    try {
      await navigator.clipboard.writeText(requestID);
      setFeedback("Request ID copied.");
    } catch {
      setFeedback("Unable to copy request ID in this browser context.");
    }
  }

  async function handleExport(format: AuditExportFormat) {
    try {
      setFeedback("");
      setExportingFormat(format);
      const blob = await exportAuditLogs(queryParams, format);
      downloadAuditBlob(blob, format);
      setFeedback(`Audit ${format.toUpperCase()} export started.`);
    } catch (error) {
      const display = describeApiError(error, "Failed to export audit logs.");
      setFeedback(display.message);
    } finally {
      setExportingFormat(null);
    }
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
        <CardContent className="space-y-4 pt-6">
          <form className="space-y-3" onSubmit={submitFilter}>
            <div className="grid gap-2 md:grid-cols-3 xl:grid-cols-4">
              <Input
                onChange={(event) => updateFilter("startTime", event.target.value)}
                type="date"
                value={filterInputs.startTime}
              />
              <Input
                onChange={(event) => updateFilter("endTime", event.target.value)}
                type="date"
                value={filterInputs.endTime}
              />
              <Input
                onChange={(event) => updateFilter("username", event.target.value)}
                placeholder="username"
                value={filterInputs.username}
              />
              <Input
                onChange={(event) => updateFilter("userID", event.target.value)}
                placeholder="user id"
                type="number"
                value={filterInputs.userID}
              />
              <Input
                onChange={(event) => updateFilter("module", event.target.value)}
                placeholder="module"
                value={filterInputs.module}
              />
              <Input
                onChange={(event) => updateFilter("action", event.target.value)}
                placeholder="action"
                value={filterInputs.action}
              />
              <Input
                onChange={(event) => updateFilter("targetType", event.target.value)}
                placeholder="target type"
                value={filterInputs.targetType}
              />
              <Input
                onChange={(event) => updateFilter("targetID", event.target.value)}
                placeholder="target id"
                value={filterInputs.targetID}
              />
              <select
                className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 shadow-sm outline-none transition focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
                onChange={(event) => updateFilter("success", event.target.value as SuccessFilter)}
                value={filterInputs.success}
              >
                <option value="all">All Results</option>
                <option value="true">Success</option>
                <option value="false">Failed</option>
              </select>
              <Input
                onChange={(event) => updateFilter("resultCode", event.target.value)}
                placeholder="result code"
                value={filterInputs.resultCode}
              />
              <Input
                onChange={(event) => updateFilter("requestID", event.target.value)}
                placeholder="request id"
                value={filterInputs.requestID}
              />
              <Input
                onChange={(event) => updateFilter("traceID", event.target.value)}
                placeholder="trace id"
                value={filterInputs.traceID}
              />
            </div>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex gap-2">
                <Button type="submit">Filter</Button>
                <Button onClick={clearFilter} type="button" variant="ghost">
                  Clear
                </Button>
              </div>
              <div className="flex gap-2">
                <Button
                  disabled={exportingFormat !== null}
                  onClick={() => handleExport("csv")}
                  type="button"
                  variant="ghost"
                >
                  {exportingFormat === "csv" ? "Exporting..." : "Export CSV"}
                </Button>
                <Button
                  disabled={exportingFormat !== null}
                  onClick={() => handleExport("jsonl")}
                  type="button"
                  variant="ghost"
                >
                  {exportingFormat === "jsonl" ? "Exporting..." : "Export JSONL"}
                </Button>
              </div>
            </div>
          </form>
          {feedback && (
            <p className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
              {feedback}
            </p>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Logs</CardTitle>
          </CardHeader>
          <CardContent>
            {logsQuery.isLoading ? (
              <p className="text-sm text-slate-600">Loading logs...</p>
            ) : logsQuery.isError ? (
              <QueryErrorCard
                className="shadow-none"
                title="Failed to load audit logs"
                message={logsLoadError?.message || "Failed to load logs."}
                hint={logsLoadError?.hint}
                onRetry={() => logsQuery.refetch()}
              />
            ) : (
              <div className="space-y-3">
                <div className="overflow-x-auto rounded-lg border border-slate-200">
                  <table className="min-w-[1180px] w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-600">
                      <tr>
                        <th className="px-4 py-3">Time</th>
                        <th className="px-4 py-3">User</th>
                        <th className="px-4 py-3">IP</th>
                        <th className="px-4 py-3">Module</th>
                        <th className="px-4 py-3">Action</th>
                        <th className="px-4 py-3">Target</th>
                        <th className="px-4 py-3">Request</th>
                        <th className="px-4 py-3">Result</th>
                        <th className="px-4 py-3">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(logsQuery.data?.items || []).map((item) => {
                        const taskID = taskIDFor(item);
                        return (
                          <tr
                            className={[
                              "border-t border-slate-200",
                              selectedLog?.id === item.id ? "bg-slate-50" : ""
                            ].join(" ")}
                            key={item.id}
                          >
                            <td className="px-4 py-3">{new Date(item.created_at).toLocaleString()}</td>
                            <td className="px-4 py-3">{item.username || "-"}</td>
                            <td className="px-4 py-3">{item.ip || "-"}</td>
                            <td className="px-4 py-3">{item.module}</td>
                            <td className="px-4 py-3">{item.action}</td>
                            <td className="px-4 py-3">
                              {taskID ? (
                                <Link className="font-medium text-slate-900 underline" to={`/tasks?task_id=${taskID}`}>
                                  Task #{taskID}
                                </Link>
                              ) : (
                                targetLabel(item)
                              )}
                            </td>
                            <td className="px-4 py-3">
                              {item.request_id ? (
                                <div className="flex items-center gap-2">
                                  <code className="max-w-[160px] truncate rounded bg-slate-100 px-2 py-1 text-xs">
                                    {item.request_id}
                                  </code>
                                  <Button onClick={() => copyRequestID(item.request_id)} size="sm" variant="ghost">
                                    Copy
                                  </Button>
                                </div>
                              ) : (
                                "-"
                              )}
                            </td>
                            <td className="px-4 py-3">
                              <span
                                className={[
                                  "rounded px-2 py-1 text-xs font-medium",
                                  item.success ? "bg-emerald-100 text-emerald-700" : "bg-rose-100 text-rose-700"
                                ].join(" ")}
                              >
                                {item.success ? "success" : "failed"}
                              </span>
                            </td>
                            <td className="px-4 py-3">
                              <Button
                                onClick={() => setSelectedLog(item)}
                                size="sm"
                                variant={selectedLog?.id === item.id ? "default" : "ghost"}
                              >
                                {selectedLog?.id === item.id ? "Viewing" : "Details"}
                              </Button>
                            </td>
                          </tr>
                        );
                      })}
                      {(logsQuery.data?.items || []).length === 0 && (
                        <tr>
                          <td className="px-4 py-8 text-center text-slate-500" colSpan={9}>
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

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Log Detail</CardTitle>
          </CardHeader>
          <CardContent>
            {selectedLog ? (
              <div className="space-y-4">
                <dl className="grid gap-3">
                  <DetailLine label="Time" value={new Date(selectedLog.created_at).toLocaleString()} />
                  <DetailLine label="User" value={selectedLog.username || "-"} />
                  <DetailLine label="Module" value={selectedLog.module} />
                  <DetailLine label="Action" value={selectedLog.action} />
                  <DetailLine label="Target" value={targetLabel(selectedLog)} />
                  <DetailLine label="Result" value={selectedLog.success ? "success" : "failed"} />
                  <DetailLine label="Result Code" value={selectedLog.result_code || "-"} />
                  <DetailLine label="Request ID" value={selectedLog.request_id || "-"} />
                  <DetailLine label="Trace ID" value={selectedLog.trace_id || "-"} />
                  <DetailLine label="IP" value={selectedLog.ip || "-"} />
                </dl>
                {taskIDFor(selectedLog) && (
                  <Link
                    className="inline-flex rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-800 hover:bg-slate-50"
                    to={`/tasks?task_id=${taskIDFor(selectedLog)}`}
                  >
                    Open task detail
                  </Link>
                )}
                <div>
                  <p className="text-xs font-medium uppercase text-slate-500">Request Summary</p>
                  <pre className="mt-2 max-h-48 overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-100">
                    {selectedLog.request_summary || "-"}
                  </pre>
                </div>
                <div>
                  <p className="text-xs font-medium uppercase text-slate-500">Result Message</p>
                  <pre className="mt-2 max-h-48 overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-100">
                    {selectedLog.result_message || "-"}
                  </pre>
                </div>
              </div>
            ) : (
              <p className="text-sm text-slate-600">Select a log row to view forensic details.</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
