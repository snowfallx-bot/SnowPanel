package repository

import (
	"context"
	"errors"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type HostRepository interface {
	Create(ctx context.Context, host *model.Host) error
	GetByID(ctx context.Context, id int64) (*model.Host, error)
	GetByAddressPort(ctx context.Context, address string, port int) (*model.Host, error)
	List(ctx context.Context) ([]model.Host, error)
	Update(ctx context.Context, host *model.Host) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	UpdateHealth(
		ctx context.Context,
		id int64,
		status int16,
		agentVersion string,
		lastSeenAt *time.Time,
	) error
}

type hostRepository struct {
	db *gorm.DB
}

func NewHostRepository(db *gorm.DB) HostRepository {
	return &hostRepository{db: db}
}

func (r *hostRepository) Create(ctx context.Context, host *model.Host) error {
	return r.db.WithContext(ctx).Create(host).Error
}

func (r *hostRepository) GetByID(ctx context.Context, id int64) (*model.Host, error) {
	var host model.Host
	err := r.db.WithContext(ctx).First(&host, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (r *hostRepository) GetByAddressPort(ctx context.Context, address string, port int) (*model.Host, error) {
	var host model.Host
	err := r.db.WithContext(ctx).First(&host, "address = ? AND port = ?", address, port).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (r *hostRepository) List(ctx context.Context) ([]model.Host, error) {
	var items []model.Host
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *hostRepository) Update(ctx context.Context, host *model.Host) error {
	return r.db.WithContext(ctx).
		Model(&model.Host{}).
		Where("id = ?", host.ID).
		Updates(map[string]interface{}{
			"name":          host.Name,
			"address":       host.Address,
			"port":          host.Port,
			"status":        host.Status,
			"agent_version": host.AgentVersion,
			"last_seen_at":  host.LastSeenAt,
		}).Error
}

func (r *hostRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).
		Model(&model.Host{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}

func (r *hostRepository) UpdateHealth(
	ctx context.Context,
	id int64,
	status int16,
	agentVersion string,
	lastSeenAt *time.Time,
) error {
	updates := map[string]interface{}{
		"status":        status,
		"agent_version": agentVersion,
		"last_seen_at":  lastSeenAt,
	}
	return r.db.WithContext(ctx).
		Model(&model.Host{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}
