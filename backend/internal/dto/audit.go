package dto

type ListAuditLogsQuery struct {
	Page   int    `form:"page"`
	Size   int    `form:"size"`
	Module string `form:"module"`
	Action string `form:"action"`
}

type AuditLog struct {
	ID             int64  `json:"id"`
	UserID         *int64 `json:"user_id"`
	Username       string `json:"username"`
	IP             string `json:"ip"`
	Module         string `json:"module"`
	Action         string `json:"action"`
	TargetType     string `json:"target_type"`
	TargetID       string `json:"target_id"`
	RequestSummary string `json:"request_summary"`
	Success        bool   `json:"success"`
	ResultCode     string `json:"result_code"`
	ResultMessage  string `json:"result_message"`
	CreatedAt      string `json:"created_at"`
}

type ListAuditLogsResult struct {
	Page  int        `json:"page"`
	Size  int        `json:"size"`
	Total int64      `json:"total"`
	Items []AuditLog `json:"items"`
}

type RecordAuditInput struct {
	UserID         *int64
	Username       string
	IP             string
	Module         string
	Action         string
	TargetType     string
	TargetID       string
	RequestSummary string
	Success        bool
	ResultCode     string
	ResultMessage  string
}
