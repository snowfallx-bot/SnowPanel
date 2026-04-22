package service

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type DockerService interface {
	ListContainers(ctx context.Context) (dto.ListDockerContainersResult, error)
	StartContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error)
	StopContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error)
	RestartContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error)
	ListImages(ctx context.Context) (dto.ListDockerImagesResult, error)
}

type dockerService struct {
	agentClient grpcclient.AgentClient
}

func NewDockerService(agentClient grpcclient.AgentClient) DockerService {
	return &dockerService{agentClient: agentClient}
}

func (s *dockerService) ListContainers(ctx context.Context) (dto.ListDockerContainersResult, error) {
	result, err := s.agentClient.ListDockerContainers(ctx)
	if err != nil {
		return dto.ListDockerContainersResult{}, mapAgentError(err)
	}

	containers := make([]dto.DockerContainerInfo, 0, len(result.Containers))
	for _, item := range result.Containers {
		containers = append(containers, dto.DockerContainerInfo{
			ID:     item.ID,
			Name:   item.Name,
			Image:  item.Image,
			State:  item.State,
			Status: item.Status,
		})
	}

	s.emitAudit(ctx, "docker", "list_containers", "", true)
	return dto.ListDockerContainersResult{Containers: containers}, nil
}

func (s *dockerService) StartContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error) {
	result, err := s.agentClient.StartDockerContainer(ctx, grpcclient.DockerContainerActionRequest{ID: id})
	if err != nil {
		s.emitAudit(ctx, "docker", "start_container", id, false)
		return dto.DockerContainerActionResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "docker", "start_container", id, true)
	return dto.DockerContainerActionResult{ID: result.ID, State: result.State}, nil
}

func (s *dockerService) StopContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error) {
	result, err := s.agentClient.StopDockerContainer(ctx, grpcclient.DockerContainerActionRequest{ID: id})
	if err != nil {
		s.emitAudit(ctx, "docker", "stop_container", id, false)
		return dto.DockerContainerActionResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "docker", "stop_container", id, true)
	return dto.DockerContainerActionResult{ID: result.ID, State: result.State}, nil
}

func (s *dockerService) RestartContainer(ctx context.Context, id string) (dto.DockerContainerActionResult, error) {
	result, err := s.agentClient.RestartDockerContainer(ctx, grpcclient.DockerContainerActionRequest{ID: id})
	if err != nil {
		s.emitAudit(ctx, "docker", "restart_container", id, false)
		return dto.DockerContainerActionResult{}, mapAgentError(err)
	}
	s.emitAudit(ctx, "docker", "restart_container", id, true)
	return dto.DockerContainerActionResult{ID: result.ID, State: result.State}, nil
}

func (s *dockerService) ListImages(ctx context.Context) (dto.ListDockerImagesResult, error) {
	result, err := s.agentClient.ListDockerImages(ctx)
	if err != nil {
		return dto.ListDockerImagesResult{}, mapAgentError(err)
	}

	images := make([]dto.DockerImageInfo, 0, len(result.Images))
	for _, item := range result.Images {
		images = append(images, dto.DockerImageInfo{
			ID:       item.ID,
			RepoTags: item.RepoTags,
			Size:     item.Size,
		})
	}

	s.emitAudit(ctx, "docker", "list_images", "", true)
	return dto.ListDockerImagesResult{Images: images}, nil
}

func (s *dockerService) emitAudit(_ context.Context, _ string, _ string, _ string, _ bool) {
	// Reserved for audit integration in stage 19.
}
