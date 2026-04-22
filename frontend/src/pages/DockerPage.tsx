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
    onSuccess(result, payload) {
      setFeedback(`${payload.action} success: ${result.id} -> ${result.state}`);
      queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Docker action failed");
    }
  });

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

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-slate-900">Docker</h2>
          <p className="text-sm text-slate-500">Manage containers and view images.</p>
        </div>
        <Button
          onClick={() => {
            queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
            queryClient.invalidateQueries({ queryKey: ["docker", "images"] });
          }}
          size="sm"
          variant="ghost"
        >
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Containers</CardTitle>
        </CardHeader>
        <CardContent>
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
                  {(containersQuery.data?.containers || []).map((item) => (
                    <tr className="border-t border-slate-200" key={item.id || item.name}>
                      <td className="px-4 py-3">{item.name || item.id}</td>
                      <td className="px-4 py-3">{item.image || "-"}</td>
                      <td className="px-4 py-3">{item.state || "-"}</td>
                      <td className="px-4 py-3">{item.status || "-"}</td>
                      <td className="px-4 py-3">
                        <div className="flex gap-2">
                          <Button onClick={() => handleAction(item, "start")} size="sm" variant="ghost">
                            Start
                          </Button>
                          <Button onClick={() => handleAction(item, "stop")} size="sm" variant="ghost">
                            Stop
                          </Button>
                          <Button onClick={() => handleAction(item, "restart")} size="sm" variant="ghost">
                            Restart
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {(containersQuery.data?.containers || []).length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={5}>
                        No containers found.
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
