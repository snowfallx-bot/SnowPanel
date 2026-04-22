import { http, unwrap } from "@/lib/http";
import { LoginPayload, LoginResult, UserProfile } from "@/types/auth";

export function login(payload: LoginPayload) {
  return unwrap<LoginResult>(http.post("/api/v1/auth/login", payload));
}

export function getMe() {
  return unwrap<UserProfile>(http.get("/api/v1/auth/me"));
}
