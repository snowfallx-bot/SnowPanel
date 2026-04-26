import { http, unwrap } from "@/lib/http";
import { withEncodedSegment } from "@/api/path";
import {
  CreateCronTaskPayload,
  CreateCronTaskResult,
  DeleteCronTaskResult,
  ListCronTasksResult,
  ToggleCronTaskResult,
  UpdateCronTaskPayload,
  UpdateCronTaskResult
} from "@/types/cron";

export function listCronTasks() {
  return unwrap<ListCronTasksResult>(http.get("/api/v1/cron"));
}

export function createCronTask(payload: CreateCronTaskPayload) {
  return unwrap<CreateCronTaskResult>(http.post("/api/v1/cron", payload));
}

export function updateCronTask(id: string, payload: UpdateCronTaskPayload) {
  return unwrap<UpdateCronTaskResult>(http.put(withEncodedSegment("/api/v1/cron", id), payload));
}

export function deleteCronTask(id: string) {
  return unwrap<DeleteCronTaskResult>(http.delete(withEncodedSegment("/api/v1/cron", id)));
}

export function enableCronTask(id: string) {
  return unwrap<ToggleCronTaskResult>(http.post(withEncodedSegment("/api/v1/cron", id, "/enable")));
}

export function disableCronTask(id: string) {
  return unwrap<ToggleCronTaskResult>(http.post(withEncodedSegment("/api/v1/cron", id, "/disable")));
}
