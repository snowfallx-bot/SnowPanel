import { http, unwrap } from "@/lib/http";
import {
  CreateWebsiteRequest,
  ListWebsitesResponse,
  UpdateWebsiteRequest,
  Website,
  WebsiteListItem,
} from "@/types/website";

export function listWebsites() {
  return unwrap<ListWebsitesResponse>(http.get("/api/v1/websites"));
}

export function getWebsite(id: number) {
  return unwrap<Website>(http.get(`/api/v1/websites/${id}`));
}

export function createWebsite(payload: CreateWebsiteRequest) {
  return unwrap<Website>(http.post("/api/v1/websites", payload));
}

export function updateWebsite(id: number, payload: UpdateWebsiteRequest) {
  return unwrap<Website>(http.put(`/api/v1/websites/${id}`, payload));
}

export function deleteWebsite(id: number) {
  return unwrap<{}>(http.delete(`/api/v1/websites/${id}`));
}

export function enableWebsite(id: number) {
  return unwrap<{ message: string }>(http.post(`/api/v1/websites/${id}/enable`));
}

export function disableWebsite(id: number) {
  return unwrap<{ message: string }>(http.post(`/api/v1/websites/${id}/disable`));
}
