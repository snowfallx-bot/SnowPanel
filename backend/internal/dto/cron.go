package dto

type CronTask struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
	Command    string `json:"command"`
	Enabled    bool   `json:"enabled"`
}

type ListCronTasksResult struct {
	Tasks []CronTask `json:"tasks"`
}

type CreateCronTaskRequest struct {
	Expression string `json:"expression" binding:"required"`
	Command    string `json:"command" binding:"required"`
	Enabled    bool   `json:"enabled"`
}

type CreateCronTaskResult struct {
	Task CronTask `json:"task"`
}

type UpdateCronTaskPath struct {
	ID string `uri:"id" binding:"required"`
}

type UpdateCronTaskRequest struct {
	Expression string `json:"expression" binding:"required"`
	Command    string `json:"command" binding:"required"`
	Enabled    bool   `json:"enabled"`
}

type UpdateCronTaskResult struct {
	Task CronTask `json:"task"`
}

type DeleteCronTaskPath struct {
	ID string `uri:"id" binding:"required"`
}

type DeleteCronTaskResult struct {
	ID string `json:"id"`
}

type ToggleCronTaskPath struct {
	ID string `uri:"id" binding:"required"`
}

type ToggleCronTaskRequest struct {
	Enabled bool `json:"enabled"`
}

type ToggleCronTaskResult struct {
	Task CronTask `json:"task"`
}
