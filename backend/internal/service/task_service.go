package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

const (
	TaskStatusPending  = "pending"
	TaskStatusRunning  = "running"
	TaskStatusSuccess  = "success"
	TaskStatusFailed   = "failed"
	TaskStatusCanceled = "canceled"

	TaskTypeDockerRestart  = "docker_restart"
	TaskTypeServiceRestart = "service_restart"

	taskOperationDockerRestart  = "docker.restart"
	taskOperationServiceRestart = "service.restart"
)

type TaskService interface {
	CreateDockerRestartTask(
		ctx context.Context,
		req dto.CreateDockerRestartTaskRequest,
		triggeredBy *int64,
		username string,
	) (dto.CreateTaskResult, error)
	CreateServiceRestartTask(
		ctx context.Context,
		req dto.CreateServiceRestartTaskRequest,
		triggeredBy *int64,
		username string,
	) (dto.CreateTaskResult, error)
	CancelTask(ctx context.Context, id int64, username string) error
	RetryTask(ctx context.Context, id int64, triggeredBy *int64, username string) (dto.CreateTaskResult, error)
	ListTasks(ctx context.Context, query dto.ListTasksQuery) (dto.ListTasksResult, error)
	GetTaskDetail(ctx context.Context, id int64) (dto.TaskDetail, error)
}

type taskService struct {
	repo           repository.TaskRepository
	dockerService  DockerService
	serviceManager ServiceManagerService
}

