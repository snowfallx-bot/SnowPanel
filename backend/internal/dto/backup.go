package dto

type CreateBackupMetadataRequest struct {
	ResourceType string `json:"resource_type" binding:"required,max=32"`
	ResourceID   string `json:"resource_id" binding:"required,max=128"`
	StorageType  string `json:"storage_type,omitempty" binding:"omitempty,max=32"`
	FilePath     string `json:"file_path,omitempty" binding:"omitempty,max=1024"`
}

type CreateBackupTaskRequest struct {
	ResourceType   string `json:"resource_type" binding:"required,max=32"`
	ResourceID     string `json:"resource_id" binding:"required,max=128"`
	StorageType    string `json:"storage_type,omitempty" binding:"omitempty,max=32"`
	FilePath       string `json:"file_path,omitempty" binding:"omitempty,max=1024"`
	IdempotencyKey string `json:"idempotency_key,omitempty" binding:"omitempty,max=128"`
}

type VerifyBackupRequest struct {
	SizeBytes int64  `json:"size_bytes" binding:"required,min=1"`
	Checksum  string `json:"checksum" binding:"required,max=128"`
	FilePath  string `json:"file_path,omitempty" binding:"omitempty,max=1024"`
}

type CreateBackupVerifyTaskRequest struct {
	SizeBytes      int64  `json:"size_bytes" binding:"required,min=1"`
	Checksum       string `json:"checksum" binding:"required,max=128"`
	FilePath       string `json:"file_path,omitempty" binding:"omitempty,max=1024"`
	IdempotencyKey string `json:"idempotency_key,omitempty" binding:"omitempty,max=128"`
}

type CreateBackupTaskResult struct {
	Backup BackupSummary    `json:"backup"`
	Task   CreateTaskResult `json:"task"`
}

type CreateBackupVerifyTaskResult struct {
	Task CreateTaskResult `json:"task"`
}

type BackupRetentionCleanupRequest struct {
	DryRun              bool `json:"dry_run"`
	RetentionDays       int  `json:"retention_days,omitempty"`
	ArchiveBeforeDelete bool `json:"archive_before_delete,omitempty"`
}

type BackupRetentionCleanupResult struct {
	DryRun              bool   `json:"dry_run"`
	RetentionDays       int    `json:"retention_days"`
	Cutoff              string `json:"cutoff"`
	MatchedRows         int64  `json:"matched_rows"`
	DeletedRows         int64  `json:"deleted_rows"`
	ArchiveBeforeDelete bool   `json:"archive_before_delete"`
}

type ListBackupsQuery struct {
	Page         int    `form:"page"`
	Size         int    `form:"size"`
	Status       string `form:"status"`
	ResourceType string `form:"resource_type"`
	ResourceID   string `form:"resource_id"`
}

type BackupSummary struct {
	ID           int64  `json:"id"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	StorageType  string `json:"storage_type"`
	FilePath     string `json:"file_path"`
	SizeBytes    int64  `json:"size_bytes"`
	Checksum     string `json:"checksum"`
	Status       string `json:"status"`
	CreatedBy    *int64 `json:"created_by"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ListBackupsResult struct {
	Page  int             `json:"page"`
	Size  int             `json:"size"`
	Total int64           `json:"total"`
	Items []BackupSummary `json:"items"`
}
