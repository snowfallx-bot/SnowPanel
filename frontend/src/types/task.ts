export interface TaskSummary {
  id: number;
  type: string;
  status: string;
  progress: number;
  error_message: string;
  triggered_by: number | null;
  created_at: string;
  updated_at: string;
}

export interface TaskLog {
  id: number;
  level: string;
  message: string;
  metadata: string;
  created_at: string;
}

export interface ListTasksResult {
  page: number;
  size: number;
  total: number;
  items: TaskSummary[];
}

export interface TaskDetailResult {
  summary: TaskSummary;
  logs: TaskLog[];
}

export interface CreateDemoTaskResult {
  id: number;
  type: string;
  status: string;
}
