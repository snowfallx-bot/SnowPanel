package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

const (
	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
)

type TaskService interface {
	CreateDemoTask(ctx context.Context, triggeredBy *int64, username string) (dto.CreateDemoTaskResult, error)
	ListTasks(ctx context.Context, query dto.ListTasksQuery) (dto.ListTasksResult, error)
	GetTaskDetail(ctx context.Context, id int64) (dto.TaskDetail, error)
}

type taskService struct {
	repo repository.TaskRepository
}

func NewTaskService(repo repository.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

func (s *taskService) CreateDemoTask(
	ctx context.Context,
	triggeredBy *int64,
	username string,
) (dto.CreateDemoTaskResult, error) {
	task := &model.Task{
		Type:        "mock_backup",
		Status:      TaskStatusPending,
		Progress:    0,
		Payload:     `{"demo":"mock_backup"}`,
		Result:      `{}`,
		ErrorMsg:    "",
		TriggeredBy: triggeredBy,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return dto.CreateDemoTaskResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:   task.ID,
		Level:    "info",
		Message:  "task created",
		Metadata: marshalTaskMetadata(map[string]interface{}{"actor": username}),
	})

	go s.runDemoTask(task.ID)

	return dto.CreateDemoTaskResult{
		ID:     task.ID,
		Type:   task.Type,
		Status: task.Status,
	}, nil
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

func (s *taskService) runDemoTask(taskID int64) {
	ctx := context.Background()

	if err := s.repo.UpdateStatus(ctx, taskID, TaskStatusRunning, 5, ""); err != nil {
		return
	}
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:   taskID,
		Level:    "info",
		Message:  "mock backup task started",
		Metadata: marshalTaskMetadata(map[string]interface{}{"step": "start"}),
	})

	steps := []struct {
		progress int
		message  string
		sleep    time.Duration
	}{
		{25, "collecting metadata", 800 * time.Millisecond},
		{55, "packing files", 1200 * time.Millisecond},
		{80, "uploading archive", 1000 * time.Millisecond},
		{95, "verifying checksum", 600 * time.Millisecond},
	}

	for _, step := range steps {
		time.Sleep(step.sleep)
		_ = s.repo.UpdateStatus(ctx, taskID, TaskStatusRunning, step.progress, "")
		_ = s.repo.AppendLog(ctx, &model.TaskLog{
			TaskID:   taskID,
			Level:    "info",
			Message:  step.message,
			Metadata: marshalTaskMetadata(map[string]interface{}{"progress": step.progress}),
		})
	}

	_ = s.repo.UpdateStatus(ctx, taskID, TaskStatusSuccess, 100, "")
	_ = s.repo.AppendLog(ctx, &model.TaskLog{
		TaskID:   taskID,
		Level:    "info",
		Message:  "mock backup completed",
		Metadata: marshalTaskMetadata(map[string]interface{}{"result": "success"}),
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
