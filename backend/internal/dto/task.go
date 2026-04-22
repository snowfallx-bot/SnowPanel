package dto

type CreateDemoTaskResult struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type ListTasksQuery struct {
	Page int `form:"page"`
	Size int `form:"size"`
}

type TaskLog struct {
	ID        int64  `json:"id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Metadata  string `json:"metadata"`
	CreatedAt string `json:"created_at"`
}

type TaskSummary struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	Error       string `json:"error_message"`
	TriggeredBy *int64 `json:"triggered_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ListTasksResult struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Total int64         `json:"total"`
	Items []TaskSummary `json:"items"`
}

type TaskDetail struct {
	Summary TaskSummary `json:"summary"`
	Logs    []TaskLog   `json:"logs"`
}
