package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	appmetrics "github.com/snowfallx-bot/SnowPanel/backend/internal/metrics"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
)

const (
	TaskStatusPending  = "pending"
	TaskStatusRunning  = "running"
	TaskStatusSuccess  = "success"
	TaskStatusFailed   = "failed"
	TaskStatusCanceled = "canceled"

	TaskTypeDockerRestart  = "docker_restart"
	TaskTypeServiceRestart = "service_restart"
	TaskTypeBackupCreate   = "backup_create"
	TaskTypeBackupVerify   = "backup_verify"

	taskOperationDockerRestart  = "docker.restart"
	taskOperationServiceRestart = "service.restart"
	taskOperationBackupCreate   = "backup.create"
	taskOperationBackupVerify   = "backup.verify"
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
	CreateBackupTask(
		ctx context.Context,
		req dto.CreateBackupTaskRequest,
		triggeredBy *int64,
		username string,
	) (dto.CreateBackupTaskResult, error)
	CreateBackupVerifyTask(
		ctx context.Context,
		backupID int64,
		req dto.CreateBackupVerifyTaskRequest,
		triggeredBy *int64,
		username string,
	) (dto.CreateBackupVerifyTaskResult, error)
	CancelTask(ctx context.Context, id int64, username string) error
	RetryTask(ctx context.Context, id int64, triggeredBy *int64, username string) (dto.CreateTaskResult, error)
	ListTasks(ctx context.Context, query dto.ListTasksQuery) (dto.ListTasksResult, error)
	GetTaskDetail(ctx context.Context, id int64) (dto.TaskDetail, error)
	RunWorker(ctx context.Context, options TaskWorkerOptions)
}

type taskService struct {
	repo           repository.TaskRepository
	dockerService  DockerService
	serviceManager ServiceManagerService
	backupService  BackupService
	options        TaskServiceOptions
	metrics        *appmetrics.Set
}

type TaskServiceOptions struct {
	AsyncExecution bool
	MaxAttempts    int
	Metrics        *appmetrics.Set
	BackupService  BackupService
}

type TaskWorkerOptions struct {
	WorkerID      string
	Concurrency   int
	LeaseDuration time.Duration
	PollInterval  time.Duration
}

type taskPayload struct {
	Operation    string `json:"operation"`
	ContainerID  string `json:"container_id,omitempty"`
	ServiceName  string `json:"service_name,omitempty"`
	BackupID     int64  `json:"backup_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	StorageType  string `json:"storage_type,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	Checksum     string `json:"checksum,omitempty"`
}

type nonRetryableTaskError struct {
	err error
}

func (e nonRetryableTaskError) Error() string {
	return e.err.Error()
}

func (e nonRetryableTaskError) Unwrap() error {
	return e.err
}

func NewTaskService(
	repo repository.TaskRepository,
	dockerService DockerService,
	serviceManager ServiceManagerService,
) TaskService {
	return NewTaskServiceWithOptions(repo, dockerService, serviceManager, TaskServiceOptions{
		AsyncExecution: true,
		MaxAttempts:    1,
	})
}

func NewTaskServiceWithOptions(
	repo repository.TaskRepository,
	dockerService DockerService,
	serviceManager ServiceManagerService,
	options TaskServiceOptions,
) TaskService {
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = 1
	}
	return &taskService{
		repo:           repo,
		dockerService:  dockerService,
		serviceManager: serviceManager,
		backupService:  options.BackupService,
		options:        options,
		metrics:        options.Metrics,
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
	idempotencyKey, err := normalizeTaskIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			err,
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
		idempotencyKey,
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
	idempotencyKey, err := normalizeTaskIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return dto.CreateTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			err,
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
		idempotencyKey,
	)
}