type taskPayload struct {
	Operation   string `json:"operation"`
	ContainerID string `json:"container_id,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}

func NewTaskService(
	repo repository.TaskRepository,
	dockerService DockerService,
	serviceManager ServiceManagerService,
) TaskService {
	return &taskService{
		repo:           repo,
		dockerService:  dockerService,
		serviceManager: serviceManager,
	}
}

func (s *taskService) CreateDockerRestartTask(
	ctx context.Context,
	req dto.CreateDockerRestartTaskRequest,
	triggeredBy *int64,
	username string,
) (dto.CreateTaskResult, error) {
	containerID := strings.TrimSpace(req.ContainerID)
	if containerID == "" {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("container_id is required"),
		)
	}

	if s.dockerService == nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			errors.New("docker service is not configured"),
		)
	}

	return s.createAndRunTask(
		ctx,
		TaskTypeDockerRestart,
		taskPayload{
			Operation:   taskOperationDockerRestart,
			ContainerID: containerID,
		},
		triggeredBy,
		username,
	)
}

func (s *taskService) CreateServiceRestartTask(
	ctx context.Context,
	req dto.CreateServiceRestartTaskRequest,
	triggeredBy *int64,
	username string,
) (dto.CreateTaskResult, error) {
	serviceName := strings.TrimSpace(req.ServiceName)
	if serviceName == "" {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("service_name is required"),
		)
	}

	if s.serviceManager == nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			errors.New("service manager is not configured"),
		)
	}

	return s.createAndRunTask(
		ctx,
		TaskTypeServiceRestart,
		taskPayload{
			Operation:   taskOperationServiceRestart,
			ServiceName: serviceName,
		},
		triggeredBy,
		username,
	)
}

func (s *taskService) CancelTask(ctx context.Context, id int64, username string) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if task == nil {
		return apperror.ErrTaskNotFound
	}

	switch task.Status {
	case TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled:
		return apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			fmt.Errorf("task is already terminal with status '%s'", task.Status),
		)
	}

	if err := s.repo.UpdateStatus(
		ctx,
		id,
		TaskStatusCanceled,
		task.Progress,
		"canceled by operator",
	); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:   id,
		Level:    "warn",
		Message:  "task canceled",
		Metadata: marshalTaskMetadata(map[string]interface{}{"actor": username}),
	})

	return nil
}

func (s *taskService) RetryTask(
	ctx context.Context,
	id int64,
	triggeredBy *int64,
	username string,
) (dto.CreateTaskResult, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if task == nil {
		return dto.CreateTaskResult{}, apperror.ErrTaskNotFound
	}
	if task.Status != TaskStatusFailed && task.Status != TaskStatusCanceled {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("only failed or canceled tasks can be retried"),
		)
	}

	payload, payloadErr := unmarshalTaskPayload(task.Payload)
	if payloadErr != nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			payloadErr,
		)
	}

	result, err := s.createAndRunTask(ctx, task.Type, payload, triggeredBy, username)
	if err != nil {
		return dto.CreateTaskResult{}, err
	}

	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: "task retried",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"actor":         username,
			"retry_task_id": result.ID,
		}),
	})

	return result, nil
}

func (s *taskService) ListTasks(ctx context.Context, query dto.ListTasksQuery) (dto.ListTasksResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 20
	}

	items, total, err := s.repo.List(ctx, repository.TaskListFilter{
		Page: page,
		Size: size,
	})
	if err != nil {
		return dto.ListTasksResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.TaskSummary, 0, len(items))
	for _, item := range items {
		result = append(result, mapTaskSummary(item))
	}

	return dto.ListTasksResult{
		Page:  page,
		Size:  size,
		Total: total,
		Items: result,
	}, nil
}

func (s *taskService) GetTaskDetail(ctx context.Context, id int64) (dto.TaskDetail, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.TaskDetail{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if task == nil {
		return dto.TaskDetail{}, apperror.ErrTaskNotFound
	}

	logs, err := s.repo.ListLogs(ctx, id, 200)
	if err != nil {
		return dto.TaskDetail{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	logItems := make([]dto.TaskLog, 0, len(logs))
	for _, log := range logs {
		logItems = append(logItems, dto.TaskLog{
			ID:        log.ID,
			Level:     log.Level,
			Message:   log.Message,
			Metadata:  log.Metadata,
			CreatedAt: log.CreatedAt.Format(time.RFC3339),
		})
	}

	return dto.TaskDetail{
		Summary: mapTaskSummary(*task),
		Logs:    logItems,
	}, nil
}

func (s *taskService) createAndRunTask(
	ctx context.Context,
	taskType string,
	payload taskPayload,
	triggeredBy *int64,
	username string,
) (dto.CreateTaskResult, error) {
	task := &model.Task{
		Type:        taskType,
		Status:      TaskStatusPending,
		Progress:    0,
		Payload:     marshalTaskPayload(payload),
		Result:      `{}`,
		ErrorMsg:    "",
		TriggeredBy: triggeredBy,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: "task queued",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"actor":    username,
			"type":     taskType,
			"payload":  payload,
			"status":   TaskStatusPending,
			"progress": 0,
		}),
	})

	go s.runTask(task.ID, payload)

	return dto.CreateTaskResult{
		ID:     task.ID,
		Type:   task.Type,
		Status: task.Status,
	}, nil
}

func (s *taskService) runTask(taskID int64, payload taskPayload) {
	ctx := context.Background()

	if s.isCanceled(ctx, taskID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   taskID,
			Level:    "warn",
			Message:  "task canceled before execution",
			Metadata: "{}",
		})
		return
	}

	if !s.setRunningProgress(ctx, taskID, 5) {
		return
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  taskID,
		Level:   "info",
		Message: "task execution started",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"operation": payload.Operation,
			"progress":  5,
		}),
	})

	if s.isCanceled(ctx, taskID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   taskID,
			Level:    "warn",
			Message:  "task canceled before operation",
			Metadata: "{}",
		})
		return
	}

	switch payload.Operation {
	case taskOperationDockerRestart:
		if !s.setRunningProgress(ctx, taskID, 30) {
			_ = s.repo.AppendLog(ctx, &model.TaskLog{
				TaskID:   taskID,
				Level:    "warn",
				Message:  "task canceled before docker restart",
				Metadata: "{}",
			})
			return
		}
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:  taskID,
			Level:   "info",
			Message: "restarting docker container",
			Metadata: marshalTaskMetadata(map[string]interface{}{
				"container_id": payload.ContainerID,
				"progress":     30,
			}),
		})
		result, err := s.dockerService.RestartContainer(ctx, payload.ContainerID)
		if err != nil {
			s.markTaskFailed(ctx, taskID, err, map[string]interface{}{
				"operation":    payload.Operation,
				"container_id": payload.ContainerID,
			})
			return
		}
		if !s.setRunningProgress(ctx, taskID, 85) {
			_ = s.repo.AppendLog(ctx, &model.TaskLog{
				TaskID:   taskID,
				Level:    "warn",
				Message:  "task canceled after docker restart",
				Metadata: "{}",
			})
			return
		}
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:  taskID,
			Level:   "info",
			Message: "docker container restarted",
			Metadata: marshalTaskMetadata(map[string]interface{}{
				"container_id": result.ID,
				"state":        result.State,
				"progress":     85,
			}),
		})
	case taskOperationServiceRestart:
		if !s.setRunningProgress(ctx, taskID, 30) {
			_ = s.repo.AppendLog(ctx, &model.TaskLog{
				TaskID:   taskID,
				Level:    "warn",
				Message:  "task canceled before service restart",
				Metadata: "{}",
			})
			return
		}
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:  taskID,
			Level:   "info",
			Message: "restarting system service",
			Metadata: marshalTaskMetadata(map[string]interface{}{
				"service_name": payload.ServiceName,
				"progress":     30,
			}),
		})
		result, err := s.serviceManager.RestartService(ctx, payload.ServiceName)
		if err != nil {
			s.markTaskFailed(ctx, taskID, err, map[string]interface{}{
				"operation":    payload.Operation,
				"service_name": payload.ServiceName,
			})
			return
		}
		if !s.setRunningProgress(ctx, taskID, 85) {
			_ = s.repo.AppendLog(ctx, &model.TaskLog{
				TaskID:   taskID,
				Level:    "warn",
				Message:  "task canceled after service restart",
				Metadata: "{}",
			})
			return
		}
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:  taskID,
			Level:   "info",
			Message: "system service restarted",
			Metadata: marshalTaskMetadata(map[string]interface{}{
				"service_name": result.Name,
				"status":       result.Status,
				"progress":     85,
			}),
		})
	default:
		s.markTaskFailed(ctx, taskID, errors.New("unsupported task operation"), map[string]interface{}{
			"operation": payload.Operation,
		})
		return
	}

	if s.isCanceled(ctx, taskID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   taskID,
			Level:    "warn",
			Message:  "task canceled after operation",
			Metadata: "{}",
		})
		return
	}

	_ = s.repo.UpdateStatus(ctx, taskID, TaskStatusSuccess, 100, "")
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  taskID,
		Level:   "info",
		Message: "task completed",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"operation": payload.Operation,
			"status":    TaskStatusSuccess,
			"progress":  100,
		}),
	})
}

func mapTaskSummary(task model.Task) dto.TaskSummary {
	return dto.TaskSummary{
		ID:          task.ID,
		Type:        task.Type,
		Status:      task.Status,
		Progress:    task.Progress,
		Error:       task.ErrorMsg,
		TriggeredBy: task.TriggeredBy,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	}
}

func marshalTaskMetadata(data map[string]interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func marshalTaskPayload(payload taskPayload) string {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func unmarshalTaskPayload(raw string) (taskPayload, error) {
	var payload taskPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return taskPayload{}, fmt.Errorf("invalid task payload: %w", err)
	}

	switch payload.Operation {
	case taskOperationDockerRestart:
		if strings.TrimSpace(payload.ContainerID) == "" {
			return taskPayload{}, errors.New("container_id is required in task payload")
		}
	case taskOperationServiceRestart:
		if strings.TrimSpace(payload.ServiceName) == "" {
			return taskPayload{}, errors.New("service_name is required in task payload")
		}
	default:
		return taskPayload{}, fmt.Errorf("unsupported operation '%s'", payload.Operation)
	}

	return payload, nil
}

func (s *taskService) isCanceled(ctx context.Context, taskID int64) bool {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil || task == nil {
		return false
	}
	return task.Status == TaskStatusCanceled
}

func (s *taskService) markTaskFailed(
	ctx context.Context,
	taskID int64,
	err error,
	metadata map[string]interface{},
) {
	if s.isCanceled(ctx, taskID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   taskID,
			Level:    "warn",
			Message:  "task canceled while operation was running",
			Metadata: "{}",
		})
		return
	}

	_ = s.repo.UpdateStatus(ctx, taskID, TaskStatusFailed, 100, err.Error())

	fields := map[string]interface{}{
		"error": err.Error(),
	}
	for key, value := range metadata {
		fields[key] = value
	}

	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:   taskID,
		Level:    "error",
		Message:  "task failed",
		Metadata: marshalTaskMetadata(fields),
	})
}

func (s *taskService) setRunningProgress(ctx context.Context, taskID int64, progress int) bool {
	if s.isCanceled(ctx, taskID) {
		return false
	}
	if err := s.repo.UpdateStatus(ctx, taskID, TaskStatusRunning, progress, ""); err != nil {
		return false
	}
	return true
}
