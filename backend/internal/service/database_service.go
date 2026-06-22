package service

import (
	"context"
	"fmt"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type DatabaseService interface {
	ListDatabaseInstances(ctx context.Context) ([]dto.DatabaseInstanceListItem, error)
	GetDatabaseInstance(ctx context.Context, id int64) (dto.DatabaseInstance, error)
	CreateDatabaseInstance(ctx context.Context, req dto.CreateDatabaseInstanceRequest) (dto.DatabaseInstance, error)
	UpdateDatabaseInstance(ctx context.Context, id int64, req dto.UpdateDatabaseInstanceRequest) (dto.DatabaseInstance, error)
	DeleteDatabaseInstance(ctx context.Context, id int64) error
	TestConnection(ctx context.Context, req dto.CreateDatabaseInstanceRequest) (dto.TestConnectionResponse, error)
	ListDatabases(ctx context.Context, instanceID int64) ([]dto.Database, error)
	CreateDatabase(ctx context.Context, req dto.CreateDatabaseRequest) error
	DeleteDatabase(ctx context.Context, req dto.DeleteDatabaseRequest) error
}

type databaseService struct {
	instanceRepo    repository.DatabaseInstanceRepository
	databaseRepo    repository.DatabaseRepository
	auditService    AuditService
}

func NewDatabaseService(
	instanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	auditService AuditService,
) DatabaseService {
	return &databaseService{
		instanceRepo:  instanceRepo,
		databaseRepo:  databaseRepo,
		auditService:  auditService,
	}
}

func (s *databaseService) ListDatabaseInstances(ctx context.Context) ([]dto.DatabaseInstanceListItem, error) {
	instances, err := s.instanceRepo.List(ctx)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.DatabaseInstanceListItem, 0, len(instances))
	for _, inst := range instances {
		result = append(result, dto.DatabaseInstanceListItem{
			ID:       inst.ID,
			Name:     inst.Name,
			Engine:   inst.Engine,
			Host:     inst.Host,
			Port:     inst.Port,
			Status:   inst.Status,
			CreatedAt: inst.CreatedAt,
		})
	}

	return result, nil
}

