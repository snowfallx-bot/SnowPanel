package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type AuditHandler struct {
	auditService service.AuditService
}

func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

func (h *AuditHandler) ListLogs(c *gin.Context) {
	var query dto.ListAuditLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	result, err := h.auditService.List(c.Request.Context(), query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *AuditHandler) ExportLogs(c *gin.Context) {
	var query dto.ListAuditLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	format := strings.ToLower(strings.TrimSpace(c.Query("format")))
	if format == "" {
		format = "csv"
	}
	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", exportDisposition("csv"))
	case "jsonl":
		c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
		c.Header("Content-Disposition", exportDisposition("jsonl"))
	default:
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "unsupported audit export format")
		return
	}

	if err := h.auditService.Export(c.Request.Context(), query, format, c.Writer); err != nil {
		response.FromError(c, err)
		return
	}
}

func (h *AuditHandler) CleanupRetention(c *gin.Context) {
	var req dto.AuditRetentionCleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.auditService.CleanupRetention(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "audit",
			Action:         "retention_cleanup",
			TargetType:     "audit_logs",
			RequestSummary: auditCleanupSummary(req),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "audit",
		Action:         "retention_cleanup",
		TargetType:     "audit_logs",
		RequestSummary: auditCleanupSummary(req),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "audit retention cleanup completed",
	})
	response.OK(c, result)
}

func exportDisposition(format string) string {
	return `attachment; filename="snowpanel-audit-` + time.Now().UTC().Format("20060102T150405Z") + `.` + format + `"`
}

func auditCleanupSummary(req dto.AuditRetentionCleanupRequest) string {
	return `{"dry_run":` + boolString(req.DryRun) +
		`,"retention_days":` + intString(req.RetentionDays) +
		`,"archive_before_delete":` + boolString(req.ArchiveBeforeDelete) + `}`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func intString(value int) string {
	return strconv.Itoa(value)
}
