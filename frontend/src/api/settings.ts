import { http, unwrap } from "@/lib/http";
import {
  CreateSettingRequest,
  ListSettingsResponse,
  SettingItem,
  UpdateSettingRequest,
} from "@/types/settings";

export function listSettings() {
  return unwrap<ListSettingsResponse>(http.get("/api/v1/settings"));
}

export function getSetting(key: string) {
  return unwrap<SettingItem>(http.get(`/api/v1/settings/${key}`));
}

export function createSetting(payload: CreateSettingRequest) {
  return unwrap<SettingItem>(http.post("/api/v1/settings", payload));
}

export function updateSetting(key: string, payload: UpdateSettingRequest) {
  return unwrap<SettingItem>(http.put(`/api/v1/settings/${key}`, payload));
}

export function deleteSetting(key: string) {
  return unwrap<{}>(http.delete(`/api/v1/settings/${key}`));
}
