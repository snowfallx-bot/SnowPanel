import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  createCronTask,
  deleteCronTask,
  disableCronTask,
  enableCronTask,
  listCronTasks,
  updateCronTask
} from "@/api/cron";
import { CronPage } from "@/pages/CronPage";
import { CronTask } from "@/types/cron";

vi.mock("@/api/cron", () => ({
  listCronTasks: vi.fn(),
  createCronTask: vi.fn(),
  updateCronTask: vi.fn(),
  deleteCronTask: vi.fn(),
  enableCronTask: vi.fn(),
  disableCronTask: vi.fn()
}));

const tasksFixture: CronTask[] = [
  {
    id: "task-z",
    expression: "30 1 * * *",
    command: "sync",
    enabled: true
  },
  {
    id: "task-a",
    expression: "0 0 * * *",
    command: "cleanup",
    enabled: false
  },
  {
    id: "task-m",
    expression: "*/5 * * * *",
    command: "backup",
    enabled: true
  }
];

function buildQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false }
    }
  });
}

function renderCronPage() {
  const queryClient = buildQueryClient();

  vi.mocked(listCronTasks).mockResolvedValue({
    tasks: tasksFixture
  });
  vi.mocked(createCronTask).mockImplementation(async (payload) => ({
    task: {
      id: "new-task",
      expression: payload.expression,
      command: payload.command,
      enabled: payload.enabled
    }
  }));
  vi.mocked(updateCronTask).mockImplementation(async (id, payload) => ({
    task: {
      id,
      expression: payload.expression,
      command: payload.command,
      enabled: payload.enabled
    }
  }));
  vi.mocked(deleteCronTask).mockResolvedValue({ id: "task-a" });
  vi.mocked(enableCronTask).mockImplementation(async (id) => ({
    task: {
      id,
      expression: "0 0 * * *",
      command: "cleanup",
      enabled: true
    }
  }));
  vi.mocked(disableCronTask).mockImplementation(async (id) => ({
    task: {
      id,
      expression: "*/5 * * * *",
      command: "backup",
      enabled: false
    }
  }));

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <CronPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function visibleTaskIds() {
  return Array.from(document.querySelectorAll("tbody tr"))
    .filter((row) => row.children.length > 1)
    .map((row) => row.children[0]?.textContent?.trim() || "");
}

describe("CronPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("filters and sorts tasks in task list", async () => {
    renderCronPage();
    await screen.findByText("Showing 3 / 3 tasks");

    expect(visibleTaskIds()).toEqual(["task-a", "task-m", "task-z"]);

    const [stateSelect, sortSelect] = screen.getAllByRole("combobox");
    fireEvent.change(sortSelect, { target: { value: "id-desc" } });
    expect(visibleTaskIds()).toEqual(["task-z", "task-m", "task-a"]);

    fireEvent.change(screen.getByPlaceholderText("Filter by id, expression, or command"), {
      target: { value: "backup" }
    });
    expect(screen.getByText("Showing 1 / 3 tasks")).toBeInTheDocument();
    expect(visibleTaskIds()).toEqual(["task-m"]);

    fireEvent.change(stateSelect, { target: { value: "disabled" } });
    expect(screen.getByText("No cron tasks match the current filter.")).toBeInTheDocument();
  });

  it("clears list filters and resets sort mode", async () => {
    renderCronPage();
    await screen.findByText("Showing 3 / 3 tasks");

    const keywordInput = screen.getByPlaceholderText("Filter by id, expression, or command");
    const [stateSelect, sortSelect] = screen.getAllByRole("combobox");
    const clearButton = screen.getByRole("button", { name: "Clear filters" });

    expect(clearButton).toBeDisabled();

    fireEvent.change(keywordInput, { target: { value: "task" } });
    fireEvent.change(stateSelect, { target: { value: "enabled" } });
    fireEvent.change(sortSelect, { target: { value: "id-desc" } });
    expect(clearButton).toBeEnabled();

    fireEvent.click(clearButton);
    expect(keywordInput).toHaveValue("");
    expect(stateSelect).toHaveValue("all");
    expect(sortSelect).toHaveValue("id-asc");
    expect(screen.getByText("Filters cleared.")).toBeInTheDocument();
    expect(screen.getByText("Showing 3 / 3 tasks")).toBeInTheDocument();
    expect(clearButton).toBeDisabled();
  });

  it("submits create task form and shows success feedback", async () => {
    renderCronPage();
    await screen.findByText("Showing 3 / 3 tasks");

    fireEvent.change(screen.getByPlaceholderText("*/5 * * * *"), {
      target: { value: "0 */6 * * *" }
    });
    fireEvent.change(screen.getByPlaceholderText("command"), {
      target: { value: "backup-nightly" }
    });
    fireEvent.click(screen.getAllByRole("checkbox")[0]);
    fireEvent.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => {
      expect(createCronTask).toHaveBeenCalled();
      expect(vi.mocked(createCronTask).mock.calls[0]?.[0]).toEqual({
        expression: "0 */6 * * *",
        command: "backup-nightly",
        enabled: false
      });
    });
    expect(screen.getByText("Created task: new-task")).toBeInTheDocument();
  });

  it("submits edit form and calls update api with edited payload", async () => {
    renderCronPage();
    await screen.findByText("Showing 3 / 3 tasks");

    fireEvent.click(screen.getAllByRole("button", { name: "Edit" })[0]);

    fireEvent.change(screen.getByDisplayValue("0 0 * * *"), {
      target: { value: "1 1 * * *" }
    });
    fireEvent.change(screen.getByDisplayValue("cleanup"), {
      target: { value: "cleanup-now" }
    });
    fireEvent.click(screen.getAllByRole("checkbox")[1]);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(updateCronTask).toHaveBeenCalledWith("task-a", {
        expression: "1 1 * * *",
        command: "cleanup-now",
        enabled: true
      });
    });
    expect(screen.getByText("Updated task: task-a")).toBeInTheDocument();
  });
});
