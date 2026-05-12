package service

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	appmetrics "github.com/snowfallx-bot/SnowPanel/backend/internal/metrics"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type fakeTaskRepo struct {
	mu         sync.Mutex
	nextTaskID int64
	nextLogID  int64
	heartbeats int
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

func (r *fakeTaskRepo) ClaimNextTask(
	_ context.Context,
	workerID string,
	leaseDuration time.Duration,
) (*model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, task := range r.tasks {
		if task.Status != TaskStatusPending {
			continue
		}
		if task.NextRunAt != nil && task.NextRunAt.After(now) {
			continue
		}
		if task.MaxAttempts > 0 && task.Attempt >= task.MaxAttempts {
			continue
		}
		task.Status = TaskStatusRunning
		task.Attempt++
		task.LockedBy = &workerID
		lockedUntil := now.Add(leaseDuration)
		task.LockedUntil = &lockedUntil
		if task.StartedAt == nil {
			startedAt := now
			task.StartedAt = &startedAt
		}
		task.UpdatedAt = now
		cloned := *task
		return &cloned, nil
	}
	return nil, nil
}

func (r *fakeTaskRepo) HeartbeatTask(_ context.Context, taskID int64, workerID string, leaseDuration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[taskID]
	if !ok || task.LockedBy == nil || *task.LockedBy != workerID || task.Status != TaskStatusRunning {
		return nil
	}
	lockedUntil := time.Now().Add(leaseDuration)
	task.LockedUntil = &lockedUntil
	r.heartbeats++
	return nil
}

func (r *fakeTaskRepo) CompleteTask(_ context.Context, taskID int64, workerID string, result string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[taskID]
	if !ok || task.LockedBy == nil || *task.LockedBy != workerID || task.Status != TaskStatusRunning {
		return nil
	}
	now := time.Now()
	task.Status = TaskStatusSuccess
	task.Progress = 100
	task.Result = result
	task.ErrorMsg = ""
	task.LockedBy = nil
	task.LockedUntil = nil
	task.FinishedAt = &now
	task.UpdatedAt = now
	return nil
}

func (r *fakeTaskRepo) FailTask(_ context.Context, taskID int64, workerID string, errorMessage string, nextRunAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[taskID]
	if !ok || task.LockedBy == nil || *task.LockedBy != workerID || task.Status != TaskStatusRunning {
		return nil
	}
	now := time.Now()
	if nextRunAt == nil {
		task.Status = TaskStatusFailed
		task.Progress = 100
		task.FinishedAt = &now
	} else {
		task.Status = TaskStatusPending
		task.NextRunAt = nextRunAt
	}
	task.ErrorMsg = errorMessage
	task.LockedBy = nil
	task.LockedUntil = nil
	task.UpdatedAt = now
	return nil
}

func (r *fakeTaskRepo) ReleaseStaleTasks(_ context.Context, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var released int64
	for _, task := range r.tasks {
		if task.Status == TaskStatusRunning && task.LockedUntil != nil && task.LockedUntil.Before(now) {
			task.Status = TaskStatusPending
			task.LockedBy = nil
			task.LockedUntil = nil
			task.NextRunAt = &now
			task.UpdatedAt = now
			released++
		}
	}
	return released, nil
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

func (r *fakeTaskRepo) CountByStatus(_ context.Context, status string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int64
	for _, task := range r.tasks {
		if task.Status == status {
			count++
		}
	}
	return count, nil
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

func (r *fakeTaskRepo) GetByIdempotencyKey(_ context.Context, key string) (*model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, task := range r.tasks {
		if task.IdempotencyKey != nil && *task.IdempotencyKey == key {
			cloned := *task
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *fakeTaskRepo) List(_ context.Context, filter repository.TaskListFilter) ([]model.Task, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]model.Task, 0, len(r.tasks))
	for _, item := range r.tasks {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Type != "" && item.Type != filter.Type {
			continue
		}
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

func (r *fakeTaskRepo) heartbeatCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.heartbeats
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

func TestCreateDockerRestartTaskCanQueueWithoutImmediateExecution(t *testing.T) {
	repo := newFakeTaskRepo()
	restartCalled := false
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				restartCalled = true
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
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

	time.Sleep(50 * time.Millisecond)
	task, err := repo.GetByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task == nil || task.Status != TaskStatusPending {
		t.Fatalf("expected task to remain pending, got %+v", task)
	}
	if task.MaxAttempts != 3 {
		t.Fatalf("expected max_attempts=3, got %d", task.MaxAttempts)
	}
	if restartCalled {
		t.Fatal("did not expect queued task to execute immediately")
	}
}

func TestCreateDockerRestartTaskReturnsExistingTaskForDuplicateIdempotencyKey(t *testing.T) {
	repo := newFakeTaskRepo()
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
	)

	first, err := service.CreateDockerRestartTask(
		context.Background(),
		dto.CreateDockerRestartTaskRequest{
			ContainerID:    "web",
			IdempotencyKey: "restart-web-1",
		},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected first create task success, got %v", err)
	}
	second, err := service.CreateDockerRestartTask(
		context.Background(),
		dto.CreateDockerRestartTaskRequest{
			ContainerID:    "web",
			IdempotencyKey: " restart-web-1 ",
		},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected duplicate create task success, got %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate request to return task %d, got %d", first.ID, second.ID)
	}
	count, err := repo.CountByStatus(context.Background(), TaskStatusPending)
	if err != nil {
		t.Fatalf("failed to count pending tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one pending task, got %d", count)
	}
}

func TestCreateDockerRestartTaskRejectsInvalidIdempotencyKey(t *testing.T) {
	service := NewTaskServiceWithOptions(
		newFakeTaskRepo(),
		fakeTaskDockerService{},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
	)

	_, err := service.CreateDockerRestartTask(
		context.Background(),
		dto.CreateDockerRestartTaskRequest{
			ContainerID:    "web",
			IdempotencyKey: "bad\nkey",
		},
		nil,
		"tester",
	)
	if err == nil {
		t.Fatal("expected invalid idempotency key error")
	}
}

func TestRunWorkerClaimsAndExecutesQueuedTask(t *testing.T) {
	repo := newFakeTaskRepo()
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 2*time.Second)
	task, err := repo.GetByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	if task.Attempt != 1 {
		t.Fatalf("expected attempt=1, got %d", task.Attempt)
	}
	if task.LockedBy != nil || task.LockedUntil != nil {
		t.Fatalf("expected completed task lock to be cleared, got %+v", task)
	}
}

func TestRunWorkerStopsWhenContextIsCanceled(t *testing.T) {
	service := NewTaskServiceWithOptions(
		newFakeTaskRepo(),
		fakeTaskDockerService{},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.RunWorker(ctx, TaskWorkerOptions{
			WorkerID:      "worker-1",
			Concurrency:   1,
			LeaseDuration: time.Second,
			PollInterval:  20 * time.Millisecond,
		})
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected worker to stop after context cancellation")
	}
}

func TestRunWorkerClaimsSingleTaskOnlyOnce(t *testing.T) {
	repo := newFakeTaskRepo()
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	restartCalls := 0
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				mu.Lock()
				restartCalls++
				mu.Unlock()
				once.Do(func() { close(started) })
				<-release
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   2,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("docker restart function did not start in time")
	}
	time.Sleep(80 * time.Millisecond)
	close(release)
	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()
	if restartCalls != 1 {
		t.Fatalf("expected one execution, got %d", restartCalls)
	}
}

func TestRunWorkerRecoversStaleLeasedTask(t *testing.T) {
	repo := newFakeTaskRepo()
	staleWorker := "stale-worker"
	staleUntil := time.Now().Add(-time.Minute)
	if err := repo.Create(context.Background(), &model.Task{
		Type:        TaskTypeDockerRestart,
		Status:      TaskStatusRunning,
		Progress:    30,
		Payload:     `{"operation":"docker.restart","container_id":"web"}`,
		Result:      `{}`,
		LockedBy:    &staleWorker,
		LockedUntil: &staleUntil,
		Attempt:     1,
		MaxAttempts: 3,
		StartedAt:   &staleUntil,
		ErrorMsg:    "",
		TriggeredBy: nil,
		HostID:      nil,
		NextRunAt:   nil,
		FinishedAt:  nil,
		CreatedAt:   staleUntil,
		UpdatedAt:   staleUntil,
	}); err != nil {
		t.Fatalf("failed to seed stale task: %v", err)
	}

	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-2",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, 1, TaskStatusSuccess, 2*time.Second)
	task, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task.Attempt != 2 {
		t.Fatalf("expected stale task attempt=2 after reclaim, got %d", task.Attempt)
	}
}

func TestRunWorkerRetriesFailedTask(t *testing.T) {
	repo := newFakeTaskRepo()
	var mu sync.Mutex
	restartCalls := 0
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				mu.Lock()
				defer mu.Unlock()
				restartCalls++
				if restartCalls == 1 {
					return dto.DockerContainerActionResult{}, errors.New("temporary restart failure")
				}
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 3*time.Second)
	task, err := repo.GetByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	if task.Attempt != 2 {
		t.Fatalf("expected attempt=2 after retry, got %d", task.Attempt)
	}
}

func TestRunWorkerStopsRetryingAtMaxAttempts(t *testing.T) {
	repo := newFakeTaskRepo()
	var mu sync.Mutex
	restartCalls := 0
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(context.Context, string) (dto.DockerContainerActionResult, error) {
				mu.Lock()
				restartCalls++
				mu.Unlock()
				return dto.DockerContainerActionResult{}, errors.New("permanent restart failure")
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    1,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, result.ID, TaskStatusFailed, 2*time.Second)
	time.Sleep(80 * time.Millisecond)
	task, err := repo.GetByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task.Attempt != 1 {
		t.Fatalf("expected attempt=1, got %d", task.Attempt)
	}
	mu.Lock()
	defer mu.Unlock()
	if restartCalls != 1 {
		t.Fatalf("expected one restart attempt, got %d", restartCalls)
	}
}

func TestRunWorkerDoesNotRetryInvalidPayload(t *testing.T) {
	repo := newFakeTaskRepo()
	if err := repo.Create(context.Background(), &model.Task{
		Type:        TaskTypeDockerRestart,
		Status:      TaskStatusPending,
		Progress:    0,
		Payload:     `{"operation":"docker.restart"}`,
		Result:      `{}`,
		MaxAttempts: 3,
	}); err != nil {
		t.Fatalf("failed to seed task: %v", err)
	}
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, 1, TaskStatusFailed, 2*time.Second)
	task, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task.Attempt != 1 {
		t.Fatalf("expected invalid payload to fail after one attempt, got %d", task.Attempt)
	}
	if task.NextRunAt != nil {
		t.Fatalf("did not expect retry to be scheduled for invalid payload, got %v", task.NextRunAt)
	}
}

