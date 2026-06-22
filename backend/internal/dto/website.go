package dto

import (
	"time"
)

type Website struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	RootPath  string      `json:"root_path"`
	Runtime   string      `json:"runtime"`
	Status    int16       `json:"status"`
	HostID    *int64      `json:"host_id,omitempty"`
	Domains   []string    `json:"domains"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type WebsiteListItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	RootPath  string    `json:"root_path"`
	Runtime   string    `json:"runtime"`
	Status    int16     `json:"status"`
	HostID    *int64    `json:"host_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListWebsitesResponse struct {
	Items []WebsiteListItem `json:"items"`
}

type CreateWebsiteRequest struct {
	Name      string   `json:"name" binding:"required"`
	RootPath  string   `json:"root_path" binding:"required"`
	Runtime   string   `json:"runtime" binding:"required,oneof=php node python static"`
	Domains   []string `json:"domains" binding:"required,dive,required"`
	HostID    *int64   `json:"host_id,omitempty"`
}

type UpdateWebsiteRequest struct {
	RootPath  *string  `json:"root_path,omitempty"`
	Runtime   *string  `json:"runtime,omitempty"`
	Domains   *[]string `json:"domains,omitempty"`
	Status    *int16   `json:"status,omitempty"`
	HostID    *int64   `json:"host_id,omitempty"`
}

type WebsiteDomain struct {
	ID        int64     `json:"id"`
	WebsiteID int64     `json:"website_id"`
	Domain    string    `json:"domain"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

type ListWebsiteDomainsResponse struct {
	Items []WebsiteDomain `json:"items"`
}

type CreateWebsiteDomainRequest struct {
	WebsiteID int64  `json:"website_id" binding:"required"`
	Domain    string `json:"domain" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
}

type UpdateWebsiteDomainRequest struct {
	Domain    *string `json:"domain,omitempty"`
	IsPrimary *bool   `json:"is_primary,omitempty"`
}

type DeleteWebsiteDomainRequest struct {
	DomainID int64 `json:"domain_id" binding:"required"`
}

type EnableWebsiteResponse struct {
	Message string `json:"message"`
}

type DisableWebsiteResponse struct {
	Message string `json:"message"`
}
