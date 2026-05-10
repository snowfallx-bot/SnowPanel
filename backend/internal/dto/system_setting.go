package dto

type UpsertSystemSettingInput struct {
	Key         string
	Value       string
	ValueType   string
	IsEncrypted bool
	Description string
	UpdatedBy   *int64
}

type SystemSetting struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	IsEncrypted bool   `json:"is_encrypted"`
	Description string `json:"description"`
	UpdatedBy   *int64 `json:"updated_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