func TestRunWorkerDoesNotRetryValidationError(t *testing.T) {
	repo := newFakeTaskRepo()
	var mu sync.Mutex
	restartCalls := 0
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(context.Context, string) (dto.DockerContainerActionResult, error) {
				mu.Lock()
				restartCalls++
				mu.Unlock()
				return dto.DockerContainerActionResult{}, apperror.Wrap(
					apperror.ErrBadRequest.Code,
					apperror.ErrBadRequest.HTTPStatus,
					apperror.ErrBadRequest.Message,
					errors.New("invalid container id"),
				)
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    3,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, result.ID, TaskStatusFailed, 2*time.Second)
	time.Sleep(80 * time.Millisecond)
	task, err := repo.GetByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task.Attempt != 1 {
		t.Fatalf("expected validation error to fail after one attempt, got %d", task.Attempt)
	}
	mu.Lock()
	defer mu.Unlock()
	if restartCalls != 1 {
		t.Fatalf("expected one restart attempt, got %d", restartCalls)
	}
}

func TestRunWorkerHeartbeatsLongRunningTask(t *testing.T) {
	repo := newFakeTaskRepo()
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				time.Sleep(250 * time.Millisecond)
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: 150 * time.Millisecond,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 2*time.Second)
	if repo.heartbeatCount() == 0 {
		t.Fatal("expected at least one heartbeat for long-running task")
	}
}

