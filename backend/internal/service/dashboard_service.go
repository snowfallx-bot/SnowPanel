package service

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type DashboardService interface {
	GetSummary(ctx context.Context) (dto.DashboardSummary, error)
}

type dashboardService struct {
	agentClient grpcclient.AgentClient
}

func NewDashboardService(agentClient grpcclient.AgentClient) DashboardService {
	return &dashboardService{
		agentClient: agentClient,
	}
}

func (s *dashboardService) GetSummary(ctx context.Context) (dto.DashboardSummary, error) {
	overview, err := s.agentClient.GetSystemOverview(ctx)
	if err != nil {
		return dto.DashboardSummary{}, mapAgentError(err)
	}

	realtime, err := s.agentClient.GetRealtimeResource(ctx)
	if err != nil {
		return dto.DashboardSummary{}, mapAgentError(err)
	}

	cpuUsage := realtime.CPUUsagePercent
	if cpuUsage == 0 {
		cpuUsage = overview.CPUUsagePercent
	}
	memoryUsage := realtime.MemoryUsagePercent
	if memoryUsage == 0 {
		memoryUsage = overview.MemoryUsagePercent
	}
	diskUsage := realtime.DiskUsagePercent
	if diskUsage == 0 {
		diskUsage = overview.DiskUsagePercent
	}

	return dto.DashboardSummary{
		Hostname:      overview.Hostname,
		SystemVersion: overview.OS,
		KernelVersion: overview.Kernel,
		CPUUsage:      cpuUsage,
		MemoryUsage:   memoryUsage,
		DiskUsage:     diskUsage,
		Uptime:        overview.Uptime,
	}, nil
}
