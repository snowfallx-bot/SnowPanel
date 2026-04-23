package service

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type fakeTaskRepo struct {
	mu         sync.Mutex
	nextTaskID int64
	nextLogID  int64
	tasks      map[int64]*model.Task
	logs       map[int64][]model.TaskLog
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{
		nextTaskID: 1,
		nextLogID:  1,
		tasks:      map[int64]*model.Task{},
		logs:       map[int64][]model.TaskLog{},
	}
}

func (r *fakeTaskRepo) Create(_ context.Context, task *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cloned := *task
	if cloned.ID == 0 {
		cloned.ID = r.nextTaskID
		r.nextTaskID++
	}
	cloned.CreatedAt = now
	cloned.UpdatedAt = now
	r.tasks[cloned.ID] = &cloned
	task.ID = cloned.ID
	task.CreatedAt = cloned.CreatedAt
	task.UpdatedAt = cloned.UpdatedAt
	return nil
}

func (r *fakeTaskRepo) UpdateStatus(
	_ context.Context,
	id int64,
	status string,
	progress int,
	errorMessage string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return errors.New("task not found")
	}
	task.Status = status
	task.Progress = progress
	task.ErrorMsg = errorMessage
	task.UpdatedAt = time.Now()
	return nil
}

func (r *fakeTaskRepo) GetByID(_ context.Context, id int64) (*model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return nil, nil
	}
	cloned := *task
	return &cloned, nil
}

func (r *fakeTaskRepo) List(_ context.Context, filter repository.TaskListFilter) ([]model.Task, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]model.Task, 0, len(r.tasks))
	for _, item := range r.tasks {
		items = append(items, *item)
	}
	slices.SortFunc(items, func(a model.Task, b model.Task) int {
		if a.ID > b.ID {
			return -1
		}
		if a.ID < b.ID {
			return 1
		}
		return 0
	})

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	size := filter.Size
	if size <= 0 {
		size = 20
	}

	start := (page - 1) * size
	if start >= len(items) {
		return []model.Task{}, int64(len(items)), nil
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], int64(len(items)), nil
}

func (r *fakeTaskRepo) AppendLog(_ context.Context, log *model.TaskLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *log
	if cloned.ID == 0 {
		cloned.ID = r.nextLogID
		r.nextLogID++
	}
	cloned.CreatedAt = time.Now()
	r.logs[cloned.TaskID] = append(r.logs[cloned.TaskID], cloned)
	log.ID = cloned.ID
	log.CreatedAt = cloned.CreatedAt
	return nil
}

func (r *fakeTaskRepo) ListLogs(_ context.Context, taskID int64, limit int) ([]model.TaskLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := append([]model.TaskLog(nil), r.logs[taskID]...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

type fakeTaskDockerService struct {
	restartFn func(context.Context, string) (dto.DockerContainerActionResult, error)
}

func (f fakeTaskDockerService) ListContainers(context.Context) (dto.ListDockerContainersResult, error) {
	return dto.ListDockerContainersResult{}, nil
}

func (f fakeTaskDockerService) StartContainer(context.Context, string) (dto.DockerContainerActionResult, error) {
	return dto.DockerContainerActionResult{}, nil
}

func (f fakeTaskDockerService) StopContainer(context.Context, string) (dto.DockerContainerActionResult, error) {
	return dto.DockerContainerActionResult{}, nil
}

func (f fakeTaskDockerService) RestartContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error) {
	if f.restartFn != nil {
		return f.restartFn(ctx, id)
	}
	return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
}

func (f fakeTaskDockerService) ListImages(context.Context) (dto.ListDockerImagesResult, error) {
	return dto.ListDockerImagesResult{}, nil
}

type fakeServiceManager struct {
	restartFn func(context.Context, string) (dto.ServiceActionResult, error)
}

func (f fakeServiceManager) ListServices(context.Context, dto.ListServicesQuery) (dto.ListServicesResult, error) {
	return dto.ListServicesResult{}, nil
}

func (f fakeServiceManager) StartService(context.Context, string) (dto.ServiceActionResult, error) {
	return dto.ServiceActionResult{}, nil
}

func (f fakeServiceManager) StopService(context.Context, string) (dto.ServiceActionResult, error) {
	return dto.ServiceActionResult{}, nil
}

func (f fakeServiceManager) RestartService(ctx context.Context, name string) (dto.ServiceActionResult, error) {
	if f.restartFn != nil {
		return f.restartFn(ctx, name)
	}
	return dto.ServiceActionResult{Name: name, Status: "active"}, nil
}

func TestCreateDockerRestartTaskRunsToSuccess(t *testing.T) {
	repo := newFakeTaskRepo()
	service := NewTaskService(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
	)

	result, err := service.CreateDockerRestartTask(
		context.Background(),
		dto.CreateDockerRestartTaskRequest{ContainerID: "web"},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected create task success, got %v", err)
	}
	if result.Type != TaskTypeDockerRestart {
		t.Fatalf("unexpected task type: %s", result.Type)
	}

	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 2*time.Second)

	logs, err := repo.ListLogs(context.Background(), result.ID, 200)
	if err != nil {
		t.Fatalf("expected log list success, got %v", err)
	}
	hasRestartLog := false
	for _, item := range logs {
		if item.Message == "docker container restarted" {
			hasRestartLog = true
			break
		}
	}
	if !hasRestartLog {
		t.Fatalf("expected docker restart log in task logs")
	}
}

