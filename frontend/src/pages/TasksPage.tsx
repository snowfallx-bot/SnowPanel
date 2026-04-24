import { FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  cancelTask,
  createDockerRestartTask,
  createServiceRestartTask,
  getTaskDetail,
  listTasks,
  retryTask
} from "@/api/tasks";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
import { useAuthStore } from "@/store/auth-store";

function isActiveTask(status: string) {
  return status === "pending" || status === "running";
}

function isRetryableTask(status: string) {
  return status === "failed" || status === "canceled";
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
  if (status === "canceled") {
    return "bg-slate-200 text-slate-700";
  }
  return "bg-slate-100 text-slate-700";
}

const taskStatusOptions = [
  { value: "all", label: "All Status" },
  { value: "pending", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "success", label: "Success" },
  { value: "failed", label: "Failed" },
  { value: "canceled", label: "Canceled" }
];

const taskTypeOptions = [
  { value: "all", label: "All Types" },
  { value: "docker_restart", label: "Docker Restart" },
  { value: "service_restart", label: "Service Restart" }
];

export function TasksPage() {
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const canManageTasks = (user?.permissions || []).includes("tasks.manage");
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [statusFilter, setStatusFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [feedback, setFeedback] = useState("");
  const [dockerContainerID, setDockerContainerID] = useState("");
  const [serviceName, setServiceName] = useState("");
  const statusParam = statusFilter === "all" ? undefined : statusFilter;
  const typeParam = typeFilter === "all" ? undefined : typeFilter;

  const tasksQuery = useQuery({
    queryKey: ["tasks", page, size, statusFilter, typeFilter],
    queryFn: () =>
      listTasks({
        page,
        size,
        status: statusParam,
        type: typeParam
      }),
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
  const tasksLoadError = tasksQuery.isError
    ? describeApiError(tasksQuery.error, "Failed to load tasks.")
    : null;
  const detailLoadError = detailQuery.isError
    ? describeApiError(detailQuery.error, "Failed to load task detail.")
    : null;

  const createDockerRestartMutation = useMutation({
    mutationFn: createDockerRestartTask,
    onSuccess(result) {
      setFeedback(`Queued docker restart task #${result.id}`);
      setSelectedTaskId(result.id);
      setDockerContainerID("");
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "detail", result.id] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Create docker restart task failed");
    }
  });

  const createServiceRestartMutation = useMutation({
    mutationFn: createServiceRestartTask,
    onSuccess(result) {
      setFeedback(`Queued service restart task #${result.id}`);
      setSelectedTaskId(result.id);
      setServiceName("");
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "detail", result.id] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Create service restart task failed");
    }
  });

  const cancelMutation = useMutation({
    mutationFn: cancelTask,
    onSuccess(result) {
      setFeedback(`Task #${result.id} canceled`);
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "detail", result.id] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Cancel task failed");
    }
  });

  const retryMutation = useMutation({
    mutationFn: retryTask,
    onSuccess(result) {
      setFeedback(`Retried task as #${result.id}`);
      setSelectedTaskId(result.id);
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "detail", result.id] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Retry task failed");
    }
  });

  const message = useMemo(() => {
    return feedback;
  }, [feedback]);

  const total = tasksQuery.data?.total ?? 0;
  const maxPage = Math.max(1, Math.ceil(total / size));

  async function handleCreateDockerRestartTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const containerID = dockerContainerID.trim();
    if (!containerID) {
      return;
    }
    await createDockerRestartMutation.mutateAsync({ container_id: containerID });
  }

  async function handleCreateServiceRestartTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = serviceName.trim();
    if (!value) {
      return;
    }
    await createServiceRestartMutation.mutateAsync({ service_name: value });
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Tasks</h2>
        <p className="text-sm text-slate-500">Queue real operations and track asynchronous execution.</p>
      </div>

      {canManageTasks ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Queue Docker Restart Task</CardTitle>
            </CardHeader>
            <CardContent>
              <form className="flex gap-2" onSubmit={handleCreateDockerRestartTask}>
                <Input
                  placeholder="container id or name"
                  value={dockerContainerID}
                  onChange={(event) => setDockerContainerID(event.target.value)}
                />
                <Button disabled={createDockerRestartMutation.isPending} type="submit">
                  {createDockerRestartMutation.isPending ? "Queueing..." : "Queue"}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Queue Service Restart Task</CardTitle>
            </CardHeader>
            <CardContent>
              <form className="flex gap-2" onSubmit={handleCreateServiceRestartTask}>
                <Input
                  placeholder="service name (e.g. nginx.service)"
                  value={serviceName}
                  onChange={(event) => setServiceName(event.target.value)}
                />
                <Button disabled={createServiceRestartMutation.isPending} type="submit">
                  {createServiceRestartMutation.isPending ? "Queueing..." : "Queue"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      ) : (
        <p className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">
          You only have read permission for tasks. Create/cancel/retry actions require `tasks.manage`.
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Task List</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="mb-3 flex flex-wrap gap-2">
            <label className="text-sm text-slate-600">
              Status
              <select
                className="ml-2 rounded-md border border-slate-300 px-2 py-1 text-sm"
                value={statusFilter}
                onChange={(event) => {
                  setStatusFilter(event.target.value);
                  setPage(1);
                }}
              >
                {taskStatusOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-sm text-slate-600">
              Type
              <select
                className="ml-2 rounded-md border border-slate-300 px-2 py-1 text-sm"
                value={typeFilter}
                onChange={(event) => {
                  setTypeFilter(event.target.value);
                  setPage(1);
                }}
              >
                {taskTypeOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {tasksQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading tasks...</p>
          ) : tasksQuery.isError ? (
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load tasks"
              message={tasksLoadError?.message || "Failed to load tasks."}
              hint={tasksLoadError?.hint}
              onRetry={() => tasksQuery.refetch()}
            />
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
                      <tr className="border-t border-slate-200" key={item.id}>
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
                          <div className="flex gap-2">
                            <Button
                              onClick={() => setSelectedTaskId(item.id)}
                              size="sm"
                              variant={selectedTaskId === item.id ? "default" : "ghost"}
                            >
                              {selectedTaskId === item.id ? "Viewing" : "View"}
                            </Button>
                            {canManageTasks && isActiveTask(item.status) && (
                              <Button
                                onClick={() => cancelMutation.mutate(item.id)}
                                size="sm"
                                variant="ghost"
                                disabled={cancelMutation.isPending}
                              >
                                Cancel
                              </Button>
                            )}
                            {canManageTasks && isRetryableTask(item.status) && (
                              <Button
                                onClick={() => retryMutation.mutate(item.id)}
                                size="sm"
                                variant="ghost"
                                disabled={retryMutation.isPending}
                              >
                                Retry
                              </Button>
                            )}
                          </div>
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
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load task detail"
              message={detailLoadError?.message || "Failed to load task detail."}
              hint={detailLoadError?.hint}
              onRetry={() => detailQuery.refetch()}
            />
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
