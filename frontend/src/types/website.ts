export interface Website {
  id: number;
  name: string;
  root_path: string;
  runtime: string;
  status: number;
  host_id?: number;
  domains: string[];
  created_at: string;
  updated_at: string;
}

export interface WebsiteListItem {
  id: number;
  name: string;
  root_path: string;
  runtime: string;
  status: number;
  host_id?: number;
  created_at: string;
  updated_at: string;
}

export interface ListWebsitesResponse {
  items: WebsiteListItem[];
}

export interface CreateWebsiteRequest {
  name: string;
  root_path: string;
  runtime: string;
  domains: string[];
  host_id?: number;
}

export interface UpdateWebsiteRequest {
  root_path?: string;
  runtime?: string;
  domains?: string[];
  status?: number;
  host_id?: number;
}

export type WebsiteRuntime = 'php' | 'node' | 'python' | 'static';

export type WebsiteStatus = 0 | 1;
