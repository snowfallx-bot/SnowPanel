import { http, unwrap } from "@/lib/http";
import { EnrollHostInput, HostFormInput, HostSummary, ListHostsResult, RotateCertificateInput } from "@/types/host";

export function listHosts() {
  return unwrap<ListHostsResult>(http.get("/api/v1/hosts"));
}

export function createHost(input: HostFormInput) {
  return unwrap<HostSummary>(http.post("/api/v1/hosts", input));
}

export function updateHost(id: number, input: HostFormInput) {
  return unwrap<HostSummary>(http.put(`/api/v1/hosts/${id}`, input));
}

export function checkHost(id: number) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/check`));
}

export function enableHost(id: number) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/enable`));
}

export function disableHost(id: number) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/disable`));
}

export function enrollHost(id: number, input: EnrollHostInput) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/enroll`, input));
}

export function revokeHost(id: number, reason: string) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/revoke`, { reason }));
}

export function rotateCertificate(id: number, input: RotateCertificateInput) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/rotate-certificate`, input));
}

export function validateCertificate(id: number) {
  return unwrap<HostSummary>(http.post(`/api/v1/hosts/${id}/validate-certificate`, {}));
}