func TestRunWorkerSkipsCanceledQueuedTask(t *testing.T) {
	repo := newFakeTaskRepo()
	restartCalled := false
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				restartCalled = true
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
		},
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
	if err := service.CancelTask(context.Background(), result.ID, "tester"); err != nil {
		t.Fatalf("expected cancel success, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	time.Sleep(80 * time.Millisecond)
	task, err := repo.GetByID(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}
	if task.Status != TaskStatusCanceled {
		t.Fatalf("expected task to stay canceled, got %+v", task)
	}
	if restartCalled {
		t.Fatal("did not expect worker to execute canceled queued task")
	}
}

func TestRunWorkerKeepsCanceledStatusAfterRunningOperationCompletes(t *testing.T) {
	repo := newFakeTaskRepo()
	started := make(chan struct{})
	release := make(chan struct{})
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				close(started)
				<-release
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("docker restart function did not start in time")
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

func TestRunWorkerRecordsTaskMetrics(t *testing.T) {
	repo := newFakeTaskRepo()
	registry := prometheus.NewRegistry()
	metricsSet := appmetrics.New(registry)
	service := NewTaskServiceWithOptions(
		repo,
		fakeTaskDockerService{
			restartFn: func(_ context.Context, id string) (dto.DockerContainerActionResult, error) {
				return dto.DockerContainerActionResult{ID: id, State: "running"}, nil
			},
		},
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
			Metrics:        metricsSet,
		},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, repo, result.ID, TaskStatusSuccess, 2*time.Second)

	if got := testutil.ToFloat64(metricsSet.TaskWorkerClaims.WithLabelValues("success")); got != 1 {
		t.Fatalf("expected one successful claim metric, got %f", got)
	}
	if got := testutil.ToFloat64(metricsSet.TasksCompletedTotal.WithLabelValues(TaskTypeDockerRestart, TaskStatusSuccess)); got != 1 {
		t.Fatalf("expected one completed task metric, got %f", got)
	}
	if got := testutil.CollectAndCount(metricsSet.TaskDuration); got == 0 {
		t.Fatal("expected task duration metric to be collected")
	}
}

func TestRunWorkerCompletesBackupCreateTask(t *testing.T) {
	taskRepo := newFakeTaskRepo()
	backupRepo := newFakeBackupRepo()
	backupSvc := NewBackupServiceWithOptions(backupRepo, BackupServiceOptions{LocalDir: t.TempDir()})
	taskSvc := NewTaskServiceWithOptions(
		taskRepo,
		nil,
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
			BackupService:  backupSvc,
		},
	)

	result, err := taskSvc.CreateBackupTask(
		context.Background(),
		dto.CreateBackupTaskRequest{
			ResourceType: BackupResourcePostgres,
			ResourceID:   "primary",
			StorageType:  BackupStorageLocal,
		},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected backup task create success, got %v", err)
	}
	if result.Task.Type != TaskTypeBackupCreate {
		t.Fatalf("unexpected task type: %s", result.Task.Type)
	}
	backup, err := backupRepo.GetByID(context.Background(), result.Backup.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}
	if backup == nil || backup.Status != BackupStatusPending {
		t.Fatalf("expected pending backup metadata, got %+v", backup)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go taskSvc.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, taskRepo, result.Task.ID, TaskStatusSuccess, 2*time.Second)
	backup, err = backupRepo.GetByID(context.Background(), result.Backup.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}
	if backup.Status != BackupStatusSuccess {
		t.Fatalf("expected backup status success, got %+v", backup)
	}
	if backup.FilePath == "" || backup.SizeBytes <= 0 || !strings.HasPrefix(backup.Checksum, "sha256:") {
		t.Fatalf("expected backup artifact metadata, got %+v", backup)
	}
	if _, err := os.Stat(backup.FilePath); err != nil {
		t.Fatalf("expected backup artifact file to exist: %v", err)
	}
}