func (s *taskService) CreateBackupTask(
	ctx context.Context,
	req dto.CreateBackupTaskRequest,
	triggeredBy *int64,
	username string,
) (dto.CreateBackupTaskResult, error) {
	if s.backupService == nil {
		return dto.CreateBackupTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			errors.New("backup service is not configured"),
		)
	}
	idempotencyKey, err := normalizeTaskIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return dto.CreateBackupTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			err,
		)
	}

	backup, err := s.backupService.CreateMetadata(ctx, dto.CreateBackupMetadataRequest{
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		StorageType:  req.StorageType,
		FilePath:     req.FilePath,
	}, triggeredBy)
	if err != nil {
		return dto.CreateBackupTaskResult{}, err
	}

	task, err := s.createAndRunTask(
		ctx,
		TaskTypeBackupCreate,
		taskPayload{
			Operation:    taskOperationBackupCreate,
			BackupID:     backup.ID,
			ResourceType: backup.ResourceType,
			ResourceID:   backup.ResourceID,
			StorageType:  backup.StorageType,
			FilePath:     backup.FilePath,
		},
		triggeredBy,
		username,
		idempotencyKey,
	)
	if err != nil {
		_, _ = s.backupService.MarkStatus(ctx, backup.ID, BackupStatusFailed)
		return dto.CreateBackupTaskResult{}, err
	}
	return dto.CreateBackupTaskResult{Backup: backup, Task: task}, nil
}

