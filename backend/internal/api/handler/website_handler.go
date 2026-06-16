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

type WebsiteHandler struct {
	websiteService service.WebsiteService
	auditService   service.AuditService
}

func NewWebsiteHandler(websiteService service.WebsiteService, auditService service.AuditService) *WebsiteHandler {
	return &WebsiteHandler{
		websiteService: websiteService,
		auditService:   auditService,
	}
}

func (h *WebsiteHandler) ListWebsites(c *gin.Context) {
	result, err := h.websiteService.ListWebsites(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.ListWebsitesResponse{Items: result})
}

func (h *WebsiteHandler) GetWebsite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website id")
		return
	}

	result, err := h.websiteService.GetWebsite(c.Request.Context(), idInt)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *WebsiteHandler) CreateWebsite(c *gin.Context) {
	var req dto.CreateWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.websiteService.CreateWebsite(c.Request.Context(), req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *WebsiteHandler) UpdateWebsite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website id")
		return
	}

	var req dto.UpdateWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.websiteService.UpdateWebsite(c.Request.Context(), idInt, req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *WebsiteHandler) DeleteWebsite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website id")
		return
	}

	if err := h.websiteService.DeleteWebsite(c.Request.Context(), idInt); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{})
}

func (h *WebsiteHandler) EnableWebsite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website id")
		return
	}

	if err := h.websiteService.EnableWebsite(c.Request.Context(), idInt); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "website enabled"})
}

func (h *WebsiteHandler) DisableWebsite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "website id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid website id")
		return
	}

	if err := h.websiteService.DisableWebsite(c.Request.Context(), idInt); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "website disabled"})
}
