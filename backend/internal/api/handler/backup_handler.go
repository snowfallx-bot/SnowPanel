package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type BackupHandler struct {
	backupService service.BackupService
	taskService   service.TaskService
	auditService  service.AuditService
}

func NewBackupHandler(
	backupService service.BackupService,
	taskService service.TaskService,
	auditService service.AuditService,
) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		taskService:   taskService,
		auditService:  auditService,
	}
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	var query dto.ListBackupsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	result, err := h.backupService.List(c.Request.Context(), query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var req dto.CreateBackupMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}

	result, err := h.backupService.CreateMetadata(c.Request.Context(), req, userIDPtr)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "backups",
			Action:         "create",
			TargetType:     "backup",
			TargetID:       req.ResourceID,
			RequestSummary: backupCreateSummary(req),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "backups",
		Action:         "create",
		TargetType:     "backup",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: backupCreateSummary(req),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "backup metadata created",
	})
	response.OK(c, result)
}

func (h *BackupHandler) CreateBackupTask(c *gin.Context) {
	var req dto.CreateBackupTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}
	username, _ := middleware.GetCurrentUsername(c)

	result, err := h.taskService.CreateBackupTask(c.Request.Context(), req, userIDPtr, username)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "backups",
			Action:         "create_task",
			TargetType:     "backup",
			TargetID:       req.ResourceID,
			RequestSummary: backupTaskCreateSummary(req),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "backups",
		Action:         "create_task",
		TargetType:     "backup",
		TargetID:       strconv.FormatInt(result.Backup.ID, 10),
		RequestSummary: backupTaskCreateSummary(req),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "backup task created",
	})
	response.OK(c, result)
}

func (h *BackupHandler) VerifyBackup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid backup id")
		return
	}

	var req dto.VerifyBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, svcErr := h.backupService.Verify(c.Request.Context(), id, req)
	if svcErr != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "backups",
			Action:         "verify",
			TargetType:     "backup",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: backupVerifySummary(req),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  svcErr.Error(),
		})
		response.FromError(c, svcErr)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "backups",
		Action:         "verify",
		TargetType:     "backup",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: backupVerifySummary(req),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "backup metadata verified",
	})
	response.OK(c, result)
}

func (h *BackupHandler) VerifyBackupTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid backup id")
		return
	}

	var req dto.CreateBackupVerifyTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}
	username, _ := middleware.GetCurrentUsername(c)

	result, svcErr := h.taskService.CreateBackupVerifyTask(c.Request.Context(), id, req, userIDPtr, username)
	if svcErr != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "backups",
			Action:         "verify_task",
			TargetType:     "backup",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: backupVerifyTaskSummary(req),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  svcErr.Error(),
		})
		response.FromError(c, svcErr)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "backups",
		Action:         "verify_task",
		TargetType:     "backup",
		TargetID:       strconv.FormatInt(id, 10),
		RequestSummary: backupVerifyTaskSummary(req),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "backup verify task created",
	})
	response.OK(c, result)
}

func (h *BackupHandler) CleanupRetention(c *gin.Context) {
	var req dto.BackupRetentionCleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.backupService.CleanupRetention(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "backups",
			Action:         "retention_cleanup",
			TargetType:     "backup",
			RequestSummary: backupCleanupSummary(req),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "backups",
		Action:         "retention_cleanup",
		TargetType:     "backup",
		RequestSummary: backupCleanupSummary(req),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "backup retention cleanup completed",
	})
	response.OK(c, result)
}

func backupCreateSummary(req dto.CreateBackupMetadataRequest) string {
	return fmt.Sprintf(
		`{"resource_type":%q,"resource_id":%q,"storage_type":%q}`,
		req.ResourceType,
		req.ResourceID,
		req.StorageType,
	)
}

func backupTaskCreateSummary(req dto.CreateBackupTaskRequest) string {
	return fmt.Sprintf(
		`{"resource_type":%q,"resource_id":%q,"storage_type":%q}`,
		req.ResourceType,
		req.ResourceID,
		req.StorageType,
	)
}

func backupVerifySummary(req dto.VerifyBackupRequest) string {
	return fmt.Sprintf(
		`{"size_bytes":%d,"checksum":%q}`,
		req.SizeBytes,
		req.Checksum,
	)
}

func backupVerifyTaskSummary(req dto.CreateBackupVerifyTaskRequest) string {
	return fmt.Sprintf(
		`{"size_bytes":%d,"checksum":%q}`,
		req.SizeBytes,
		req.Checksum,
	)
}

func backupCleanupSummary(req dto.BackupRetentionCleanupRequest) string {
	return fmt.Sprintf(
		`{"dry_run":%t,"retention_days":%d,"archive_before_delete":%t}`,
		req.DryRun,
		req.RetentionDays,
		req.ArchiveBeforeDelete,
	)
}
