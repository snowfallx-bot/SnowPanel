import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
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
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
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
  const [searchParams, setSearchParams] = useSearchParams();
  const [feedback, setFeedback] = useState("");
  const [filter, setFilter] = useState(() => searchParams.get("container") || "");
  const [stateFilter, setStateFilter] = useState(
    () => searchParams.get("state")?.trim().toLowerCase() || "all"
  );
  const [imageFilter, setImageFilter] = useState(() => searchParams.get("image") || "");
  const [activeActionKey, setActiveActionKey] = useState("");
  const dockerContainersQueryKey = ["docker", "containers"] as const;
  const dockerImagesQueryKey = ["docker", "images"] as const;

  const containersQuery = useQuery({
    queryKey: dockerContainersQueryKey,
    queryFn: listDockerContainers
  });

  const imagesQuery = useQuery({
    queryKey: dockerImagesQueryKey,
    queryFn: listDockerImages
  });
  const containersLoadError = containersQuery.isError
    ? describeApiError(containersQuery.error, "Failed to load docker containers.")
    : null;
  const imagesLoadError = imagesQuery.isError
    ? describeApiError(imagesQuery.error, "Failed to load docker images.")
    : null;

  function describeDockerMutationError(error: unknown, fallback: string) {
    return describeApiError(error, fallback).message;
  }

  async function runMutationAction(action: () => Promise<unknown>) {
    try {
      await action();
    } catch {
      // onError already updates feedback; avoid unhandled promise rejection from event handlers.
    }
  }

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
      queryClient.invalidateQueries({ queryKey: dockerContainersQueryKey });
    },
    onError(error) {
      setFeedback(describeDockerMutationError(error, "Docker action failed"));
    },
    onSettled() {
      setActiveActionKey("");
    }
  });

  const filteredContainers = useMemo(() => {
    const normalizeState = (value: string) => {
      const normalized = value.trim().toLowerCase();
      return normalized || "unknown";
    };
    const keyword = filter.trim().toLowerCase();
    const items = containersQuery.data?.containers || [];

    return items.filter((item) => {
      const stateMatch = stateFilter === "all" || normalizeState(item.state) === stateFilter;
      if (!stateMatch) {
        return false;
      }
      if (!keyword) {
        return true;
      }
      const haystacks = [item.name, item.id, item.image, item.state, item.status]
        .map((value) => value.toLowerCase())
        .join(" ");
      return haystacks.includes(keyword);
    });
  }, [containersQuery.data?.containers, filter, stateFilter]);

  const containerStateOptions = useMemo(() => {
    const states = new Set<string>();
    for (const item of containersQuery.data?.containers || []) {
      const normalized = item.state.trim().toLowerCase() || "unknown";
      states.add(normalized);
    }
    if (stateFilter !== "all") {
      states.add(stateFilter);
    }
    return ["all", ...Array.from(states).sort()];
  }, [containersQuery.data?.containers, stateFilter]);

  const filteredImages = useMemo(() => {
    const keyword = imageFilter.trim().toLowerCase();
    const items = imagesQuery.data?.images || [];
    if (!keyword) {
      return items;
    }
    return items.filter((item) => {
      const idMatched = item.id.toLowerCase().includes(keyword);
      const tagsMatched = item.repo_tags.join(" ").toLowerCase().includes(keyword);
      return idMatched || tagsMatched;
    });
  }, [imageFilter, imagesQuery.data?.images]);

  const hasActiveFilters = useMemo(
    () => filter.trim() !== "" || stateFilter !== "all" || imageFilter.trim() !== "",
    [filter, imageFilter, stateFilter]
  );

  useEffect(() => {
    const nextParams = new URLSearchParams(searchParams);
    const containerKeyword = filter.trim();
    const imageKeyword = imageFilter.trim();

    if (containerKeyword) {
      nextParams.set("container", containerKeyword);
    } else {
      nextParams.delete("container");
    }

    if (stateFilter !== "all") {
      nextParams.set("state", stateFilter);
    } else {
      nextParams.delete("state");
    }

    if (imageKeyword) {
      nextParams.set("image", imageKeyword);
    } else {
      nextParams.delete("image");
    }

    if (nextParams.toString() !== searchParams.toString()) {
      setSearchParams(nextParams, { replace: true });
    }
  }, [filter, imageFilter, searchParams, setSearchParams, stateFilter]);

  async function handleAction(item: DockerContainerInfo, action: DockerAction) {
    const confirmed = window.confirm(`${action.toUpperCase()} container "${item.name || item.id}"?`);
    if (!confirmed) {
      return;
    }
    await runMutationAction(() => actionMutation.mutateAsync({ id: item.id || item.name, action }));
  }

  function isActionPending(item: DockerContainerInfo, action: DockerAction) {
    return activeActionKey === `${action}:${item.id || item.name}`;
  }

  function refreshAll() {
    setFeedback("Refreshing docker data...");
    queryClient.invalidateQueries({ queryKey: dockerContainersQueryKey });
    queryClient.invalidateQueries({ queryKey: dockerImagesQueryKey });
  }

  function clearFilters() {
    setFilter("");
    setStateFilter("all");
    setImageFilter("");
    setFeedback("Filters cleared.");
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold text-slate-900">Docker</h2>
          <p className="text-sm text-slate-500">Manage containers and view images.</p>
        </div>
        <div className="flex items-center gap-2">
          <Button disabled={!hasActiveFilters} onClick={clearFilters} size="sm" variant="ghost">
            Clear filters
          </Button>
          <Button
            disabled={containersQuery.isFetching || imagesQuery.isFetching}
            onClick={refreshAll}
            size="sm"
            variant="ghost"
          >
            {containersQuery.isFetching || imagesQuery.isFetching ? "Refreshing..." : "Refresh"}
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Containers</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2 md:grid-cols-[minmax(0,1fr)_220px]">
            <Input
              onChange={(event) => setFilter(event.target.value)}
              placeholder="Filter by name, image, state, or status"
              value={filter}
            />
            <select
              className="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700"
              onChange={(event) => setStateFilter(event.target.value)}
              value={stateFilter}
            >
              {containerStateOptions.map((value) => (
                <option key={value} value={value}>
                  {value === "all" ? "All states" : value}
                </option>
              ))}
            </select>
          </div>
          {containersQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading containers...</p>
          ) : containersQuery.isError ? (
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load containers"
              message={containersLoadError?.message || "Failed to load docker containers."}
              hint={containersLoadError?.hint}
              onRetry={() => containersQuery.refetch()}
            />
          ) : (
            <>
              <p className="text-xs text-slate-500">
                Showing {filteredContainers.length} / {(containersQuery.data?.containers || []).length} containers
              </p>
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
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Images</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Input
            onChange={(event) => setImageFilter(event.target.value)}
            placeholder="Filter images by id or tag"
            value={imageFilter}
          />
          {imagesQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading images...</p>
          ) : imagesQuery.isError ? (
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load images"
              message={imagesLoadError?.message || "Failed to load docker images."}
              hint={imagesLoadError?.hint}
              onRetry={() => imagesQuery.refetch()}
            />
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
                  {filteredImages.map((item) => (
                    <tr className="border-t border-slate-200" key={item.id}>
                      <td className="px-4 py-3">{item.id}</td>
                      <td className="px-4 py-3">{item.repo_tags.join(", ") || "<none>"}</td>
                      <td className="px-4 py-3">{formatSize(item.size)}</td>
                    </tr>
                  ))}
                  {filteredImages.length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={3}>
                        {(imagesQuery.data?.images || []).length === 0
                          ? "No images found."
                          : "No images match the current filter."}
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