func (s *taskService) CreateBackupVerifyTask(
	ctx context.Context,
	backupID int64,
	req dto.CreateBackupVerifyTaskRequest,
	triggeredBy *int64,
	username string,
) (dto.CreateBackupVerifyTaskResult, error) {
	if backupID <= 0 {
		return dto.CreateBackupVerifyTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("backup id must be positive"),
		)
	}
	if s.backupService == nil {
		return dto.CreateBackupVerifyTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			errors.New("backup service is not configured"),
		)
	}
	idempotencyKey, err := normalizeTaskIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return dto.CreateBackupVerifyTaskResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			err,
		)
	}

	task, err := s.createAndRunTask(
		ctx,
		TaskTypeBackupVerify,
		taskPayload{
			Operation: taskOperationBackupVerify,
			BackupID:  backupID,
			SizeBytes: req.SizeBytes,
			Checksum:  req.Checksum,
			FilePath:  req.FilePath,
		},
		triggeredBy,
		username,
		idempotencyKey,
	)
	if err != nil {
		return dto.CreateBackupVerifyTaskResult{}, err
	}
	return dto.CreateBackupVerifyTaskResult{Task: task}, nil
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

	result, err := s.createAndRunTask(ctx, task.Type, payload, triggeredBy, username, nil)
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
		Page:   page,
		Size:   size,
		Status: query.Status,
		Type:   query.Type,
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
	idempotencyKey *string,
) (dto.CreateTaskResult, error) {
	if idempotencyKey != nil {
		existing, err := s.repo.GetByIdempotencyKey(ctx, *idempotencyKey)
		if err != nil {
			return dto.CreateTaskResult{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}
		if existing != nil {
			return dto.CreateTaskResult{
				ID:     existing.ID,
				Type:   existing.Type,
				Status: existing.Status,
			}, nil
		}
	}

	task := &model.Task{
		Type:           taskType,
		Status:         TaskStatusPending,
		Progress:       0,
		Payload:        marshalTaskPayload(payload),
		Result:         `{}`,
		ErrorMsg:       "",
		TriggeredBy:    triggeredBy,
		IdempotencyKey: idempotencyKey,
		MaxAttempts:    s.options.MaxAttempts,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		if idempotencyKey != nil {
			existing, findErr := s.repo.GetByIdempotencyKey(ctx, *idempotencyKey)
			if findErr == nil && existing != nil {
				return dto.CreateTaskResult{
					ID:     existing.ID,
					Type:   existing.Type,
					Status: existing.Status,
				}, nil
			}
		}
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

	if s.options.AsyncExecution {
		go s.runTask(task.ID, payload)
	}

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
	case taskOperationBackupCreate:
		if err := s.executeBackupCreate(ctx, taskID, payload, ""); err != nil {
			s.markTaskFailed(ctx, taskID, err, map[string]interface{}{
				"operation": payload.Operation,
				"backup_id": payload.BackupID,
			})
			return
		}
	case taskOperationBackupVerify:
		if err := s.executeBackupVerify(ctx, taskID, payload, ""); err != nil {
			s.markTaskFailed(ctx, taskID, err, map[string]interface{}{
				"operation": payload.Operation,
				"backup_id": payload.BackupID,
			})
			return
		}
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

func (s *taskService) RunWorker(ctx context.Context, options TaskWorkerOptions) {
	options = normalizeTaskWorkerOptions(options)
	workers := make(chan struct{}, options.Concurrency)
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		_, _ = s.repo.ReleaseStaleTasks(ctx, time.Now())
		s.refreshTaskGauges(ctx)
		s.claimAvailableTasks(ctx, options, workers)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *taskService) claimAvailableTasks(
	ctx context.Context,
	options TaskWorkerOptions,
	workers chan struct{},
) {
	for {
		select {
		case workers <- struct{}{}:
		default:
			return
		}

		task, err := s.repo.ClaimNextTask(ctx, options.WorkerID, options.LeaseDuration)
		if err != nil {
			s.metrics.ObserveTaskWorkerClaim("error")
			<-workers
			return
		}
		if task == nil {
			s.metrics.ObserveTaskWorkerClaim("empty")
			<-workers
			return
		}
		s.metrics.ObserveTaskWorkerClaim("success")

		go func(claimed model.Task) {
			defer func() {
				<-workers
			}()
			s.runClaimedTask(ctx, options.WorkerID, options.LeaseDuration, claimed)
		}(*task)
	}
}

func (s *taskService) runClaimedTask(
	ctx context.Context,
	workerID string,
	leaseDuration time.Duration,
	task model.Task,
) {
	stopHeartbeat := s.startTaskHeartbeat(ctx, task.ID, workerID, leaseDuration)
	defer stopHeartbeat()

	payload, err := unmarshalTaskPayload(task.Payload)
	if err != nil {
		s.failClaimedTask(ctx, task, workerID, nonRetryableTaskError{err: err}, map[string]interface{}{"payload": task.Payload})
		return
	}

	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: "task claimed by worker",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"worker_id": workerID,
			"attempt":   task.Attempt,
		}),
	})

	if s.isCanceled(ctx, task.ID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   task.ID,
			Level:    "warn",
			Message:  "task canceled before worker execution",
			Metadata: "{}",
		})
		return
	}

	if !s.setRunningProgress(ctx, task.ID, 5) {
		return
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: "task execution started",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"operation": payload.Operation,
			"progress":  5,
			"worker_id": workerID,
		}),
	})

	if err := s.executeTaskOperation(ctx, task.ID, payload, workerID); err != nil {
		s.failClaimedTask(ctx, task, workerID, err, map[string]interface{}{
			"operation": payload.Operation,
		})
		return
	}

	if s.isCanceled(ctx, task.ID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   task.ID,
			Level:    "warn",
			Message:  "task canceled after operation",
			Metadata: "{}",
		})
		return
	}

	_ = s.repo.CompleteTask(ctx, task.ID, workerID, `{}`)
	s.metrics.ObserveTaskCompleted(task.Type, TaskStatusSuccess, time.Since(task.CreatedAt))
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: "task completed",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"operation": payload.Operation,
			"status":    TaskStatusSuccess,
			"progress":  100,
			"worker_id": workerID,
		}),
	})
}

func (s *taskService) startTaskHeartbeat(
	ctx context.Context,
	taskID int64,
	workerID string,
	leaseDuration time.Duration,
) context.CancelFunc {
	heartbeatCtx, cancel := context.WithCancel(ctx)
	interval := taskHeartbeatInterval(leaseDuration)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				_ = s.repo.HeartbeatTask(heartbeatCtx, taskID, workerID, leaseDuration)
			}
		}
	}()

	return cancel
}

