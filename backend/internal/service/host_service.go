package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

const (
	HostStatusDisabled int16 = 0
	HostStatusOnline   int16 = 1
	HostStatusOffline  int16 = 2
)

type HostService interface {
	ListHosts(ctx context.Context) (dto.ListHostsResult, error)
	GetHost(ctx context.Context, id int64) (dto.Host, error)
	CreateHost(ctx context.Context, req dto.CreateHostRequest) (dto.Host, error)
	UpdateHost(ctx context.Context, id int64, req dto.UpdateHostRequest) (dto.Host, error)
	EnableHost(ctx context.Context, id int64) (dto.Host, error)
	DisableHost(ctx context.Context, id int64) (dto.Host, error)
	CheckHost(ctx context.Context, id int64) (dto.Host, error)

	// P3-2: Enrollment & Identity
	EnrollHost(ctx context.Context, id int64, token, hostname, agentVersion string, capabilities []string) (dto.Host, error)
	RevokeHost(ctx context.Context, id int64, reason string) (dto.Host, error)
	RotateHostCertificate(ctx context.Context, id int64, reuseKey bool) (dto.Host, error)
	ValidateHostCertificate(ctx context.Context, id int64) (dto.Host, error)
}

type hostService struct {
	repo         repository.HostRepository
	agentTimeout time.Duration
}

func NewHostService(repo repository.HostRepository, agentTimeout time.Duration) HostService {
	return &hostService{
		repo:         repo,
		agentTimeout: agentTimeout,
	}
}

func (s *hostService) ListHosts(ctx context.Context) (dto.ListHostsResult, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return dto.ListHostsResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.Host, 0, len(items))
	for _, item := range items {
		result = append(result, mapHost(item, nil))
	}

	return dto.ListHostsResult{Items: result}, nil
}

func (s *hostService) GetHost(ctx context.Context, id int64) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	return mapHost(*host, nil), nil
}

func (s *hostService) CreateHost(ctx context.Context, req dto.CreateHostRequest) (dto.Host, error) {
	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	if err := validateHostEndpoint(address, req.Port); err != nil {
		return dto.Host{}, err
	}

	existing, err := s.repo.GetByAddressPort(ctx, address, req.Port)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil {
		return dto.Host{}, duplicateHostError()
	}

	health, err := s.probeHost(ctx, address, req.Port)
	if err != nil {
		return dto.Host{}, err
	}

	if name == "" {
		name = health.Identity.Hostname
	}
	if name == "" {
		name = net.JoinHostPort(address, fmt.Sprintf("%d", req.Port))
	}

	now := time.Now().UTC()
	host := &model.Host{
		Name:         name,
		Address:      address,
		Port:         req.Port,
		Status:       HostStatusOnline,
		AgentVersion: health.Identity.Version,
		LastSeenAt:   &now,
	}

	if err := s.repo.Create(ctx, host); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return mapHost(*host, &health.Identity), nil
}

func (s *hostService) UpdateHost(ctx context.Context, id int64, req dto.UpdateHostRequest) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	if err := validateHostEndpoint(address, req.Port); err != nil {
		return dto.Host{}, err
	}

	existing, err := s.repo.GetByAddressPort(ctx, address, req.Port)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil && existing.ID != id {
		return dto.Host{}, duplicateHostError()
	}

	health, err := s.probeHost(ctx, address, req.Port)
	if err != nil {
		return dto.Host{}, err
	}

	if name == "" {
		name = health.Identity.Hostname
	}
	if name == "" {
		name = net.JoinHostPort(address, fmt.Sprintf("%d", req.Port))
	}

	now := time.Now().UTC()
	host.Name = name
	host.Address = address
	host.Port = req.Port
	host.Status = HostStatusOnline
	host.AgentVersion = health.Identity.Version
	host.LastSeenAt = &now

	if err := s.repo.Update(ctx, host); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return mapHost(*host, &health.Identity), nil
}

func (s *hostService) EnableHost(ctx context.Context, id int64) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	health, err := s.probeHost(ctx, host.Address, host.Port)
	if err != nil {
		return dto.Host{}, err
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateHealth(ctx, id, HostStatusOnline, health.Identity.Version, &now); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	host.Status = HostStatusOnline
	host.AgentVersion = health.Identity.Version
	host.LastSeenAt = &now
	return mapHost(*host, &health.Identity), nil
}

func (s *hostService) DisableHost(ctx context.Context, id int64) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	if err := s.repo.UpdateStatus(ctx, id, HostStatusDisabled); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	host.Status = HostStatusDisabled
	return mapHost(*host, nil), nil
}

func (s *hostService) CheckHost(ctx context.Context, id int64) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	if host.Status == HostStatusDisabled {
		return dto.Host{}, apperror.ErrHostDisabled
	}

	health, err := s.probeHost(ctx, host.Address, host.Port)
	if err != nil {
		_ = s.repo.UpdateHealth(ctx, id, HostStatusOffline, host.AgentVersion, nil)
		return dto.Host{}, err
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateHealth(ctx, id, HostStatusOnline, health.Identity.Version, &now); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	host.Status = HostStatusOnline
	host.AgentVersion = health.Identity.Version
	host.LastSeenAt = &now
	return mapHost(*host, &health.Identity), nil
}

