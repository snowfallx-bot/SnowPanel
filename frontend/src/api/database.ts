import { http } from '@/lib/http';
import type {
  DatabaseInstance,
  Database,
  ListDatabaseInstancesResponse,
  CreateDatabaseInstanceRequest,
  UpdateDatabaseInstanceRequest,
  TestConnectionResponse,
  ListDatabasesResponse,
  CreateDatabaseRequest,
  DeleteDatabaseRequest,
} from '@/types/database';

// Database Instance API
export const listDatabaseInstances = () => {
  return http.get<ListDatabaseInstancesResponse>('/database/instances');
};

export const getDatabaseInstance = (id: number) => {
  return http.get<DatabaseInstance>(`/database/instances/${id}`);
};

export const createDatabaseInstance = (data: CreateDatabaseInstanceRequest) => {
  return http.post<DatabaseInstance>('/database/instances', data);
};

export const updateDatabaseInstance = (id: number, data: UpdateDatabaseInstanceRequest) => {
  return http.put<DatabaseInstance>(`/database/instances/${id}`, data);
};

export const deleteDatabaseInstance = (id: number) => {
  return http.delete(`/database/instances/${id}`);
};

export const testConnection = (data: CreateDatabaseInstanceRequest) => {
  return http.post<TestConnectionResponse>('/database/instances/test', data);
};

// Database API
export const listDatabases = (instanceId: number) => {
  return http.get<ListDatabasesResponse>(`/database/instances/${instanceId}/databases`);
};

export const createDatabase = (instanceId: number, data: CreateDatabaseRequest) => {
  return http.post(`/database/instances/${instanceId}/databases`, data);
};

export const deleteDatabase = (instanceId: number, data: DeleteDatabaseRequest) => {
  return http.delete(`/database/instances/${instanceId}/databases`, { data });
};