func TestRunWorkerVerifiesBackupTask(t *testing.T) {
	taskRepo := newFakeTaskRepo()
	backupRepo := newFakeBackupRepo()
	backupSvc := NewBackupService(backupRepo)
	backup, err := backupSvc.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: BackupResourcePostgres,
		ResourceID:   "primary",
		StorageType:  BackupStorageLocal,
	}, nil)
	if err != nil {
		t.Fatalf("expected backup metadata create success, got %v", err)
	}

	taskSvc := NewTaskServiceWithOptions(
		taskRepo,
		nil,
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
			BackupService:  backupSvc,
		},
	)
	result, err := taskSvc.CreateBackupVerifyTask(
		context.Background(),
		backup.ID,
		dto.CreateBackupVerifyTaskRequest{
			SizeBytes: 4096,
			Checksum:  testChecksumA,
			FilePath:  "backups/postgres.dump.sql",
		},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected backup verify task create success, got %v", err)
	}
	if result.Task.Type != TaskTypeBackupVerify {
		t.Fatalf("unexpected task type: %s", result.Task.Type)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go taskSvc.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, taskRepo, result.Task.ID, TaskStatusSuccess, 2*time.Second)
	verified, err := backupRepo.GetByID(context.Background(), backup.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}
	if verified.Status != BackupStatusSuccess {
		t.Fatalf("expected verified backup success, got %+v", verified)
	}
	if verified.Checksum != "sha256:"+testChecksumA || verified.SizeBytes != 4096 {
		t.Fatalf("expected checksum and size to be recorded, got %+v", verified)
	}
}

