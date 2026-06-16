export interface HostSummary {
  id: number;
  name: string;
  address: string;
  port: number;
  status: string;
  agent_version: string;
  capabilities?: string[];
  last_seen_at?: string | null;
  enrollment_id?: string;
  revoked: boolean;
  revoked_at?: string | null;
  revoked_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface ListHostsResult {
  items: HostSummary[];
}

export interface HostFormInput {
  name?: string;
  address: string;
  port: number;
}

export interface EnrollHostInput {
  token: string;
  hostname: string;
  agent_version?: string;
  capabilities?: string[];
}

export interface RotateCertificateInput {
  reuse_key: boolean;
}
