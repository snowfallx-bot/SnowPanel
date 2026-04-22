import { http, unwrap } from "@/lib/http";
import { CreateDemoTaskResult, ListTasksResult, TaskDetailResult } from "@/types/task";

export function listTasks(params: { page: number; size: number }) {
  return unwrap<ListTasksResult>(http.get("/api/v1/tasks", { params }));
}

export function getTaskDetail(id: number) {
  return unwrap<TaskDetailResult>(http.get(`/api/v1/tasks/${id}`));
}

export function createDemoTask() {
  return unwrap<CreateDemoTaskResult>(http.post("/api/v1/tasks/demo"));
}
