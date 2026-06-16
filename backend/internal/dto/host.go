package dto

type CreateHostRequest struct {
	Name    string `json:"name"`
	Address string `json:"address" binding:"required"`
	Port    int    `json:"port" binding:"required"`
}

type UpdateHostRequest struct {
	Name    string `json:"name"`
	Address string `json:"address" binding:"required"`
	Port    int    `json:"port" binding:"required"`
}

type EnrollHostRequest struct {
	Token        string   `json:"token" binding:"required"`
	Hostname     string   `json:"hostname" binding:"required"`
	AgentVersion string   `json:"agent_version"`
	Capabilities []string `json:"capabilities"`
}

type Host struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	Port            int      `json:"port"`
	Status          string   `json:"status"`
	AgentVersion    string   `json:"agent_version"`
	Capabilities    []string `json:"capabilities,omitempty"`
	LastSeenAt      *string  `json:"last_seen_at,omitempty"`
	EnrollmentID    string   `json:"enrollment_id,omitempty"`
	Revoked         bool     `json:"revoked"`
	RevokedAt       *string  `json:"revoked_at,omitempty"`
	RevokedReason   string   `json:"revoked_reason,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type ListHostsResult struct {
	Items []Host `json:"items"`
}