func (s *taskService) executeTaskOperation(
	ctx context.Context,
	taskID int64,
	payload taskPayload,
	workerID string,
) error {
	switch payload.Operation {
	case taskOperationDockerRestart:
		if !s.setRunningProgress(ctx, taskID, 30) {
			return errors.New("task canceled before docker restart")
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
			return err
		}
		if !s.setRunningProgress(ctx, taskID, 85) {
			return errors.New("task canceled after docker restart")
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
		return nil
	case taskOperationServiceRestart:
		if !s.setRunningProgress(ctx, taskID, 30) {
			return errors.New("task canceled before service restart")
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
			return err
		}
		if !s.setRunningProgress(ctx, taskID, 85) {
			return errors.New("task canceled after service restart")
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
		return nil
	case taskOperationBackupCreate:
		return s.executeBackupCreate(ctx, taskID, payload, workerID)
	case taskOperationBackupVerify:
		return s.executeBackupVerify(ctx, taskID, payload, workerID)
	default:
		return nonRetryableTaskError{err: errors.New("unsupported task operation")}
	}
}

func (s *taskService) executeBackupCreate(
	ctx context.Context,
	taskID int64,
	payload taskPayload,
	workerID string,
) error {
	if s.backupService == nil {
		return nonRetryableTaskError{err: errors.New("backup service is not configured")}
	}
	if !s.setRunningProgress(ctx, taskID, 30) {
		return errors.New("task canceled before backup artifact creation")
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  taskID,
		Level:   "info",
		Message: "creating backup artifact",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"backup_id":     payload.BackupID,
			"resource_type": payload.ResourceType,
			"resource_id":   payload.ResourceID,
			"progress":      30,
			"worker_id":     workerID,
		}),
	})
	backup, err := s.backupService.CreateArtifact(ctx, payload.BackupID)
	if err != nil {
		return err
	}
	if !s.setRunningProgress(ctx, taskID, 85) {
		return errors.New("task canceled after backup artifact creation")
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  taskID,
		Level:   "info",
		Message: "backup artifact created",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"backup_id":  backup.ID,
			"file_path":  backup.FilePath,
			"size_bytes": backup.SizeBytes,
			"checksum":   backup.Checksum,
			"status":     BackupStatusSuccess,
			"progress":   85,
			"worker_id":  workerID,
		}),
	})
	return nil
}

func (s *taskService) executeBackupVerify(
	ctx context.Context,
	taskID int64,
	payload taskPayload,
	workerID string,
) error {
	if s.backupService == nil {
		return nonRetryableTaskError{err: errors.New("backup service is not configured")}
	}
	if !s.setRunningProgress(ctx, taskID, 30) {
		return errors.New("task canceled before backup verification")
	}
	_, err := s.backupService.Verify(ctx, payload.BackupID, dto.VerifyBackupRequest{
		SizeBytes: payload.SizeBytes,
		Checksum:  payload.Checksum,
		FilePath:  payload.FilePath,
	})
	if err != nil {
		return err
	}
	if !s.setRunningProgress(ctx, taskID, 85) {
		return errors.New("task canceled after backup verification")
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:  taskID,
		Level:   "info",
		Message: "backup metadata verified",
		Metadata: marshalTaskMetadata(map[string]interface{}{
			"backup_id": payload.BackupID,
			"checksum":  payload.Checksum,
			"progress":  85,
			"worker_id": workerID,
		}),
	})
	return nil
}

