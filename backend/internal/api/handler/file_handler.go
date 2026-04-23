package handler

import (
	"fmt"
	"net/http"
	"path"
	"strings"

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

func (h *FileHandler) DownloadFile(c *gin.Context) {
	var query dto.DownloadFileQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	fileName := path.Base(strings.TrimSpace(query.Path))
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "download.bin"
	}

	headersWritten := false
	writeChunk := func(chunk []byte) error {
		if !headersWritten {
			contentType := "application/octet-stream"
			if len(chunk) > 0 {
				sample := chunk
				if len(sample) > 512 {
					sample = sample[:512]
				}
				contentType = http.DetectContentType(sample)
			}
			c.Header("Content-Type", contentType)
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
			c.Header("X-Content-Type-Options", "nosniff")
			headersWritten = true
		}
		_, err := c.Writer.Write(chunk)
		return err
	}

	_, err := h.fileService.DownloadFile(c.Request.Context(), query, writeChunk)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "download",
			TargetType:     "path",
			TargetID:       query.Path,
			RequestSummary: `{"op":"download"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		if !headersWritten {
			response.FromError(c, err)
		} else {
			_ = c.Error(err)
		}
		return
	}

	if !headersWritten {
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
		c.Header("X-Content-Type-Options", "nosniff")
		c.Status(http.StatusOK)
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "download",
		TargetType:     "path",
		TargetID:       query.Path,
		RequestSummary: `{"op":"download"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "download success",
	})
}

func (h *FileHandler) UploadFile(c *gin.Context) {
	targetPath := strings.TrimSpace(c.PostForm("path"))
	if targetPath == "" {
		targetPath = strings.TrimSpace(c.Query("path"))
	}
	if targetPath == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "path is required")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "file is required")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "unable to open uploaded file")
		return
	}
	defer file.Close()

	result, err := h.fileService.UploadFile(
		c.Request.Context(),
		dto.UploadFileRequest{Path: targetPath},
		file.Read,
	)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "upload",
			TargetType:     "path",
			TargetID:       targetPath,
			RequestSummary: fmt.Sprintf(`{"op":"upload","filename":%q}`, fileHeader.Filename),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "upload",
		TargetType:     "path",
		TargetID:       targetPath,
		RequestSummary: fmt.Sprintf(`{"op":"upload","filename":%q}`, fileHeader.Filename),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "upload success",
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

func (h *FileHandler) RenameFile(c *gin.Context) {
	var req dto.RenameFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	result, err := h.fileService.RenameFile(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "rename",
			TargetType:     "path",
			TargetID:       req.SourcePath,
			RequestSummary: `{"op":"rename"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "rename",
		TargetType:     "path",
		TargetID:       req.SourcePath,
		RequestSummary: `{"op":"rename"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "rename success",
	})
	response.OK(c, result)
}