func TestCreateBackupVerifyTaskRejectsPartialManualVerification(t *testing.T) {
	cases := []struct {
		name string
		req  dto.CreateBackupVerifyTaskRequest
	}{
		{
			name: "checksum only",
			req:  dto.CreateBackupVerifyTaskRequest{Checksum: testChecksumA},
		},
		{
			name: "size only",
			req:  dto.CreateBackupVerifyTaskRequest{SizeBytes: 4096},
		},
		{
			name: "file path only",
			req:  dto.CreateBackupVerifyTaskRequest{FilePath: "backups/postgres.dump.sql"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskRepo := newFakeTaskRepo()
			backupRepo := newFakeBackupRepo()
			backupSvc := NewBackupService(backupRepo)
			taskSvc := NewTaskServiceWithOptions(
				taskRepo,
				nil,
				nil,
				TaskServiceOptions{
					AsyncExecution: false,
					MaxAttempts:    2,
					BackupService:  backupSvc,
				},
			)

			_, err := taskSvc.CreateBackupVerifyTask(
				context.Background(),
				1,
				tc.req,
				nil,
				"tester",
			)
			appErr, ok := apperror.As(err)
			if !ok || appErr.Code != apperror.ErrBadRequest.Code {
				t.Fatalf("expected bad request, got %v", err)
			}
			if len(taskRepo.tasks) != 0 {
				t.Fatalf("expected no task to be enqueued, got %d", len(taskRepo.tasks))
			}
		})
	}
}

func TestRunWorkerVerifiesRecordedBackupArtifact(t *testing.T) {
	taskRepo := newFakeTaskRepo()
	backupRepo := newFakeBackupRepo()
	backupSvc := NewBackupServiceWithOptions(backupRepo, BackupServiceOptions{LocalDir: t.TempDir()})
	backup, err := backupSvc.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: BackupResourceAppMetadata,
		ResourceID:   "primary",
		StorageType:  BackupStorageLocal,
	}, nil)
	if err != nil {
		t.Fatalf("expected backup metadata create success, got %v", err)
	}
	artifact, err := backupSvc.CreateArtifact(context.Background(), backup.ID)
	if err != nil {
		t.Fatalf("expected backup artifact create success, got %v", err)
	}

	taskSvc := NewTaskServiceWithOptions(
		taskRepo,
		nil,
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
			BackupService:  backupSvc,
		},
	)
	result, err := taskSvc.CreateBackupVerifyTask(
		context.Background(),
		backup.ID,
		dto.CreateBackupVerifyTaskRequest{},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected backup verify task create success, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go taskSvc.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, taskRepo, result.Task.ID, TaskStatusSuccess, 2*time.Second)
	verified, err := backupRepo.GetByID(context.Background(), backup.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}
	if verified.Status != BackupStatusSuccess ||
		verified.Checksum != artifact.Checksum ||
		verified.SizeBytes != artifact.SizeBytes ||
		verified.FilePath != artifact.FilePath {
		t.Fatalf("expected recorded artifact to be verified, got %+v want %+v", verified, artifact)
	}
}

