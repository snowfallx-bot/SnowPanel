import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  listDockerContainers,
  listDockerImages,
  restartDockerContainer,
  startDockerContainer,
  stopDockerContainer
} from "@/api/docker";
import { DockerPage } from "@/pages/DockerPage";

vi.mock("@/api/docker", () => ({
  listDockerContainers: vi.fn(),
  listDockerImages: vi.fn(),
  startDockerContainer: vi.fn(),
  stopDockerContainer: vi.fn(),
  restartDockerContainer: vi.fn()
}));

const containersFixture = [
  {
    id: "container-web",
    name: "web-01",
    image: "nginx:1.25",
    state: "running",
    status: "Up 1 hour"
  },
  {
    id: "container-job",
    name: "job-01",
    image: "busybox:1.36",
    state: "exited",
    status: "Exited (0) 10m ago"
  }
];

const imagesFixture = [
  {
    id: "sha256:nginx",
    repo_tags: ["nginx:1.25", "nginx:latest"],
    size: 100 * 1024 * 1024
  },
  {
    id: "sha256:redis",
    repo_tags: ["redis:7"],
    size: 70 * 1024 * 1024
  }
];

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

function buildQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false }
    }
  });
}

function renderDockerPage(initialEntry = "/docker") {
  const queryClient = buildQueryClient();

  vi.mocked(listDockerContainers).mockResolvedValue({
    containers: containersFixture
  });
  vi.mocked(listDockerImages).mockResolvedValue({
    images: imagesFixture
  });
  vi.mocked(startDockerContainer).mockResolvedValue({
    id: "container-web",
    state: "running"
  });
  vi.mocked(stopDockerContainer).mockResolvedValue({
    id: "container-web",
    state: "stopped"
  });
  vi.mocked(restartDockerContainer).mockResolvedValue({
    id: "container-web",
    state: "running"
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <DockerPage />
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function getSearchParams() {
  const search = screen.getByTestId("location-search").textContent ?? "";
  return new URLSearchParams(search);
}

describe("DockerPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("restores filter state from URL search params", async () => {
    renderDockerPage("/docker?container=web&state=running&image=nginx");

    await screen.findByText("Showing 1 / 2 containers");

    expect(screen.getByPlaceholderText("Filter by name, image, state, or status")).toHaveValue("web");
    expect(screen.getByRole("combobox")).toHaveValue("running");
    expect(screen.getByPlaceholderText("Filter images by id or tag")).toHaveValue("nginx");

    expect(screen.getByText("web-01")).toBeInTheDocument();
    expect(screen.queryByText("job-01")).not.toBeInTheDocument();
    expect(screen.getByText("sha256:nginx")).toBeInTheDocument();
    expect(screen.queryByText("sha256:redis")).not.toBeInTheDocument();
  });

  it("updates URL params from filters and clears them with clear button", async () => {
    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    const containerInput = screen.getByPlaceholderText("Filter by name, image, state, or status");
    const imageInput = screen.getByPlaceholderText("Filter images by id or tag");
    const stateSelect = screen.getByRole("combobox");
    const clearButton = screen.getByRole("button", { name: "Clear filters" });

    expect(clearButton).toBeDisabled();

    fireEvent.change(containerInput, { target: { value: "web" } });
    fireEvent.change(stateSelect, { target: { value: "running" } });
    fireEvent.change(imageInput, { target: { value: "nginx" } });

    expect(clearButton).toBeEnabled();

    await waitFor(() => {
      const params = getSearchParams();
      expect(params.get("container")).toBe("web");
      expect(params.get("state")).toBe("running");
      expect(params.get("image")).toBe("nginx");
    });

    fireEvent.click(clearButton);

    expect(containerInput).toHaveValue("");
    expect(stateSelect).toHaveValue("all");
    expect(imageInput).toHaveValue("");
    expect(screen.getByText("Filters cleared.")).toBeInTheDocument();
    expect(clearButton).toBeDisabled();

    await waitFor(() => {
      const params = getSearchParams();
      expect(params.get("container")).toBeNull();
      expect(params.get("state")).toBeNull();
      expect(params.get("image")).toBeNull();
    });
  });

  it("shows filtered empty-state copy when filters produce no match", async () => {
    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    fireEvent.change(screen.getByPlaceholderText("Filter by name, image, state, or status"), {
      target: { value: "does-not-exist" }
    });
    fireEvent.change(screen.getByPlaceholderText("Filter images by id or tag"), {
      target: { value: "does-not-exist" }
    });

    expect(screen.getByText("No containers match the current filter.")).toBeInTheDocument();
    expect(screen.getByText("No images match the current filter.")).toBeInTheDocument();
  });

  it("runs start/stop/restart actions and shows success feedback", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);

    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    fireEvent.click(screen.getAllByRole("button", { name: "Start" })[0]);
    await waitFor(() => {
      expect(startDockerContainer).toHaveBeenCalledWith("container-web");
    });
    expect(screen.getByText("start success: container-web -> running")).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);
    await waitFor(() => {
      expect(stopDockerContainer).toHaveBeenCalledWith("container-web");
    });
    expect(screen.getByText("stop success: container-web -> stopped")).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole("button", { name: "Restart" })[0]);
    await waitFor(() => {
      expect(restartDockerContainer).toHaveBeenCalledWith("container-web");
    });
    expect(screen.getByText("restart success: container-web -> running")).toBeInTheDocument();

    expect(confirmSpy).toHaveBeenCalledTimes(3);
  });

  it("shows action error feedback when docker action fails", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    vi.mocked(stopDockerContainer).mockRejectedValueOnce(new Error("stop failed for container-web"));

    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);

    await waitFor(() => {
      expect(stopDockerContainer).toHaveBeenCalledWith("container-web");
    });
    expect(screen.getByText("stop failed for container-web")).toBeInTheDocument();
  });

  it("does not run action when confirmation is cancelled", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);

    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    fireEvent.click(screen.getAllByRole("button", { name: "Start" })[0]);

    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(startDockerContainer).not.toHaveBeenCalled();
    expect(screen.queryByText(/start requested:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/start success:/i)).not.toBeInTheDocument();
  });

  it("shows pending action labels while docker actions are running", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    const startDeferred = createDeferred<{ id: string; state: string }>();
    vi.mocked(startDockerContainer).mockImplementationOnce(() => startDeferred.promise);
    fireEvent.click(screen.getAllByRole("button", { name: "Start" })[0]);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Starting..." })).toBeDisabled();
    });
    startDeferred.resolve({ id: "container-web", state: "running" });
    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: "Start" })[0]).toBeEnabled();
    });

    const stopDeferred = createDeferred<{ id: string; state: string }>();
    vi.mocked(stopDockerContainer).mockImplementationOnce(() => stopDeferred.promise);
    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Stopping..." })).toBeDisabled();
    });
    stopDeferred.resolve({ id: "container-web", state: "stopped" });
    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: "Stop" })[0]).toBeEnabled();
    });

    const restartDeferred = createDeferred<{ id: string; state: string }>();
    vi.mocked(restartDockerContainer).mockImplementationOnce(() => restartDeferred.promise);
    fireEvent.click(screen.getAllByRole("button", { name: "Restart" })[0]);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Restarting..." })).toBeDisabled();
    });
    restartDeferred.resolve({ id: "container-web", state: "running" });
    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: "Restart" })[0]).toBeEnabled();
    });
  });

  it("disables other action buttons while one action is pending", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderDockerPage("/docker");
    await screen.findByText("Showing 2 / 2 containers");

    const startDeferred = createDeferred<{ id: string; state: string }>();
    vi.mocked(startDockerContainer).mockImplementationOnce(() => startDeferred.promise);

    fireEvent.click(screen.getAllByRole("button", { name: "Start" })[0]);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Starting..." })).toBeDisabled();
    });

    for (const button of screen.getAllByRole("button", { name: "Start" })) {
      expect(button).toBeDisabled();
    }
    for (const button of screen.getAllByRole("button", { name: "Stop" })) {
      expect(button).toBeDisabled();
    }
    for (const button of screen.getAllByRole("button", { name: "Restart" })) {
      expect(button).toBeDisabled();
    }

    startDeferred.resolve({ id: "container-web", state: "running" });
    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: "Start" })[0]).toBeEnabled();
    });
  });
});
