export interface ServiceInfo {
  name: string;
  display_name: string;
  status: string;
}

export interface ListServicesResult {
  services: ServiceInfo[];
}

export interface ServiceActionResult {
  name: string;
  status: string;
}