func TestRunWorkerMarksBackupVerifyMismatchFailed(t *testing.T) {
	taskRepo := newFakeTaskRepo()
	backupRepo := newFakeBackupRepo()
	now := time.Now()
	backupRepo.items[1] = model.Backup{
		ID:           1,
		ResourceType: BackupResourcePostgres,
		ResourceID:   "primary",
		StorageType:  BackupStorageLocal,
		FilePath:     "backups/postgres.dump.sql",
		SizeBytes:    2048,
		Checksum:     "sha256:" + testChecksumA,
		Status:       BackupStatusSuccess,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	backupSvc := NewBackupService(backupRepo)
	taskSvc := NewTaskServiceWithOptions(
		taskRepo,
		nil,
		nil,
		TaskServiceOptions{
			AsyncExecution: false,
			MaxAttempts:    2,
			BackupService:  backupSvc,
		},
	)

	result, err := taskSvc.CreateBackupVerifyTask(
		context.Background(),
		1,
		dto.CreateBackupVerifyTaskRequest{
			SizeBytes: 2048,
			Checksum:  testChecksumB,
		},
		nil,
		"tester",
	)
	if err != nil {
		t.Fatalf("expected backup verify task create success, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go taskSvc.RunWorker(ctx, TaskWorkerOptions{
		WorkerID:      "worker-1",
		Concurrency:   1,
		LeaseDuration: time.Second,
		PollInterval:  20 * time.Millisecond,
	})

	waitForTaskStatus(t, taskRepo, result.Task.ID, TaskStatusFailed, 2*time.Second)
	backup, err := backupRepo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}
	if backup.Status != BackupStatusFailed {
		t.Fatalf("expected backup status failed, got %+v", backup)
	}
	task, err := taskRepo.GetByID(context.Background(), result.Task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task.Attempt != 1 {
		t.Fatalf("expected non-retryable mismatch to fail after one attempt, got %d", task.Attempt)
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

func TestListTasksSupportsStatusAndTypeFilters(t *testing.T) {
	repo := newFakeTaskRepo()
	_ = repo.Create(context.Background(), &model.Task{
		Type:     TaskTypeDockerRestart,
		Status:   TaskStatusSuccess,
		Progress: 100,
		Payload:  `{"operation":"docker.restart","container_id":"web"}`,
		Result:   `{}`,
	})
	_ = repo.Create(context.Background(), &model.Task{
		Type:     TaskTypeServiceRestart,
		Status:   TaskStatusRunning,
		Progress: 30,
		Payload:  `{"operation":"service.restart","service_name":"nginx.service"}`,
		Result:   `{}`,
	})
	_ = repo.Create(context.Background(), &model.Task{
		Type:     TaskTypeServiceRestart,
		Status:   TaskStatusSuccess,
		Progress: 100,
		Payload:  `{"operation":"service.restart","service_name":"redis.service"}`,
		Result:   `{}`,
	})

	service := NewTaskService(repo, nil, nil)
	result, err := service.ListTasks(context.Background(), dto.ListTasksQuery{
		Page:   1,
		Size:   20,
		Status: TaskStatusSuccess,
		Type:   TaskTypeServiceRestart,
	})
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total=1, got %d", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Type != TaskTypeServiceRestart || result.Items[0].Status != TaskStatusSuccess {
		t.Fatalf("unexpected item: %+v", result.Items[0])
	}
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

func TestMarshalTaskMetadataRedactsSensitiveFields(t *testing.T) {
	metadata := marshalTaskMetadata(map[string]interface{}{
		"actor":         "tester",
		"access_token":  "secret-token",
		"service_name":  "nginx.service",
		"nested_secret": map[string]interface{}{"password": "plain"},
	})

	if strings.Contains(metadata, "secret-token") || strings.Contains(metadata, "plain") {
		t.Fatalf("task metadata leaked sensitive data: %s", metadata)
	}
	if !strings.Contains(metadata, "nginx.service") {
		t.Fatalf("expected non-sensitive metadata to remain visible: %s", metadata)
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
