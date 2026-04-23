import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  listDockerContainers,
  listDockerImages,
  restartDockerContainer,
  startDockerContainer,
  stopDockerContainer
} from "@/api/docker";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { DockerContainerInfo } from "@/types/docker";

type DockerAction = "start" | "stop" | "restart";

function formatSize(size: number) {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  if (size < 1024 * 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(size / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function DockerPage() {
  const queryClient = useQueryClient();
  const [feedback, setFeedback] = useState("");
  const [filter, setFilter] = useState("");
  const [activeActionKey, setActiveActionKey] = useState("");

  const containersQuery = useQuery({
    queryKey: ["docker", "containers"],
    queryFn: listDockerContainers
  });

  const imagesQuery = useQuery({
    queryKey: ["docker", "images"],
    queryFn: listDockerImages
  });

  const actionMutation = useMutation({
    mutationFn: async (payload: { id: string; action: DockerAction }) => {
      if (payload.action === "start") {
        return startDockerContainer(payload.id);
      }
      if (payload.action === "stop") {
        return stopDockerContainer(payload.id);
      }
      return restartDockerContainer(payload.id);
    },
    onMutate(payload) {
      setActiveActionKey(`${payload.action}:${payload.id}`);
      setFeedback(`${payload.action} requested: ${payload.id}`);
    },
    onSuccess(result, payload) {
      setFeedback(`${payload.action} success: ${result.id} -> ${result.state}`);
      queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Docker action failed");
    },
    onSettled() {
      setActiveActionKey("");
    }
  });

  const filteredContainers = useMemo(() => {
    const keyword = filter.trim().toLowerCase();
    const items = containersQuery.data?.containers || [];
    if (!keyword) {
      return items;
    }
    return items.filter((item) => {
      const haystacks = [item.name, item.id, item.image, item.state, item.status]
        .map((value) => value.toLowerCase())
        .join(" ");
      return haystacks.includes(keyword);
    });
  }, [containersQuery.data?.containers, filter]);

  const message = useMemo(() => {
    if (containersQuery.isError) {
      return containersQuery.error instanceof Error
        ? containersQuery.error.message
        : "Failed to load docker containers";
    }
    if (imagesQuery.isError) {
      return imagesQuery.error instanceof Error
        ? imagesQuery.error.message
        : "Failed to load docker images";
    }
    return feedback;
  }, [containersQuery.error, containersQuery.isError, feedback, imagesQuery.error, imagesQuery.isError]);

  async function handleAction(item: DockerContainerInfo, action: DockerAction) {
    const confirmed = window.confirm(`${action.toUpperCase()} container "${item.name || item.id}"?`);
    if (!confirmed) {
      return;
    }
    await actionMutation.mutateAsync({ id: item.id || item.name, action });
  }

  function isActionPending(item: DockerContainerInfo, action: DockerAction) {
    return activeActionKey === `${action}:${item.id || item.name}`;
  }

  function refreshAll() {
    setFeedback("Refreshing docker data...");
    queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
    queryClient.invalidateQueries({ queryKey: ["docker", "images"] });
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold text-slate-900">Docker</h2>
          <p className="text-sm text-slate-500">Manage containers and view images.</p>
        </div>
        <Button
          disabled={containersQuery.isFetching || imagesQuery.isFetching}
          onClick={refreshAll}
          size="sm"
          variant="ghost"
        >
          {containersQuery.isFetching || imagesQuery.isFetching ? "Refreshing..." : "Refresh"}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Containers</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2">
            <Input
              onChange={(event) => setFilter(event.target.value)}
              placeholder="Filter by name, image, state, or status"
              value={filter}
            />
          </div>
          {containersQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading containers...</p>
          ) : (
            <div className="overflow-hidden rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Image</th>
                    <th className="px-4 py-3">State</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredContainers.map((item) => (
                    <tr className="border-t border-slate-200" key={item.id || item.name}>
                      <td className="px-4 py-3">{item.name || item.id}</td>
                      <td className="px-4 py-3">{item.image || "-"}</td>
                      <td className="px-4 py-3">{item.state || "-"}</td>
                      <td className="px-4 py-3">{item.status || "-"}</td>
                      <td className="px-4 py-3">
                        <div className="flex gap-2">
                          <Button
                            disabled={actionMutation.isPending}
                            onClick={() => handleAction(item, "start")}
                            size="sm"
                            variant="ghost"
                          >
                            {isActionPending(item, "start") ? "Starting..." : "Start"}
                          </Button>
                          <Button
                            disabled={actionMutation.isPending}
                            onClick={() => handleAction(item, "stop")}
                            size="sm"
                            variant="ghost"
                          >
                            {isActionPending(item, "stop") ? "Stopping..." : "Stop"}
                          </Button>
                          <Button
                            disabled={actionMutation.isPending}
                            onClick={() => handleAction(item, "restart")}
                            size="sm"
                            variant="ghost"
                          >
                            {isActionPending(item, "restart") ? "Restarting..." : "Restart"}
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {filteredContainers.length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={5}>
                        {(containersQuery.data?.containers || []).length === 0
                          ? "No containers found."
                          : "No containers match the current filter."}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Images</CardTitle>
        </CardHeader>
        <CardContent>
          {imagesQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading images...</p>
          ) : (
            <div className="overflow-hidden rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-4 py-3">Image ID</th>
                    <th className="px-4 py-3">Tags</th>
                    <th className="px-4 py-3">Size</th>
                  </tr>
                </thead>
                <tbody>
                  {(imagesQuery.data?.images || []).map((item) => (
                    <tr className="border-t border-slate-200" key={item.id}>
                      <td className="px-4 py-3">{item.id}</td>
                      <td className="px-4 py-3">{item.repo_tags.join(", ") || "<none>"}</td>
                      <td className="px-4 py-3">{formatSize(item.size)}</td>
                    </tr>
                  ))}
                  {(imagesQuery.data?.images || []).length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={3}>
                        No images found.
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
