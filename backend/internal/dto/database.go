package dto

import (
	"time"
)

type DatabaseInstance struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Engine            string    `json:"engine"`
	Host              string    `json:"host"`
	Port              int       `json:"port"`
	Username          string    `json:"username"`
	Status            int16     `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DatabasesCount    int64     `json:"databases_count,omitempty"`
}

type DatabaseInstanceListItem struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Engine            string    `json:"engine"`
	Host              string    `json:"host"`
	Port              int       `json:"port"`
	Status            int16     `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

type ListDatabaseInstancesResponse struct {
	Items []DatabaseInstanceListItem `json:"items"`
}

type CreateDatabaseInstanceRequest struct {
	Name     string `json:"name" binding:"required"`
	Engine   string `json:"engine" binding:"required,oneof=postgresql mysql"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateDatabaseInstanceRequest struct {
	Name     *string `json:"name,omitempty"`
	Engine   *string `json:"engine,omitempty"`
	Host     *string `json:"host,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Status   *int16  `json:"status,omitempty"`
}

type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Database struct {
	ID         int64  `json:"id"`
	InstanceID int64  `json:"instance_id"`
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Charset    string `json:"charset"`
	Collation  string `json:"collation"`
	CreatedAt  time.Time `json:"created_at"`
}

type ListDatabasesResponse struct {
	Items []Database `json:"items"`
}

type CreateDatabaseRequest struct {
	InstanceID int64  `json:"instance_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Owner      string `json:"owner" binding:"required"`
	Charset    string `json:"charset" default:"utf8mb4"`
	Collation  string `json:"collation" default:"utf8mb4_general_ci"`
}

type DeleteDatabaseRequest struct {
	InstanceID int64 `json:"instance_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
}
