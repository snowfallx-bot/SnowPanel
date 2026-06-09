import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { checkHost, createHost, disableHost, enableHost, listHosts, updateHost } from "@/api/hosts";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
import { useAuthStore } from "@/store/auth-store";
import { HostFormInput, HostSummary } from "@/types/host";

function statusClasses(status: string) {
  if (status === "online") {
    return "bg-emerald-100 text-emerald-700";
  }
  if (status === "disabled") {
    return "bg-slate-200 text-slate-700";
  }
  return "bg-rose-100 text-rose-700";
}

function emptyForm(): HostFormInput {
  return { name: "", address: "", port: 50051 };
}

export function HostsPage() {
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const canManageHosts = (user?.permissions || []).includes("hosts.manage");
  const [form, setForm] = useState<HostFormInput>(emptyForm());
  const [editingHostId, setEditingHostId] = useState<number | null>(null);
  const [feedback, setFeedback] = useState("");
  const hostsQueryKey = ["hosts", "registry"] as const;

  const hostsQuery = useQuery({
    queryKey: hostsQueryKey,
    queryFn: listHosts
  });
  const hostsLoadError = hostsQuery.isError ? describeApiError(hostsQuery.error, "Failed to load hosts.") : null;

  function resetForm() {
    setForm(emptyForm());
    setEditingHostId(null);
  }

  function startEdit(host: HostSummary) {
    setEditingHostId(host.id);
    setForm({ name: host.name, address: host.address, port: host.port });
    setFeedback("");
  }

  function normalizeForm(): HostFormInput | null {
    const address = form.address.trim();
    const name = form.name?.trim() || "";
    const port = Number(form.port);
    if (!address || !Number.isInteger(port) || port <= 0 || port > 65535) {
      setFeedback("Address is required and port must be between 1 and 65535.");
      return null;
    }
    return { name, address, port };
  }

  const createMutation = useMutation({
    mutationFn: createHost,
    onSuccess(result) {
      setFeedback(`Host #${result.id} created and online.`);
      resetForm();
      queryClient.invalidateQueries({ queryKey: hostsQueryKey });
    },
    onError(error) {
      setFeedback(describeApiError(error, "Create host failed.").message);
    }
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, input }: { id: number; input: HostFormInput }) => updateHost(id, input),
    onSuccess(result) {
      setFeedback(`Host #${result.id} updated and online.`);
      resetForm();
      queryClient.invalidateQueries({ queryKey: hostsQueryKey });
    },
    onError(error) {
      setFeedback(describeApiError(error, "Update host failed.").message);
    }
  });

  const checkMutation = useMutation({
    mutationFn: checkHost,
    onSuccess(result) {
      setFeedback(`Host #${result.id} health check succeeded.`);
      queryClient.invalidateQueries({ queryKey: hostsQueryKey });
    },
    onError(error) {
      setFeedback(describeApiError(error, "Host health check failed.").message);
      queryClient.invalidateQueries({ queryKey: hostsQueryKey });
    }
  });

  const enableMutation = useMutation({
    mutationFn: enableHost,
    onSuccess(result) {
      setFeedback(`Host #${result.id} enabled.`);
      queryClient.invalidateQueries({ queryKey: hostsQueryKey });
    },
    onError(error) {
      setFeedback(describeApiError(error, "Enable host failed.").message);
    }
  });

  const disableMutation = useMutation({
    mutationFn: disableHost,
    onSuccess(result) {
      setFeedback(`Host #${result.id} disabled.`);
      queryClient.invalidateQueries({ queryKey: hostsQueryKey });
    },
    onError(error) {
      setFeedback(describeApiError(error, "Disable host failed.").message);
    }
  });

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const input = normalizeForm();
    if (!input) {
      return;
    }
    if (editingHostId) {
      await updateMutation.mutateAsync({ id: editingHostId, input }).catch(() => undefined);
      return;
    }
    await createMutation.mutateAsync(input).catch(() => undefined);
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Hosts</h2>
        <p className="text-sm text-slate-500">Register and manage core-agent targets for multi-host operations.</p>
      </div>

      {canManageHosts ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{editingHostId ? `Edit Host #${editingHostId}` : "Register Host"}</CardTitle>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3 md:grid-cols-[1fr_1.5fr_120px_auto_auto]" onSubmit={handleSubmit}>
              <Input
                placeholder="name (optional)"
                value={form.name || ""}
                onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))}
              />
              <Input
                placeholder="address or hostname"
                value={form.address}
                onChange={(event) => setForm((prev) => ({ ...prev, address: event.target.value }))}
              />
              <Input
                min={1}
                max={65535}
                type="number"
                value={form.port}
                onChange={(event) => setForm((prev) => ({ ...prev, port: Number(event.target.value) }))}
              />
              <Button disabled={isSaving} type="submit">
                {isSaving ? "Saving..." : editingHostId ? "Update" : "Create"}
              </Button>
              {editingHostId && (
                <Button onClick={resetForm} type="button" variant="ghost">
                  Cancel
                </Button>
              )}
            </form>
            <p className="mt-2 text-xs text-slate-500">
              Create/update probes the target agent before saving so status, version, and last seen stay trustworthy.
            </p>
          </CardContent>
        </Card>
      ) : (
        <p className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">
          You only have read permission for hosts. Lifecycle actions require `hosts.manage`.
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Host Registry</CardTitle>
        </CardHeader>
        <CardContent>
          {hostsQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading hosts...</p>
          ) : hostsQuery.isError ? (
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load hosts"
              message={hostsLoadError?.message || "Failed to load hosts."}
              hint={hostsLoadError?.hint}
              onRetry={() => hostsQuery.refetch()}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-4 py-3">ID</th>
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Endpoint</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Agent</th>
                    <th className="px-4 py-3">Capabilities</th>
                    <th className="px-4 py-3">Last Seen</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {(hostsQuery.data?.items || []).map((host) => (
                    <tr className="border-t border-slate-200" key={host.id}>
                      <td className="px-4 py-3">#{host.id}</td>
                      <td className="px-4 py-3 font-medium text-slate-800">{host.name}</td>
                      <td className="px-4 py-3">{host.address}:{host.port}</td>
                      <td className="px-4 py-3">
                        <span className={["rounded px-2 py-1 text-xs font-medium", statusClasses(host.status)].join(" ")}>
                          {host.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">{host.agent_version || "-"}</td>
                      <td className="px-4 py-3">{host.capabilities?.length ? host.capabilities.join(", ") : "-"}</td>
                      <td className="px-4 py-3">
                        {host.last_seen_at ? new Date(host.last_seen_at).toLocaleString() : "-"}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-2">
                          <Button
                            disabled={!canManageHosts || checkMutation.isPending || host.status === "disabled"}
                            onClick={() => checkMutation.mutate(host.id)}
                            size="sm"
                            variant="ghost"
                          >
                            Check
                          </Button>
                          {canManageHosts && (
                            <Button onClick={() => startEdit(host)} size="sm" variant="ghost">
                              Edit
                            </Button>
                          )}
                          {canManageHosts && host.status === "disabled" ? (
                            <Button
                              disabled={enableMutation.isPending}
                              onClick={() => enableMutation.mutate(host.id)}
                              size="sm"
                              variant="ghost"
                            >
                              Enable
                            </Button>
                          ) : canManageHosts ? (
                            <Button
                              disabled={disableMutation.isPending}
                              onClick={() => disableMutation.mutate(host.id)}
                              size="sm"
                              variant="ghost"
                            >
                              Disable
                            </Button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  ))}
                  {(hostsQuery.data?.items || []).length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={8}>
                        No hosts registered. The primary agent is still available as the default target.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {feedback && (
        <p className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">{feedback}</p>
      )}
    </div>
  );
}
