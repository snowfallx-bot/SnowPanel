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

type Host struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Address      string   `json:"address"`
	Port         int      `json:"port"`
	Status       string   `json:"status"`
	AgentVersion string   `json:"agent_version"`
	Capabilities []string `json:"capabilities,omitempty"`
	LastSeenAt   *string  `json:"last_seen_at,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type ListHostsResult struct {
	Items []Host `json:"items"`
}
