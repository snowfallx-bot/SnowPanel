import { http, unwrap } from "@/lib/http";
import { ListServicesResult, ServiceActionResult } from "@/types/service";
import { withEncodedSegment } from "@/api/path";

export function listServices(keyword: string) {
  return unwrap<ListServicesResult>(http.get("/api/v1/services", { params: { keyword } }));
}

export function startService(name: string) {
  return unwrap<ServiceActionResult>(http.post(withEncodedSegment("/api/v1/services", name, "/start")));
}

export function stopService(name: string) {
  return unwrap<ServiceActionResult>(http.post(withEncodedSegment("/api/v1/services", name, "/stop")));
}

export function restartService(name: string) {
  return unwrap<ServiceActionResult>(http.post(withEncodedSegment("/api/v1/services", name, "/restart")));
}