func TestCancelTaskMarksPendingTaskCanceled(t *testing.T) {
	repo := newFakeTaskRepo()
	if err := repo.Create(context.Background(), &model.Task{
		Type:     TaskTypeServiceRestart,
		Status:   TaskStatusPending,
		Progress: 0,
		Payload:  `{"operation":"service.restart","service_name":"nginx.service"}`,
		Result:   `{}`,
	}); err != nil {
		t.Fatalf("failed to seed task: %v", err)
	}

	service := NewTaskService(repo, nil, nil)
	if err := service.CancelTask(context.Background(), 1, "tester"); err != nil {
		t.Fatalf("expected cancel success, got %v", err)
	}

	task, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task == nil || task.Status != TaskStatusCanceled {
		t.Fatalf("expected canceled task status, got %+v", task)
	}
}

func TestRetryTaskFromFailedCreatesNewTask(t *testing.T) {
	repo := newFakeTaskRepo()
	if err := repo.Create(context.Background(), &model.Task{
		Type:     TaskTypeServiceRestart,
		Status:   TaskStatusFailed,
		Progress: 100,
		Payload:  `{"operation":"service.restart","service_name":"nginx.service"}`,
		Result:   `{}`,
		ErrorMsg: "mock failure",
	}); err != nil {
		t.Fatalf("failed to seed failed task: %v", err)
	}

	service := NewTaskService(repo, nil, fakeServiceManager{
		restartFn: func(_ context.Context, name string) (dto.ServiceActionResult, error) {
			return dto.ServiceActionResult{Name: name, Status: "active"}, nil
		},
	})

	result, err := service.RetryTask(context.Background(), 1, nil, "tester")
	if err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if result.ID == 1 {
		t.Fatalf("expected retry to create a new task")
	}

	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 2*time.Second)
}

func TestCancelTaskKeepsCanceledStatusAfterRunningOperationCompletes(t *testing.T) {
	repo := newFakeTaskRepo()
	started := make(chan struct{})
	release := make(chan struct{})

	service := NewTaskService(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				close(started)
				<-release
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
	)

	result, err := service.CreateDockerRestartTask(
		context.Background(),
		dto.CreateDockerRestartTaskRequest{ContainerID: "web"},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected create task success, got %v", err)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("docker restart function did not start in time")
	}

	if err := service.CancelTask(context.Background(), result.ID, "tester"); err != nil {
		t.Fatalf("expected cancel success, got %v", err)
	}

	close(release)

	task := waitForTaskTerminalStatus(t, repo, result.ID, 2*time.Second)
	if task.Status != TaskStatusCanceled {
		t.Fatalf("expected final status canceled, got %+v", task)
	}
}

func waitForTaskStatus(
	t *testing.T,
	repo *fakeTaskRepo,
	taskID int64,
	wantStatus string,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task, err := repo.GetByID(context.Background(), taskID)
		if err == nil && task != nil && task.Status == wantStatus {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	task, _ := repo.GetByID(context.Background(), taskID)
	t.Fatalf("task %d did not reach status %s, current=%+v", taskID, wantStatus, task)
}

func waitForTaskTerminalStatus(
	t *testing.T,
	repo *fakeTaskRepo,
	taskID int64,
	timeout time.Duration,
) model.Task {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task, err := repo.GetByID(context.Background(), taskID)
		if err == nil && task != nil {
			switch task.Status {
			case TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled:
				return *task
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	task, _ := repo.GetByID(context.Background(), taskID)
	t.Fatalf("task %d did not reach terminal status, current=%+v", taskID, task)
	return model.Task{}
}