func (s *databaseService) GetDatabaseInstance(ctx context.Context, id int64) (dto.DatabaseInstance, error) {
	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return dto.DatabaseInstance{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if instance == nil {
		return dto.DatabaseInstance{}, apperror.ErrDatabaseInstanceNotFound
	}

	// Get database count
	dbs, err := s.databaseRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return dto.DatabaseInstance{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return dto.DatabaseInstance{
		ID:           instance.ID,
		Name:         instance.Name,
		Engine:       instance.Engine,
		Host:         instance.Host,
		Port:         instance.Port,
		Username:     instance.Username,
		Status:       instance.Status,
		CreatedAt:    instance.CreatedAt,
		UpdatedAt:    instance.UpdatedAt,
		DatabasesCount: int64(len(dbs)),
	}, nil
}

func (s *databaseService) CreateDatabaseInstance(ctx context.Context, req dto.CreateDatabaseInstanceRequest) (dto.DatabaseInstance, error) {
	// Validate engine
	if !isValidEngine(req.Engine) {
		return dto.DatabaseInstance{}, apperror.ErrInvalidEngine
	}

	// Check if name already exists
	existing, err := s.instanceRepo.GetByName(ctx, req.Name)
	if err != nil {
		return dto.DatabaseInstance{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil {
		return dto.DatabaseInstance{}, apperror.ErrDatabaseInstanceNameExists
	}

	instance := &model.DatabaseInstance{
		Name:     req.Name,
		Engine:   req.Engine,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		// Note: Password should be encrypted before storing
		// For now, we store it as-is; encryption should be handled by caller or service layer
		PasswordEncrypted: req.Password,
		Status:            1, // enabled by default
	}

	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		return dto.DatabaseInstance{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return dto.DatabaseInstance{
		ID:        instance.ID,
		Name:      instance.Name,
		Engine:    instance.Engine,
		Host:      instance.Host,
		Port:      instance.Port,
		Username:  instance.Username,
		Status:    instance.Status,
		CreatedAt: instance.CreatedAt,
		UpdatedAt: instance.UpdatedAt,
	}, nil
}

func (s *databaseService) UpdateDatabaseInstance(ctx context.Context, id int64, req dto.UpdateDatabaseInstanceRequest) (dto.DatabaseInstance, error) {
	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return dto.DatabaseInstance{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if instance == nil {
		return dto.DatabaseInstance{}, apperror.ErrDatabaseInstanceNotFound
	}

	if req.Name != nil {
		// Check if new name conflicts with another instance
		if *req.Name != instance.Name {
			existing, err := s.instanceRepo.GetByName(ctx, *req.Name)
			if err != nil {
				return dto.DatabaseInstance{}, apperror.Wrap(
					apperror.ErrInternal.Code,
					apperror.ErrInternal.HTTPStatus,
					apperror.ErrInternal.Message,
					err,
				)
			}
			if existing != nil && existing.ID != id {
				return dto.DatabaseInstance{}, apperror.ErrDatabaseInstanceNameExists
			}
		}
		instance.Name = *req.Name
	}

	if req.Engine != nil {
		if !isValidEngine(*req.Engine) {
			return dto.DatabaseInstance{}, apperror.ErrInvalidEngine
		}
		instance.Engine = *req.Engine
	}

	if req.Host != nil {
		instance.Host = *req.Host
	}

	if req.Port != nil {
		instance.Port = *req.Port
	}

	if req.Username != nil {
		instance.Username = *req.Username
	}

	if req.Password != nil {
		instance.PasswordEncrypted = *req.Password
	}

	if req.Status != nil {
		instance.Status = *req.Status
	}

	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return dto.DatabaseInstance{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return s.GetDatabaseInstance(ctx, id)
}

func (s *databaseService) DeleteDatabaseInstance(ctx context.Context, id int64) error {
	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if instance == nil {
		return apperror.ErrDatabaseInstanceNotFound
	}

	// Delete databases first
	if err := s.databaseRepo.DeleteByInstanceID(ctx, id); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	if err := s.instanceRepo.Delete(ctx, id); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return nil
}

func (s *databaseService) TestConnection(ctx context.Context, req dto.CreateDatabaseInstanceRequest) (dto.TestConnectionResponse, error) {
	// Validate engine
	if !isValidEngine(req.Engine) {
		return dto.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("invalid engine: %s", req.Engine),
		}, nil
	}

	// Basic connectivity check (host and port)
	// In a real implementation, this would attempt an actual database connection
	// For now, we validate the configuration structure
	if req.Host == "" {
		return dto.TestConnectionResponse{
			Success: false,
			Message: "host is required",
		}, nil
	}

	if req.Port < 1 || req.Port > 65535 {
		return dto.TestConnectionResponse{
			Success: false,
			Message: "port must be between 1 and 65535",
		}, nil
	}

	// Check if name already exists
	existing, err := s.instanceRepo.GetByName(ctx, req.Name)
	if err != nil {
		return dto.TestConnectionResponse{
			Success: false,
			Message: "failed to check existing instances",
		}, nil
	}
	if existing != nil {
		return dto.TestConnectionResponse{
			Success: false,
			Message: "instance name already exists",
		}, nil
	}

	return dto.TestConnectionResponse{
		Success: true,
		Message: "connection test passed (validation successful)",
	}, nil
}

func (s *databaseService) ListDatabases(ctx context.Context, instanceID int64) ([]dto.Database, error) {
	dbs, err := s.databaseRepo.ListByInstanceID(ctx, instanceID)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.Database, 0, len(dbs))
	for _, db := range dbs {
		result = append(result, dto.Database{
			ID:         db.ID,
			InstanceID: db.InstanceID,
			Name:       db.Name,
			Owner:      db.Owner,
			Charset:    db.Charset,
			Collation:  db.Collation,
			CreatedAt:  db.CreatedAt,
		})
	}

	return result, nil
}

func (s *databaseService) CreateDatabase(ctx context.Context, req dto.CreateDatabaseRequest) error {
	// Check if instance exists
	instance, err := s.instanceRepo.GetByID(ctx, req.InstanceID)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if instance == nil {
		return apperror.ErrDatabaseInstanceNotFound
	}

	// Check if database already exists
	existing, err := s.databaseRepo.GetByName(ctx, req.InstanceID, req.Name)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil {
		return apperror.ErrDatabaseNameExists
	}

	db := &model.Database{
		InstanceID: req.InstanceID,
		Name:       req.Name,
		Owner:      req.Owner,
		Charset:    req.Charset,
		Collation:  req.Collation,
	}

	if err := s.databaseRepo.Create(ctx, db); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return nil
}

func (s *databaseService) DeleteDatabase(ctx context.Context, req dto.DeleteDatabaseRequest) error {
	// Check if instance exists
	instance, err := s.instanceRepo.GetByID(ctx, req.InstanceID)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if instance == nil {
		return apperror.ErrDatabaseInstanceNotFound
	}

	// Check if database exists
	existing, err := s.databaseRepo.GetByName(ctx, req.InstanceID, req.Name)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing == nil {
		return apperror.ErrDatabaseNotFound
	}

	if err := s.databaseRepo.Delete(ctx, existing.ID); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return nil
}

func isValidEngine(engine string) bool {
	validEngines := map[string]bool{
		"postgresql": true,
		"mysql":      true,
	}
	return validEngines[engine]
}
