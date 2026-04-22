package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type TaskListFilter struct {
	Page int
	Size int
}

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	UpdateStatus(ctx context.Context, id int64, status string, progress int, errorMessage string) error
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
	return r.db.WithContext(ctx).Create(task).Error
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
