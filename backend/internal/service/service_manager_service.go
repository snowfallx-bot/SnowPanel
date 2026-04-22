package service

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type ServiceManagerService interface {
	ListServices(ctx context.Context, query dto.ListServicesQuery) (dto.ListServicesResult, error)
	StartService(ctx context.Context, name string) (dto.ServiceActionResult, error)
	StopService(ctx context.Context, name string) (dto.ServiceActionResult, error)
	RestartService(ctx context.Context, name string) (dto.ServiceActionResult, error)
}

type serviceManagerService struct {
	agentClient grpcclient.AgentClient
}

func NewServiceManagerService(agentClient grpcclient.AgentClient) ServiceManagerService {
	return &serviceManagerService{
		agentClient: agentClient,
	}
}

func (s *serviceManagerService) ListServices(ctx context.Context, query dto.ListServicesQuery) (dto.ListServicesResult, error) {
	result, err := s.agentClient.ListServices(ctx, grpcclient.ListServicesRequest{
		Keyword: query.Keyword,
	})
	if err != nil {
		return dto.ListServicesResult{}, mapAgentError(err)
	}

	items := make([]dto.ServiceInfo, 0, len(result.Services))
	for _, item := range result.Services {
		items = append(items, dto.ServiceInfo{
			Name:        item.Name,
			DisplayName: item.DisplayName,
			Status:      item.Status,
		})
	}

	s.emitAudit(ctx, "services", "list", query.Keyword, true)
	return dto.ListServicesResult{Services: items}, nil
}

func (s *serviceManagerService) StartService(ctx context.Context, name string) (dto.ServiceActionResult, error) {
	result, err := s.agentClient.StartService(ctx, grpcclient.ServiceActionRequest{Name: name})
	if err != nil {
		s.emitAudit(ctx, "services", "start", name, false)
		return dto.ServiceActionResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "services", "start", name, true)
	return dto.ServiceActionResult{
		Name:   result.Name,
		Status: result.Status,
	}, nil
}

func (s *serviceManagerService) StopService(ctx context.Context, name string) (dto.ServiceActionResult, error) {
	result, err := s.agentClient.StopService(ctx, grpcclient.ServiceActionRequest{Name: name})
	if err != nil {
		s.emitAudit(ctx, "services", "stop", name, false)
		return dto.ServiceActionResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "services", "stop", name, true)
	return dto.ServiceActionResult{
		Name:   result.Name,
		Status: result.Status,
	}, nil
}

func (s *serviceManagerService) RestartService(ctx context.Context, name string) (dto.ServiceActionResult, error) {
	result, err := s.agentClient.RestartService(ctx, grpcclient.ServiceActionRequest{Name: name})
	if err != nil {
		s.emitAudit(ctx, "services", "restart", name, false)
		return dto.ServiceActionResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "services", "restart", name, true)
	return dto.ServiceActionResult{
		Name:   result.Name,
		Status: result.Status,
	}, nil
}

func (s *serviceManagerService) emitAudit(_ context.Context, _ string, _ string, _ string, _ bool) {
	// Reserved for audit integration in stage 19.
}
