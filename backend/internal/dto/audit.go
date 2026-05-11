package dto

type ListAuditLogsQuery struct {
	Page       int    `form:"page"`
	Size       int    `form:"size"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	UserID     int64  `form:"user_id"`
	Username   string `form:"username"`
	Module     string `form:"module"`
	Action     string `form:"action"`
	TargetType string `form:"target_type"`
	TargetID   string `form:"target_id"`
	Success    *bool  `form:"success"`
	ResultCode string `form:"result_code"`
	RequestID  string `form:"request_id"`
	TraceID    string `form:"trace_id"`
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
	RequestID      string `json:"request_id"`
	TraceID        string `json:"trace_id"`
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
	RequestID      string
	TraceID        string
}
