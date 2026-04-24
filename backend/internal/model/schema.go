package model

import "time"

type User struct {
	ID           int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Username     string     `json:"username" gorm:"column:username;size:64;not null;uniqueIndex"`
	Email        string     `json:"email" gorm:"column:email;size:255;uniqueIndex"`
	PasswordHash string     `json:"-" gorm:"column:password_hash;not null"`
	Status       int16      `json:"status" gorm:"column:status;not null;default:1"`
	LastLoginAt  *time.Time `json:"last_login_at" gorm:"column:last_login_at"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

func (User) TableName() string {
	return "users"
}

type Role struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"column:name;size:64;not null;uniqueIndex"`
	Slug        string    `json:"slug" gorm:"column:slug;size:64;not null;uniqueIndex"`
	Description string    `json:"description" gorm:"column:description;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Role) TableName() string {
	return "roles"
}

type Permission struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"column:name;size:128;not null;uniqueIndex"`
	Description string    `json:"description" gorm:"column:description;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Permission) TableName() string {
	return "permissions"
}

type RolePermission struct {
	RoleID       int64     `json:"role_id" gorm:"column:role_id;primaryKey"`
	PermissionID int64     `json:"permission_id" gorm:"column:permission_id;primaryKey"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

type UserRole struct {
	UserID    int64     `json:"user_id" gorm:"column:user_id;primaryKey"`
	RoleID    int64     `json:"role_id" gorm:"column:role_id;primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

type AuditLog struct {
	ID             int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID         *int64    `json:"user_id" gorm:"column:user_id"`
	Username       string    `json:"username" gorm:"column:username;size:64;not null"`
	IP             string    `json:"ip" gorm:"column:ip;size:64"`
	Module         string    `json:"module" gorm:"column:module;size:64;not null"`
	Action         string    `json:"action" gorm:"column:action;size:64;not null"`
	TargetType     string    `json:"target_type" gorm:"column:target_type;size:64;not null"`
	TargetID       string    `json:"target_id" gorm:"column:target_id;size:128;not null"`
	RequestSummary string    `json:"request_summary" gorm:"column:request_summary;type:jsonb;not null"`
	Success        bool      `json:"success" gorm:"column:success;not null"`
	ResultCode     string    `json:"result_code" gorm:"column:result_code;size:32;not null"`
	ResultMessage  string    `json:"result_message" gorm:"column:result_message;not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type SystemSetting struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Key         string    `json:"key" gorm:"column:key;size:128;not null;uniqueIndex"`
	Value       string    `json:"value" gorm:"column:value;not null"`
	ValueType   string    `json:"value_type" gorm:"column:value_type;size:32;not null"`
	IsEncrypted bool      `json:"is_encrypted" gorm:"column:is_encrypted;not null"`
	Description string    `json:"description" gorm:"column:description;not null"`
	UpdatedBy   *int64    `json:"updated_by" gorm:"column:updated_by"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}

type Host struct {
	ID           int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name         string     `json:"name" gorm:"column:name;size:128;not null"`
	Address      string     `json:"address" gorm:"column:address;size:255;not null"`
	Port         int        `json:"port" gorm:"column:port;not null"`
	Status       int16      `json:"status" gorm:"column:status;not null;default:1"`
	AgentVersion string     `json:"agent_version" gorm:"column:agent_version;size:64;not null"`
	LastSeenAt   *time.Time `json:"last_seen_at" gorm:"column:last_seen_at"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Host) TableName() string {
	return "hosts"
}

type Task struct {
	ID          int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Type        string     `json:"type" gorm:"column:type;size:64;not null"`
	Status      string     `json:"status" gorm:"column:status;size:16;not null"`
	Progress    int        `json:"progress" gorm:"column:progress;not null"`
	Payload     string     `json:"payload" gorm:"column:payload;type:jsonb;not null"`
	Result      string     `json:"result" gorm:"column:result;type:jsonb;not null"`
	ErrorMsg    string     `json:"error_message" gorm:"column:error_message;not null"`
	TriggeredBy *int64     `json:"triggered_by" gorm:"column:triggered_by"`
	HostID      *int64     `json:"host_id" gorm:"column:host_id"`
	StartedAt   *time.Time `json:"started_at" gorm:"column:started_at"`
	FinishedAt  *time.Time `json:"finished_at" gorm:"column:finished_at"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Task) TableName() string {
	return "tasks"
}

type TaskLog struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TaskID    int64     `json:"task_id" gorm:"column:task_id;not null"`
	Level     string    `json:"level" gorm:"column:level;size:16;not null"`
	Message   string    `json:"message" gorm:"column:message;not null"`
	Metadata  string    `json:"metadata" gorm:"column:metadata;type:jsonb;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (TaskLog) TableName() string {
	return "task_logs"
}

type Website struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"column:name;size:128;not null;uniqueIndex"`
	RootPath  string    `json:"root_path" gorm:"column:root_path;not null"`
	Runtime   string    `json:"runtime" gorm:"column:runtime;size:32;not null"`
	Status    int16     `json:"status" gorm:"column:status;not null"`
	HostID    *int64    `json:"host_id" gorm:"column:host_id"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Website) TableName() string {
	return "websites"
}

