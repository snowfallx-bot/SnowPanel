export interface CronTask {
  id: string;
  expression: string;
  command: string;
  enabled: boolean;
}

export interface ListCronTasksResult {
  tasks: CronTask[];
}

export interface CreateCronTaskPayload {
  expression: string;
  command: string;
  enabled: boolean;
}

export interface CreateCronTaskResult {
  task: CronTask;
}

export interface UpdateCronTaskPayload {
  expression: string;
  command: string;
  enabled: boolean;
}

export interface UpdateCronTaskResult {
  task: CronTask;
}

export interface DeleteCronTaskResult {
  id: string;
}

export interface ToggleCronTaskResult {
  task: CronTask;
}
