export interface DatabaseInstance {
  id: number;
  name: string;
  engine: string;
  host: string;
  port: number;
  username: string;
  status: number;
  created_at: string;
  updated_at: string;
  databases_count?: number;
}

export interface Database {
  id: number;
  instance_id: number;
  name: string;
  owner: string;
  charset: string;
  collation: string;
  created_at: string;
}

export interface ListDatabaseInstancesResponse {
  items: DatabaseInstance[];
}

export interface CreateDatabaseInstanceRequest {
  name: string;
  engine: 'postgresql' | 'mysql';
  host: string;
  port: number;
  username: string;
  password: string;
}

export interface UpdateDatabaseInstanceRequest {
  name?: string;
  engine?: 'postgresql' | 'mysql';
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  status?: number;
}

export interface TestConnectionResponse {
  success: boolean;
  message: string;
}

export interface ListDatabasesResponse {
  items: Database[];
}

export interface CreateDatabaseRequest {
  instance_id: number;
  name: string;
  owner: string;
  charset?: string;
  collation?: string;
}

export interface DeleteDatabaseRequest {
  instance_id: number;
  name: string;
}
