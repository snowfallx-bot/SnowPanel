import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createDemoTask, getTaskDetail, listTasks } from "@/api/tasks";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

function isActiveTask(status: string) {
  return status === "pending" || status === "running";
}

function statusClasses(status: string) {
  if (status === "success") {
    return "bg-emerald-100 text-emerald-700";
  }
  if (status === "failed") {
    return "bg-rose-100 text-rose-700";
  }
  if (status === "running") {
    return "bg-amber-100 text-amber-700";
  }
  return "bg-slate-100 text-slate-700";
}

export function TasksPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [feedback, setFeedback] = useState("");

  const tasksQuery = useQuery({
    queryKey: ["tasks", page, size],
    queryFn: () => listTasks({ page, size }),
    refetchInterval(query) {
      const hasActive = (query.state.data?.items || []).some((item) => isActiveTask(item.status));
      return hasActive ? 2000 : false;
    }
  });

  const detailQuery = useQuery({
    queryKey: ["tasks", "detail", selectedTaskId],
    queryFn: () => getTaskDetail(selectedTaskId as number),
    enabled: selectedTaskId !== null,
    refetchInterval(query) {
      const status = query.state.data?.summary.status;
      if (!status) {
        return false;
      }
      return isActiveTask(status) ? 2000 : false;
    }
  });

  const createMutation = useMutation({
    mutationFn: createDemoTask,
    onSuccess(result) {
      setFeedback(`Created demo task #${result.id}`);
      setSelectedTaskId(result.id);
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "detail", result.id] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Create demo task failed");
    }
  });

  const message = useMemo(() => {
    if (tasksQuery.isError) {
      return tasksQuery.error instanceof Error ? tasksQuery.error.message : "Failed to load tasks";
    }
    if (detailQuery.isError) {
      return detailQuery.error instanceof Error ? detailQuery.error.message : "Failed to load task detail";
    }
    return feedback;
  }, [detailQuery.error, detailQuery.isError, feedback, tasksQuery.error, tasksQuery.isError]);

  const total = tasksQuery.data?.total ?? 0;
  const maxPage = Math.max(1, Math.ceil(total / size));

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-2xl font-semibold text-slate-900">Tasks</h2>
          <p className="text-sm text-slate-500">Track asynchronous jobs and inspect execution logs.</p>
        </div>
        <div className="flex gap-2">
          <Button
            disabled={createMutation.isPending}
            onClick={() => createMutation.mutate()}
            size="sm"
          >
            {createMutation.isPending ? "Creating..." : "Create Demo Task"}
          </Button>
          <Button
            onClick={() => {
              queryClient.invalidateQueries({ queryKey: ["tasks"] });
              if (selectedTaskId !== null) {
                queryClient.invalidateQueries({ queryKey: ["tasks", "detail", selectedTaskId] });
              }
            }}
            size="sm"
            variant="ghost"
          >
            Refresh
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Task List</CardTitle>
        </CardHeader>
        <CardContent>
          {tasksQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading tasks...</p>
          ) : (
            <div className="space-y-3">
              <div className="overflow-hidden rounded-lg border border-slate-200">
                <table className="w-full text-left text-sm">
                  <thead className="bg-slate-50 text-slate-600">
                    <tr>
                      <th className="px-4 py-3">ID</th>
                      <th className="px-4 py-3">Type</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Progress</th>
                      <th className="px-4 py-3">Triggered By</th>
                      <th className="px-4 py-3">Updated</th>
                      <th className="px-4 py-3">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(tasksQuery.data?.items || []).map((item) => (
                      <tr
                        className="border-t border-slate-200"
                        key={item.id}
                      >
                        <td className="px-4 py-3">#{item.id}</td>
                        <td className="px-4 py-3">{item.type}</td>
                        <td className="px-4 py-3">
                          <span className={["rounded px-2 py-1 text-xs font-medium", statusClasses(item.status)].join(" ")}>
                            {item.status}
                          </span>
                        </td>
                        <td className="px-4 py-3">{item.progress}%</td>
                        <td className="px-4 py-3">{item.triggered_by ?? "-"}</td>
                        <td className="px-4 py-3">{new Date(item.updated_at).toLocaleString()}</td>
                        <td className="px-4 py-3">
                          <Button
                            onClick={() => setSelectedTaskId(item.id)}
                            size="sm"
                            variant={selectedTaskId === item.id ? "default" : "ghost"}
                          >
                            {selectedTaskId === item.id ? "Viewing" : "View"}
                          </Button>
                        </td>
                      </tr>
                    ))}
                    {(tasksQuery.data?.items || []).length === 0 && (
                      <tr>
                        <td className="px-4 py-8 text-center text-slate-500" colSpan={7}>
                          No tasks found.
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
          <CardTitle className="text-base">Task Detail</CardTitle>
        </CardHeader>
        <CardContent>
          {selectedTaskId === null ? (
            <p className="text-sm text-slate-600">Select a task from the list to view detail.</p>
          ) : detailQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading task detail...</p>
          ) : detailQuery.isError ? (
            <p className="text-sm text-rose-600">
              {detailQuery.error instanceof Error ? detailQuery.error.message : "Failed to load task detail"}
            </p>
          ) : (
            <div className="space-y-3">
              <div className="grid gap-2 rounded-md border border-slate-200 bg-slate-50 p-3 text-sm md:grid-cols-4">
                <p>
                  <span className="text-slate-500">ID:</span> #{detailQuery.data?.summary.id}
                </p>
                <p>
                  <span className="text-slate-500">Type:</span> {detailQuery.data?.summary.type}
                </p>
                <p>
                  <span className="text-slate-500">Status:</span> {detailQuery.data?.summary.status}
                </p>
                <p>
                  <span className="text-slate-500">Progress:</span> {detailQuery.data?.summary.progress}%
                </p>
                <p>
                  <span className="text-slate-500">Created:</span>{" "}
                  {detailQuery.data?.summary.created_at
                    ? new Date(detailQuery.data.summary.created_at).toLocaleString()
                    : "-"}
                </p>
                <p>
                  <span className="text-slate-500">Updated:</span>{" "}
                  {detailQuery.data?.summary.updated_at
                    ? new Date(detailQuery.data.summary.updated_at).toLocaleString()
                    : "-"}
                </p>
                <p className="md:col-span-2">
                  <span className="text-slate-500">Error:</span> {detailQuery.data?.summary.error_message || "-"}
                </p>
              </div>

              <div className="overflow-hidden rounded-lg border border-slate-200">
                <table className="w-full text-left text-sm">
                  <thead className="bg-slate-50 text-slate-600">
                    <tr>
                      <th className="px-4 py-3">Time</th>
                      <th className="px-4 py-3">Level</th>
                      <th className="px-4 py-3">Message</th>
                      <th className="px-4 py-3">Metadata</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(detailQuery.data?.logs || []).map((log) => (
                      <tr className="border-t border-slate-200" key={log.id}>
                        <td className="px-4 py-3">{new Date(log.created_at).toLocaleString()}</td>
                        <td className="px-4 py-3">{log.level}</td>
                        <td className="px-4 py-3">{log.message}</td>
                        <td className="px-4 py-3">{log.metadata || "-"}</td>
                      </tr>
                    ))}
                    {(detailQuery.data?.logs || []).length === 0 && (
                      <tr>
                        <td className="px-4 py-8 text-center text-slate-500" colSpan={4}>
                          No logs yet.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {message && (
        <p className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">{message}</p>
      )}
    </div>
  );
}
