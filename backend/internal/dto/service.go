package dto

type ListServicesQuery struct {
	Keyword string `form:"keyword"`
}

type ServiceActionPath struct {
	Name string `uri:"name" binding:"required"`
}

type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type ListServicesResult struct {
	Services []ServiceInfo `json:"services"`
}

type ServiceActionResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
