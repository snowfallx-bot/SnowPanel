package repository

import (
	"context"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type TaskListFilter struct {
	Page   int
	Size   int
	Status string
	Type   string
}

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	ClaimNextTask(ctx context.Context, workerID string, leaseDuration time.Duration) (*model.Task, error)
	HeartbeatTask(ctx context.Context, taskID int64, workerID string, leaseDuration time.Duration) error
	CompleteTask(ctx context.Context, taskID int64, workerID string, result string) error
	FailTask(ctx context.Context, taskID int64, workerID string, errorMessage string, nextRunAt *time.Time) error
	ReleaseStaleTasks(ctx context.Context, now time.Time) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status string, progress int, errorMessage string) error
	CountByStatus(ctx context.Context, status string) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	List(ctx context.Context, filter TaskListFilter) ([]model.Task, int64, error)
	AppendLog(ctx context.Context, log *model.TaskLog) error
	ListLogs(ctx context.Context, taskID int64, limit int) ([]model.TaskLog, error)
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *model.Task) error {
	if task.MaxAttempts <= 0 {
		task.MaxAttempts = 1
	}
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *taskRepository) ClaimNextTask(
	ctx context.Context,
	workerID string,
	leaseDuration time.Duration,
) (*model.Task, error) {
	now := time.Now()
	lockedUntil := now.Add(leaseDuration)
	var task model.Task
	err := r.db.WithContext(ctx).Raw(`
UPDATE tasks
SET status = 'running',
    locked_by = ?,
    locked_until = ?,
    attempt = attempt + 1,
    started_at = COALESCE(started_at, ?),
    updated_at = ?
WHERE id = (
    SELECT id
    FROM tasks
    WHERE status = 'pending'
      AND (next_run_at IS NULL OR next_run_at <= ?)
      AND attempt < max_attempts
    ORDER BY created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *`,
		workerID,
		lockedUntil,
		now,
		now,
		now,
	).Scan(&task).Error
	if err != nil {
		return nil, err
	}
	if task.ID == 0 {
		return nil, nil
	}
	return &task, nil
}

func (r *taskRepository) HeartbeatTask(
	ctx context.Context,
	taskID int64,
	workerID string,
	leaseDuration time.Duration,
) error {
	updates := map[string]interface{}{
		"locked_until": time.Now().Add(leaseDuration),
	}
	return r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND locked_by = ? AND status = ?", taskID, workerID, "running").
		Updates(updates).Error
}

func (r *taskRepository) CompleteTask(
	ctx context.Context,
	taskID int64,
	workerID string,
	result string,
) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":        "success",
		"progress":      100,
		"result":        result,
		"error_message": "",
		"locked_by":     nil,
		"locked_until":  nil,
		"finished_at":   now,
	}
	return r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND locked_by = ? AND status = ?", taskID, workerID, "running").
		Updates(updates).Error
}

func (r *taskRepository) FailTask(
	ctx context.Context,
	taskID int64,
	workerID string,
	errorMessage string,
	nextRunAt *time.Time,
) error {
	status := "failed"
	if nextRunAt != nil {
		status = "pending"
	}
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
		"locked_by":     nil,
		"locked_until":  nil,
		"next_run_at":   nextRunAt,
	}
	if status == "failed" {
		updates["progress"] = 100
		updates["finished_at"] = time.Now()
	}
	return r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND locked_by = ? AND status = ?", taskID, workerID, "running").
		Updates(updates).Error
}

func (r *taskRepository) ReleaseStaleTasks(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("status = ? AND locked_until IS NOT NULL AND locked_until < ?", "running", now).
		Updates(map[string]interface{}{
			"status":       "pending",
			"locked_by":    nil,
			"locked_until": nil,
			"next_run_at":  now,
		})
	return result.RowsAffected, result.Error
}

func (r *taskRepository) UpdateStatus(
	ctx context.Context,
	id int64,
	status string,
	progress int,
	errorMessage string,
) error {
	updates := map[string]interface{}{
		"status":        status,
		"progress":      progress,
		"error_message": errorMessage,
	}
	return r.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Updates(updates).Error
}

func (r *taskRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("status = ?", status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *taskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) List(
	ctx context.Context,
	filter TaskListFilter,
) ([]model.Task, int64, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	size := filter.Size
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	query := r.db.WithContext(ctx).Model(&model.Task{})
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if taskType := strings.TrimSpace(filter.Type); taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.Task
	if err := query.
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *taskRepository) AppendLog(ctx context.Context, log *model.TaskLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *taskRepository) ListLogs(ctx context.Context, taskID int64, limit int) ([]model.TaskLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var items []model.TaskLog
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
