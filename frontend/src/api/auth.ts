import { http, unwrap } from "@/lib/http";
import { ChangePasswordPayload, LoginPayload, LoginResult, UserProfile } from "@/types/auth";

export function login(payload: LoginPayload) {
  return unwrap<LoginResult>(http.post("/api/v1/auth/login", payload));
}

export function getMe() {
  return unwrap<UserProfile>(http.get("/api/v1/auth/me"));
}

export function changePassword(payload: ChangePasswordPayload) {
  return unwrap<LoginResult>(http.post("/api/v1/auth/change-password", payload));
}
