export interface HostSummary {
  id: number;
  name: string;
  address: string;
  port: number;
  status: string;
  agent_version: string;
  capabilities?: string[];
  last_seen_at?: string | null;
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
