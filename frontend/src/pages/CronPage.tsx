import { FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createCronTask,
  deleteCronTask,
  disableCronTask,
  enableCronTask,
  listCronTasks,
  updateCronTask
} from "@/api/cron";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { CronTask } from "@/types/cron";

export function CronPage() {
  const queryClient = useQueryClient();
  const [expression, setExpression] = useState("*/5 * * * *");
  const [command, setCommand] = useState("echo 'hello from snowpanel'");
  const [enabled, setEnabled] = useState(true);
  const [feedback, setFeedback] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingExpression, setEditingExpression] = useState("");
  const [editingCommand, setEditingCommand] = useState("");
  const [editingEnabled, setEditingEnabled] = useState(true);

  const tasksQuery = useQuery({
    queryKey: ["cron", "tasks"],
    queryFn: listCronTasks
  });

  const createMutation = useMutation({
    mutationFn: createCronTask,
    onSuccess(result) {
      setFeedback(`Created task: ${result.task.id}`);
      queryClient.invalidateQueries({ queryKey: ["cron", "tasks"] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Create cron task failed");
    }
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { expression: string; command: string; enabled: boolean } }) =>
      updateCronTask(id, payload),
    onSuccess(result) {
      setFeedback(`Updated task: ${result.task.id}`);
      setEditingId(null);
      queryClient.invalidateQueries({ queryKey: ["cron", "tasks"] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Update cron task failed");
    }
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      enabled ? enableCronTask(id) : disableCronTask(id),
    onSuccess(result) {
      setFeedback(`Updated task state: ${result.task.id} -> ${result.task.enabled ? "enabled" : "disabled"}`);
      queryClient.invalidateQueries({ queryKey: ["cron", "tasks"] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Toggle cron task failed");
    }
  });

  const deleteMutation = useMutation({
    mutationFn: deleteCronTask,
    onSuccess(result) {
      setFeedback(`Deleted task: ${result.id}`);
      queryClient.invalidateQueries({ queryKey: ["cron", "tasks"] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Delete cron task failed");
    }
  });

  const message = useMemo(() => {
    if (tasksQuery.isError) {
      return tasksQuery.error instanceof Error ? tasksQuery.error.message : "Failed to load cron tasks";
    }
    return feedback;
  }, [feedback, tasksQuery.error, tasksQuery.isError]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await createMutation.mutateAsync({ expression, command, enabled });
  }

  function beginEdit(task: CronTask) {
    setEditingId(task.id);
    setEditingExpression(task.expression);
    setEditingCommand(task.command);
    setEditingEnabled(task.enabled);
  }

  async function saveEdit(taskId: string) {
    await updateMutation.mutateAsync({
      id: taskId,
      payload: {
        expression: editingExpression,
        command: editingCommand,
        enabled: editingEnabled
      }
    });
  }

  async function handleDelete(taskId: string) {
    const confirmed = window.confirm(`Delete cron task "${taskId}"?`);
    if (!confirmed) {
      return;
    }
    await deleteMutation.mutateAsync(taskId);
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Cron Tasks</h2>
        <p className="text-sm text-slate-500">Create and manage scheduled jobs with validation and enable/disable controls.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Create Task</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="grid gap-3 md:grid-cols-[1fr_2fr_auto_auto]" onSubmit={handleCreate}>
            <Input onChange={(event) => setExpression(event.target.value)} placeholder="*/5 * * * *" value={expression} />
            <Input onChange={(event) => setCommand(event.target.value)} placeholder="command" value={command} />
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                checked={enabled}
                onChange={(event) => setEnabled(event.target.checked)}
                type="checkbox"
              />
              Enabled
            </label>
            <Button disabled={createMutation.isPending} type="submit">
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Task List</CardTitle>
        </CardHeader>
        <CardContent>
          {tasksQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading cron tasks...</p>
          ) : (
            <div className="overflow-hidden rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-4 py-3">ID</th>
                    <th className="px-4 py-3">Expression</th>
                    <th className="px-4 py-3">Command</th>
                    <th className="px-4 py-3">Enabled</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {(tasksQuery.data?.tasks || []).map((task) => (
                    <tr className="border-t border-slate-200 align-top" key={task.id}>
                      <td className="px-4 py-3">{task.id}</td>
                      <td className="px-4 py-3">
                        {editingId === task.id ? (
                          <Input
                            onChange={(event) => setEditingExpression(event.target.value)}
                            value={editingExpression}
                          />
                        ) : (
                          task.expression
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {editingId === task.id ? (
                          <Input
                            onChange={(event) => setEditingCommand(event.target.value)}
                            value={editingCommand}
                          />
                        ) : (
                          task.command
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {editingId === task.id ? (
                          <label className="flex items-center gap-2 text-sm text-slate-700">
                            <input
                              checked={editingEnabled}
                              onChange={(event) => setEditingEnabled(event.target.checked)}
                              type="checkbox"
                            />
                            Enabled
                          </label>
                        ) : task.enabled ? (
                          "Yes"
                        ) : (
                          "No"
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-2">
                          {editingId === task.id ? (
                            <>
                              <Button
                                onClick={() => saveEdit(task.id)}
                                size="sm"
                                variant="ghost"
                              >
                                Save
                              </Button>
                              <Button onClick={() => setEditingId(null)} size="sm" variant="ghost">
                                Cancel
                              </Button>
                            </>
                          ) : (
                            <>
                              <Button onClick={() => beginEdit(task)} size="sm" variant="ghost">
                                Edit
                              </Button>
                              <Button
                                onClick={() => toggleMutation.mutate({ id: task.id, enabled: !task.enabled })}
                                size="sm"
                                variant="ghost"
                              >
                                {task.enabled ? "Disable" : "Enable"}
                              </Button>
                              <Button onClick={() => handleDelete(task.id)} size="sm" variant="ghost">
                                Delete
                              </Button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                  {(tasksQuery.data?.tasks || []).length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={5}>
                        No cron tasks managed by SnowPanel yet.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
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
