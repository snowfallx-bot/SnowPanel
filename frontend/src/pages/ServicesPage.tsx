import { FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listServices, restartService, startService, stopService } from "@/api/services";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ServiceInfo } from "@/types/service";

type ActionType = "start" | "stop" | "restart";

export function ServicesPage() {
  const queryClient = useQueryClient();
  const [keyword, setKeyword] = useState("");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [feedback, setFeedback] = useState("");

  const servicesQuery = useQuery({
    queryKey: ["services", searchKeyword],
    queryFn: () => listServices(searchKeyword)
  });

  const actionMutation = useMutation({
    mutationFn: async (payload: { name: string; action: ActionType }) => {
      if (payload.action === "start") {
        return startService(payload.name);
      }
      if (payload.action === "stop") {
        return stopService(payload.name);
      }
      return restartService(payload.name);
    },
    onSuccess(result, variables) {
      setFeedback(`${variables.action} success: ${result.name} -> ${result.status}`);
      queryClient.invalidateQueries({ queryKey: ["services", searchKeyword] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Service action failed");
    }
  });

  const message = useMemo(() => {
    if (servicesQuery.isError) {
      return servicesQuery.error instanceof Error ? servicesQuery.error.message : "Failed to load services";
    }
    return feedback;
  }, [feedback, servicesQuery.error, servicesQuery.isError]);

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSearchKeyword(keyword.trim());
  }

  async function handleAction(item: ServiceInfo, action: ActionType) {
    const confirmed = window.confirm(`${action.toUpperCase()} service "${item.name}"?`);
    if (!confirmed) {
      return;
    }
    await actionMutation.mutateAsync({ name: item.name, action });
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Services</h2>
        <p className="text-sm text-slate-500">Manage Linux services via core-agent and systemd wrapper.</p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <form className="flex gap-2" onSubmit={submitSearch}>
            <Input onChange={(event) => setKeyword(event.target.value)} placeholder="keyword (optional)" value={keyword} />
            <Button type="submit">Search</Button>
          </form>
        </CardContent>
      </Card>

      {servicesQuery.isLoading ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Loading services...</CardTitle>
          </CardHeader>
        </Card>
      ) : (
        <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Display Name</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {(servicesQuery.data?.services || []).map((item) => (
                <tr className="border-t border-slate-200" key={item.name}>
                  <td className="px-4 py-3 font-medium">{item.name}</td>
                  <td className="px-4 py-3">{item.display_name || "-"}</td>
                  <td className="px-4 py-3">{item.status}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <Button
                        onClick={() => handleAction(item, "start")}
                        size="sm"
                        variant="ghost"
                      >
                        Start
                      </Button>
                      <Button
                        onClick={() => handleAction(item, "stop")}
                        size="sm"
                        variant="ghost"
                      >
                        Stop
                      </Button>
                      <Button
                        onClick={() => handleAction(item, "restart")}
                        size="sm"
                        variant="ghost"
                      >
                        Restart
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {(servicesQuery.data?.services || []).length === 0 && (
                <tr>
                  <td className="px-4 py-8 text-center text-slate-500" colSpan={4}>
                    No services found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {message && (
        <p className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">{message}</p>
      )}
    </div>
  );
}
