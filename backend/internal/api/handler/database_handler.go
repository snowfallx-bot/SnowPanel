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

type DatabaseHandler struct {
	databaseService service.DatabaseService
	auditService    service.AuditService
}

func NewDatabaseHandler(databaseService service.DatabaseService, auditService service.AuditService) *DatabaseHandler {
	return &DatabaseHandler{
		databaseService: databaseService,
		auditService:    auditService,
	}
}

func (h *DatabaseHandler) ListDatabaseInstances(c *gin.Context) {
	result, err := h.databaseService.ListDatabaseInstances(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.ListDatabaseInstancesResponse{Items: result})
}

func (h *DatabaseHandler) GetDatabaseInstance(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "database instance id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid database instance id")
		return
	}

	result, err := h.databaseService.GetDatabaseInstance(c.Request.Context(), idInt)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *DatabaseHandler) CreateDatabaseInstance(c *gin.Context) {
	var req dto.CreateDatabaseInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.databaseService.CreateDatabaseInstance(c.Request.Context(), req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)

	// 记录审计日志
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "database_instance",
		Action:       "create",
		TargetType:   "database_instance",
		TargetID:     strconv.FormatInt(result.ID, 10),
		RequestSummary: "创建数据库实例: " + result.Name,
		Success:      true,
	})
}

func (h *DatabaseHandler) UpdateDatabaseInstance(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "database instance id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid database instance id")
		return
	}

	var req dto.UpdateDatabaseInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.databaseService.UpdateDatabaseInstance(c.Request.Context(), idInt, req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)

	// 记录审计日志
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "database_instance",
		Action:       "update",
		TargetType:   "database_instance",
		TargetID:     strconv.FormatInt(idInt, 10),
		RequestSummary: "更新数据库实例: " + result.Name,
		Success:      true,
	})
}

func (h *DatabaseHandler) DeleteDatabaseInstance(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "database instance id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid database instance id")
		return
	}

	if err := h.databaseService.DeleteDatabaseInstance(c.Request.Context(), idInt); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{})

	// 记录审计日志
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "database_instance",
		Action:       "delete",
		TargetType:   "database_instance",
		TargetID:     strconv.FormatInt(idInt, 10),
		RequestSummary: "删除数据库实例",
		Success:      true,
	})
}

func (h *DatabaseHandler) TestConnection(c *gin.Context) {
	var req dto.CreateDatabaseInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	result, err := h.databaseService.TestConnection(c.Request.Context(), req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *DatabaseHandler) ListDatabases(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "database instance id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid database instance id")
		return
	}

	result, err := h.databaseService.ListDatabases(c.Request.Context(), idInt)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.ListDatabasesResponse{Items: result})
}

func (h *DatabaseHandler) CreateDatabase(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "database instance id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid database instance id")
		return
	}

	var req dto.CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}
	req.InstanceID = idInt

	if err := h.databaseService.CreateDatabase(c.Request.Context(), req); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "database created successfully"})

	// 记录审计日志
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "database",
		Action:       "create",
		TargetType:   "database",
		TargetID:     req.Name,
		RequestSummary: "创建数据库: " + req.Name,
		Success:      true,
	})
}

func (h *DatabaseHandler) DeleteDatabase(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "database instance id is required")
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid database instance id")
		return
	}

	var req dto.DeleteDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}
	req.InstanceID = idInt

	if err := h.databaseService.DeleteDatabase(c.Request.Context(), req); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{})

	// 记录审计日志
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:       "database",
		Action:       "delete",
		TargetType:   "database",
		TargetID:     req.Name,
		RequestSummary: "删除数据库: " + req.Name,
		Success:      true,
	})
}
