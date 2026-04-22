package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type AuthHandler struct {
	authService  service.AuthService
	auditService service.AuditService
}

func NewAuthHandler(authService service.AuthService, auditService service.AuditService) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		auditService: auditService,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid login payload")
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Username:       req.Username,
			Module:         "auth",
			Action:         "login",
			TargetType:     "user",
			TargetID:       req.Username,
			RequestSummary: `{"endpoint":"/api/v1/auth/login"}`,
			Success:        false,
			ResultCode:     "login_failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Username:       req.Username,
		Module:         "auth",
		Action:         "login",
		TargetType:     "user",
		TargetID:       req.Username,
		RequestSummary: `{"endpoint":"/api/v1/auth/login"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "login success",
	})
	response.OK(c, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, apperror.ErrUnauthorized.Code, apperror.ErrUnauthorized.Message)
		return
	}

	profile, err := h.authService.Me(c.Request.Context(), userID)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, profile)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, apperror.ErrUnauthorized.Code, apperror.ErrUnauthorized.Message)
		return
	}

	username, _ := middleware.GetCurrentUsername(c)

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid change password payload")
		return
	}

	resp, err := h.authService.ChangePassword(c.Request.Context(), userID, req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			UserID:         &userID,
			Username:       username,
			Module:         "auth",
			Action:         "change_password",
			TargetType:     "user",
			TargetID:       username,
			RequestSummary: `{"endpoint":"/api/v1/auth/change-password"}`,
			Success:        false,
			ResultCode:     "password_change_failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		UserID:         &userID,
		Username:       username,
		Module:         "auth",
		Action:         "change_password",
		TargetType:     "user",
		TargetID:       username,
		RequestSummary: `{"endpoint":"/api/v1/auth/change-password"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "password changed",
	})

	response.OK(c, resp)
}
