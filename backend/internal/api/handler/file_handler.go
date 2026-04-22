package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type FileHandler struct {
	fileService  service.FileService
	auditService service.AuditService
}

func NewFileHandler(fileService service.FileService, auditService service.AuditService) *FileHandler {
	return &FileHandler{
		fileService:  fileService,
		auditService: auditService,
	}
}

func (h *FileHandler) ListFiles(c *gin.Context) {
	var query dto.ListFilesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	result, err := h.fileService.ListFiles(c.Request.Context(), query)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "list",
			TargetType:     "path",
			TargetID:       query.Path,
			RequestSummary: `{"op":"list"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "list",
		TargetType:     "path",
		TargetID:       query.Path,
		RequestSummary: `{"op":"list"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "list success",
	})
	response.OK(c, result)
}

func (h *FileHandler) ReadTextFile(c *gin.Context) {
	var req dto.ReadTextFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	result, err := h.fileService.ReadTextFile(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "read",
			TargetType:     "path",
			TargetID:       req.Path,
			RequestSummary: `{"op":"read"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "read",
		TargetType:     "path",
		TargetID:       req.Path,
		RequestSummary: `{"op":"read"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "read success",
	})
	response.OK(c, result)
}

func (h *FileHandler) WriteTextFile(c *gin.Context) {
	var req dto.WriteTextFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	result, err := h.fileService.WriteTextFile(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "write",
			TargetType:     "path",
			TargetID:       req.Path,
			RequestSummary: `{"op":"write"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "write",
		TargetType:     "path",
		TargetID:       req.Path,
		RequestSummary: `{"op":"write"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "write success",
	})
	response.OK(c, result)
}

func (h *FileHandler) CreateDirectory(c *gin.Context) {
	var req dto.CreateDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	result, err := h.fileService.CreateDirectory(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "mkdir",
			TargetType:     "path",
			TargetID:       req.Path,
			RequestSummary: `{"op":"mkdir"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "mkdir",
		TargetType:     "path",
		TargetID:       req.Path,
		RequestSummary: `{"op":"mkdir"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "mkdir success",
	})
	response.OK(c, result)
}

func (h *FileHandler) DeleteFile(c *gin.Context) {
	var req dto.DeleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	result, err := h.fileService.DeleteFile(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "delete",
			TargetType:     "path",
			TargetID:       req.Path,
			RequestSummary: `{"op":"delete"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "delete",
		TargetType:     "path",
		TargetID:       req.Path,
		RequestSummary: `{"op":"delete"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "delete success",
	})
	response.OK(c, result)
}
