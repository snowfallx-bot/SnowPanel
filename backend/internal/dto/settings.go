package dto

type SettingItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	IsEncrypted bool   `json:"is_encrypted"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ListSettingsResponse struct {
	Items []SettingItem `json:"items"`
}

type CreateSettingRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	ValueType   string `json:"value_type" binding:"required"`
	Description string `json:"description"`
	IsEncrypted bool   `json:"is_encrypted"`
}

type UpdateSettingRequest struct {
	Value       *string `json:"value,omitempty"`
	ValueType   *string `json:"value_type,omitempty"`
	Description *string `json:"description,omitempty"`
	IsEncrypted *bool   `json:"is_encrypted,omitempty"`
}
