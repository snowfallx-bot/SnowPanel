import { http, unwrap } from "@/lib/http";
import { CreateTaskResult, ListTasksResult, TaskDetailResult, TaskStatusResult } from "@/types/task";

export interface ListTasksParams {
  page: number;
  size: number;
  status?: string;
  type?: string;
}

export function listTasks(params: ListTasksParams) {
  return unwrap<ListTasksResult>(http.get("/api/v1/tasks", { params }));
}

export function getTaskDetail(id: number) {
  return unwrap<TaskDetailResult>(http.get(`/api/v1/tasks/${id}`));
}

export function createDockerRestartTask(payload: { container_id: string }) {
  return unwrap<CreateTaskResult>(http.post("/api/v1/tasks/docker/restart", payload));
}

export function createServiceRestartTask(payload: { service_name: string }) {
  return unwrap<CreateTaskResult>(http.post("/api/v1/tasks/services/restart", payload));
}

export function cancelTask(id: number) {
  return unwrap<TaskStatusResult>(http.post(`/api/v1/tasks/${id}/cancel`));
}

export function retryTask(id: number) {
  return unwrap<CreateTaskResult>(http.post(`/api/v1/tasks/${id}/retry`));
}