func (s *hostService) probeHost(
	ctx context.Context,
	address string,
	port int,
) (grpcclient.HealthCheckResult, error) {
	target := net.JoinHostPort(address, fmt.Sprintf("%d", port))
	client := grpcclient.New(target, s.agentTimeout)

	probeCtx, cancel := context.WithTimeout(ctx, s.agentTimeout)
	defer cancel()

	result, err := client.CheckHealthDetails(probeCtx)
	if err != nil {
		return grpcclient.HealthCheckResult{}, apperror.Wrap(
			apperror.ErrHostUnavailable.Code,
			apperror.ErrHostUnavailable.HTTPStatus,
			apperror.ErrHostUnavailable.Message,
			err,
		)
	}

	if !strings.EqualFold(strings.TrimSpace(result.Status), "SERVING") {
		return grpcclient.HealthCheckResult{}, apperror.Wrap(
			apperror.ErrHostUnavailable.Code,
			apperror.ErrHostUnavailable.HTTPStatus,
			apperror.ErrHostUnavailable.Message,
			fmt.Errorf("unexpected agent health status '%s'", result.Status),
		)
	}

	return result, nil
}

func mapHost(host model.Host, identity *grpcclient.AgentIdentity) dto.Host {
	var lastSeenAt *string
	var revokedAt *string
	if host.LastSeenAt != nil {
		formatted := host.LastSeenAt.UTC().Format(time.RFC3339)
		lastSeenAt = &formatted
	}
	if host.RevokedAt != nil {
		formatted := host.RevokedAt.UTC().Format(time.RFC3339)
		revokedAt = &formatted
	}

	result := dto.Host{
		ID:              host.ID,
		Name:            host.Name,
		Address:         host.Address,
		Port:            host.Port,
		Status:          hostStatusLabel(host.Status),
		AgentVersion:    host.AgentVersion,
		LastSeenAt:      lastSeenAt,
		EnrollmentID:    host.EnrollmentID,
		Revoked:         host.Revoked,
		RevokedAt:       revokedAt,
		RevokedReason:   host.RevokedReason,
		CreatedAt:       host.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       host.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if identity != nil {
		result.Capabilities = append([]string(nil), identity.Capabilities...)
		if strings.TrimSpace(result.AgentVersion) == "" {
			result.AgentVersion = strings.TrimSpace(identity.Version)
		}
	}
	return result
}

func validateHostEndpoint(address string, port int) error {
	if address == "" {
		return apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("address is required"),
		)
	}
	if port <= 0 || port > 65535 {
		return apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("port must be between 1 and 65535"),
		)
	}
	return nil
}

func duplicateHostError() error {
	return apperror.Wrap(
		apperror.ErrBadRequest.Code,
		apperror.ErrBadRequest.HTTPStatus,
		apperror.ErrBadRequest.Message,
		errors.New("host with the same address and port already exists"),
	)
}

func hostStatusLabel(status int16) string {
	switch status {
	case HostStatusDisabled:
		return "disabled"
	case HostStatusOffline:
		return "offline"
	default:
		return "online"
	}
}

// ==================== Enrollment & Identity (P3-2 mTLS) ====================

func (s *hostService) EnrollHost(ctx context.Context, id int64, token, hostname, agentVersion string, capabilities []string) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	target := net.JoinHostPort(host.Address, fmt.Sprintf("%d", host.Port))
	client := grpcclient.New(target, s.agentTimeout)
	enrollmentClient := client.Enrollment()
	result, err := enrollmentClient.Enroll(ctx, token, hostname, agentVersion, capabilities)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateEnrollment(
		ctx, id, result.EnrollmentID, "", false, nil, "",
	); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	host.EnrollmentID = result.EnrollmentID
	host.LastSeenAt = &now
	return mapHost(*host, nil), nil
}

func (s *hostService) RevokeHost(ctx context.Context, id int64, reason string) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	if host.EnrollmentID == "" {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("host is not enrolled"),
		)
	}

	target := net.JoinHostPort(host.Address, fmt.Sprintf("%d", host.Port))
	client := grpcclient.New(target, s.agentTimeout)
	if err := client.Enrollment().Revoke(ctx, host.EnrollmentID, reason); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateEnrollment(
		ctx, id, "", "", true, &now, reason,
	); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	host.Revoked = true
	host.RevokedAt = &now
	host.RevokedReason = reason
	host.Status = HostStatusOffline
	return mapHost(*host, nil), nil
}

func (s *hostService) RotateHostCertificate(ctx context.Context, id int64, reuseKey bool) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	if host.EnrollmentID == "" {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("host is not enrolled"),
		)
	}

	target := net.JoinHostPort(host.Address, fmt.Sprintf("%d", host.Port))
	client := grpcclient.New(target, s.agentTimeout)
	_, err = client.Enrollment().RotateCertificates(ctx, host.EnrollmentID, reuseKey)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateEnrollment(
		ctx, id, host.EnrollmentID, "", false, nil, "",
	); err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	host.LastSeenAt = &now
	return mapHost(*host, nil), nil
}

func (s *hostService) ValidateHostCertificate(ctx context.Context, id int64) (dto.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Host{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return dto.Host{}, apperror.ErrHostNotFound
	}

	// This is a placeholder - in production this would fetch the current cert from the agent
	// and validate it against the enrollment service.
	// For now, we return the host info.
	return mapHost(*host, nil), nil
}