type WebsiteDomain struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	WebsiteID int64     `json:"website_id" gorm:"column:website_id;not null"`
	Domain    string    `json:"domain" gorm:"column:domain;size:255;not null;uniqueIndex"`
	IsPrimary bool      `json:"is_primary" gorm:"column:is_primary;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (WebsiteDomain) TableName() string {
	return "website_domains"
}

type DatabaseInstance struct {
	ID                int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name              string    `json:"name" gorm:"column:name;size:128;not null;uniqueIndex"`
	Engine            string    `json:"engine" gorm:"column:engine;size:32;not null"`
	Host              string    `json:"host" gorm:"column:host;size:255;not null"`
	Port              int       `json:"port" gorm:"column:port;not null"`
	Username          string    `json:"username" gorm:"column:username;size:128;not null"`
	PasswordEncrypted string    `json:"-" gorm:"column:password_encrypted;not null"`
	Status            int16     `json:"status" gorm:"column:status;not null"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (DatabaseInstance) TableName() string {
	return "database_instances"
}

type Database struct {
	ID         int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	InstanceID int64     `json:"instance_id" gorm:"column:instance_id;not null"`
	Name       string    `json:"name" gorm:"column:name;size:128;not null"`
	Owner      string    `json:"owner" gorm:"column:owner;size:128;not null"`
	Charset    string    `json:"charset" gorm:"column:charset;size:32;not null"`
	Collation  string    `json:"collation" gorm:"column:db_collation;size:64;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Database) TableName() string {
	return "databases"
}

type Plugin struct {
	ID          int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string     `json:"name" gorm:"column:name;size:128;not null"`
	Slug        string     `json:"slug" gorm:"column:slug;size:128;not null;uniqueIndex"`
	Version     string     `json:"version" gorm:"column:version;size:64;not null"`
	Source      string     `json:"source" gorm:"column:source;size:255;not null"`
	Enabled     bool       `json:"enabled" gorm:"column:enabled;not null"`
	InstalledAt *time.Time `json:"installed_at" gorm:"column:installed_at"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Plugin) TableName() string {
	return "plugins"
}

type Backup struct {
	ID           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ResourceType string    `json:"resource_type" gorm:"column:resource_type;size:32;not null"`
	ResourceID   string    `json:"resource_id" gorm:"column:resource_id;size:128;not null"`
	StorageType  string    `json:"storage_type" gorm:"column:storage_type;size:32;not null"`
	FilePath     string    `json:"file_path" gorm:"column:file_path;not null"`
	SizeBytes    int64     `json:"size_bytes" gorm:"column:size_bytes;not null"`
	Checksum     string    `json:"checksum" gorm:"column:checksum;size:128;not null"`
	Status       string    `json:"status" gorm:"column:status;size:16;not null"`
	CreatedBy    *int64    `json:"created_by" gorm:"column:created_by"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Backup) TableName() string {
	return "backups"
}
