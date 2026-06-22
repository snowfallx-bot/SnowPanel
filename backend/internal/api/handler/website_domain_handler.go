package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type WebsiteDomainHandler struct {
	domainService service.WebsiteDomainService
	auditService  service.AuditService
}

func NewWebsiteDomainHandler(domainService service.WebsiteDomainService, auditService service.AuditService) *WebsiteDomainHandler {
	return &WebsiteDomainHandler{
		domainService: domainService,
		auditService:  auditService,
	}
}

func (h *WebsiteDomainHandler) ListDomains(c *gin.Context) {
	websiteIDStr := c.Param("website_id")
	if websiteIDStr == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website_id is required")
		return
	}

	websiteID, err := strconv.ParseInt(websiteIDStr, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website_id")
		return
	}

	result, err := h.domainService.ListDomains(c.Request.Context(), websiteID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.ListWebsiteDomainsResponse{Items: result})
}

func (h *WebsiteDomainHandler) CreateDomain(c *gin.Context) {
	websiteIDStr := c.Param("website_id")
	if websiteIDStr == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website_id is required")
		return
	}

	websiteID, err := strconv.ParseInt(websiteIDStr, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website_id")
		return
	}

	var req dto.CreateWebsiteDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}
	req.WebsiteID = websiteID

	result, err := h.domainService.CreateDomain(c.Request.Context(), req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)

	// Record audit log
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "website_domain",
		Action:       "create",
		TargetType:   "website_domain",
		TargetID:     strconv.FormatInt(result.ID, 10),
		RequestSummary: "创建域名: " + result.Domain,
		Success:      true,
	})
}

func (h *WebsiteDomainHandler) UpdateDomain(c *gin.Context) {
	domainIDStr := c.Param("domain_id")
	if domainIDStr == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "domain_id is required")
		return
	}

	domainID, err := strconv.ParseInt(domainIDStr, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid domain_id")
		return
	}

	var req dto.UpdateWebsiteDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.domainService.UpdateDomain(c.Request.Context(), domainID, req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)

	// Record audit log
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "website_domain",
		Action:       "update",
		TargetType:   "website_domain",
		TargetID:     strconv.FormatInt(domainID, 10),
		RequestSummary: "更新域名: " + result.Domain,
		Success:      true,
	})
}

func (h *WebsiteDomainHandler) DeleteDomain(c *gin.Context) {
	domainIDStr := c.Param("domain_id")
	if domainIDStr == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "domain_id is required")
		return
	}

	domainID, err := strconv.ParseInt(domainIDStr, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid domain_id")
		return
	}

	if err := h.domainService.DeleteDomain(c.Request.Context(), domainID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{})

	// Record audit log
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "website_domain",
		Action:       "delete",
		TargetType:   "website_domain",
		TargetID:     strconv.FormatInt(domainID, 10),
		RequestSummary: "删除域名",
		Success:      true,
	})
}
