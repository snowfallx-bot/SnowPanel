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
	fileService service.FileService
}

func NewFileHandler(fileService service.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}
