package handler

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
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

	if rangeHeader := strings.TrimSpace(c.GetHeader("Range")); rangeHeader != "" && query.Offset == 0 && query.Limit == 0 {
		if rangeValue, ok := strings.CutPrefix(rangeHeader, "bytes="); ok {
			if start, ok := strings.CutSuffix(rangeValue, "-"); ok {
				if parsed, err := strconv.ParseUint(strings.TrimSpace(start), 10, 64); err == nil {
					query.Offset = parsed
				}
			}
		}
	}

	fileName := path.Base(strings.TrimSpace(query.Path))
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "download.bin"
	}

	var (
		result       dto.DownloadFileResult
		headersWritten bool
		bufferedChunks [][]byte
	)
	writeChunk := func(chunk []byte) error {
		copied := append([]byte(nil), chunk...)
		bufferedChunks = append(bufferedChunks, copied)
		return nil
	}

	var err error
	result, err = h.fileService.DownloadFile(c.Request.Context(), query, writeChunk)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "download",
			TargetType:     "path",
			TargetID:       query.Path,
			RequestSummary: fmt.Sprintf(`{"op":"download","offset":%d,"limit":%d}`, query.Offset, query.Limit),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	contentType := "application/octet-stream"
	if len(bufferedChunks) > 0 && len(bufferedChunks[0]) > 0 {
		sample := bufferedChunks[0]
		if len(sample) > 512 {
			sample = sample[:512]
		}
		contentType = http.DetectContentType(sample)
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", result.DownloadedBytes))
	if query.Offset > 0 || query.Limit > 0 {
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", result.StartOffset, result.EndOffset, result.TotalSize))
		c.Status(http.StatusPartialContent)
	} else {
		c.Status(http.StatusOK)
	}
	headersWritten = true

	for _, chunk := range bufferedChunks {
		if _, err := c.Writer.Write(chunk); err != nil {
			_ = c.Error(err)
			return
		}
	}

	if !headersWritten {
		c.Status(http.StatusOK)
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "files",
		Action:         "download",
		TargetType:     "path",
		TargetID:       query.Path,
		RequestSummary: fmt.Sprintf(`{"op":"download","offset":%d,"limit":%d}`, query.Offset, query.Limit),
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

	var req dto.UploadFileRequest
	req.Path = targetPath
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid upload params")
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
		req,
		file.Read,
	)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "files",
			Action:         "upload",
			TargetType:     "path",
			TargetID:       targetPath,
			RequestSummary: fmt.Sprintf(`{"op":"upload","filename":%q,"offset":%d}`, fileHeader.Filename, req.Offset),
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
		RequestSummary: fmt.Sprintf(`{"op":"upload","filename":%q,"offset":%d}`, fileHeader.Filename, req.Offset),
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