func (s *taskService) failClaimedTask(
	ctx context.Context,
	task model.Task,
	workerID string,
	err error,
	metadata map[string]interface{},
) {
	if s.isCanceled(ctx, task.ID) {
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   task.ID,
			Level:    "warn",
			Message:  "task canceled while operation was running",
			Metadata: marshalTaskMetadata(map[string]interface{}{"worker_id": workerID}),
		})
		return
	}

	now := time.Now()
	var nonRetryable nonRetryableTaskError
	retryable := !errors.As(err, &nonRetryable) && isRetryableTaskError(err)
	nextRunAt := nextTaskRetryAt(now, task.Attempt, task.MaxAttempts, retryable)
	fields := map[string]interface{}{
		"attempt":      task.Attempt,
		"error":        err.Error(),
		"max_attempts": task.MaxAttempts,
		"retryable":    retryable,
		"worker_id":    workerID,
	}
	message := "task failed"
	if nextRunAt != nil {
		fields["next_run_at"] = nextRunAt.Format(time.RFC3339)
		message = "task failed; retry scheduled"
	}
	for key, value := range metadata {
		fields[key] = value
	}

	_ = s.repo.FailTask(ctx, task.ID, workerID, err.Error(), nextRunAt)
	if nextRunAt == nil {
		s.metrics.ObserveTaskCompleted(task.Type, TaskStatusFailed, time.Since(task.CreatedAt))
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:   task.ID,
		Level:    "error",
		Message:  message,
		Metadata: marshalTaskMetadata(fields),
	})
}

func (s *taskService) refreshTaskGauges(ctx context.Context) {
	if s.metrics == nil {
		return
	}
	if count, err := s.repo.CountByStatus(ctx, TaskStatusPending); err == nil {
		s.metrics.SetTaskQueueDepth(count)
	}
	if count, err := s.repo.CountByStatus(ctx, TaskStatusRunning); err == nil {
		s.metrics.SetTasksRunning(count)
	}
}

func taskHeartbeatInterval(leaseDuration time.Duration) time.Duration {
	if leaseDuration <= 0 {
		return 10 * time.Second
	}
	interval := leaseDuration / 3
	if interval <= 0 {
		return leaseDuration
	}
	if interval < 50*time.Millisecond {
		return 50 * time.Millisecond
	}
	return interval
}

func nextTaskRetryAt(now time.Time, attempt int, maxAttempts int, retryable bool) *time.Time {
	if !retryable || maxAttempts <= 0 || attempt >= maxAttempts {
		return nil
	}
	next := now.Add(taskRetryBackoff(attempt))
	return &next
}

func isRetryableTaskError(err error) bool {
	if err == nil {
		return false
	}
	appErr, ok := apperror.As(err)
	if !ok {
		return true
	}
	switch appErr.HTTPStatus {
	case 400, 401, 403, 404, 413:
		return false
	default:
		return true
	}
}

func taskRetryBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	delay := time.Second << (attempt - 1)
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func normalizeTaskWorkerOptions(options TaskWorkerOptions) TaskWorkerOptions {
	options.WorkerID = strings.TrimSpace(options.WorkerID)
	if options.WorkerID == "" {
		hostname, _ := os.Hostname()
		hostname = strings.TrimSpace(hostname)
		if hostname == "" {
			hostname = "snowpanel-backend"
		}
		options.WorkerID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}
	if options.Concurrency <= 0 {
		options.Concurrency = 1
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = 30 * time.Second
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 2 * time.Second
	}
	return options
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
	return security.RedactJSON(string(bytes))
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
	case taskOperationBackupCreate:
		if payload.BackupID <= 0 {
			return taskPayload{}, errors.New("backup_id is required in task payload")
		}
	case taskOperationBackupVerify:
		if payload.BackupID <= 0 {
			return taskPayload{}, errors.New("backup_id is required in task payload")
		}
		if payload.SizeBytes <= 0 {
			return taskPayload{}, errors.New("size_bytes is required in task payload")
		}
		if strings.TrimSpace(payload.Checksum) == "" {
			return taskPayload{}, errors.New("checksum is required in task payload")
		}
	default:
		return taskPayload{}, fmt.Errorf("unsupported operation '%s'", payload.Operation)
	}

	return payload, nil
}

func normalizeTaskIdempotencyKey(raw string) (*string, error) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return nil, nil
	}
	if len(key) > 128 {
		return nil, errors.New("idempotency_key must be at most 128 bytes")
	}
	if strings.ContainsFunc(key, unicode.IsControl) {
		return nil, errors.New("idempotency_key cannot contain control characters")
	}
	return &key, nil
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
