package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/hostctx"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type hostAwareAgentClient struct {
	defaultClient grpcclient.AgentClient
	hostRepo      repository.HostRepository
	timeout       time.Duration

	mu      sync.RWMutex
	clients map[string]grpcclient.AgentClient
}

func NewHostAwareAgentClient(
	defaultTarget string,
	timeout time.Duration,
	hostRepo repository.HostRepository,
) grpcclient.AgentClient {
	defaultAgentClient := grpcclient.New(defaultTarget, timeout)
	return &hostAwareAgentClient{
		defaultClient: defaultAgentClient,
		hostRepo:      hostRepo,
		timeout:       timeout,
		clients:       map[string]grpcclient.AgentClient{},
	}
}

func (c *hostAwareAgentClient) CheckHealth(ctx context.Context) (string, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return "", err
	}
	return client.CheckHealth(ctx)
}

func (c *hostAwareAgentClient) CheckHealthDetails(ctx context.Context) (grpcclient.HealthCheckResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.HealthCheckResult{}, err
	}
	return client.CheckHealthDetails(ctx)
}

func (c *hostAwareAgentClient) GetSystemOverview(ctx context.Context) (grpcclient.SystemOverview, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.SystemOverview{}, err
	}
	return client.GetSystemOverview(ctx)
}

func (c *hostAwareAgentClient) GetRealtimeResource(ctx context.Context) (grpcclient.RealtimeResource, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.RealtimeResource{}, err
	}
	return client.GetRealtimeResource(ctx)
}

func (c *hostAwareAgentClient) ListFiles(ctx context.Context, req grpcclient.ListFilesRequest) (grpcclient.ListFilesResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ListFilesResult{}, err
	}
	return client.ListFiles(ctx, req)
}

func (c *hostAwareAgentClient) ReadTextFile(ctx context.Context, req grpcclient.ReadTextFileRequest) (grpcclient.ReadTextFileResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ReadTextFileResult{}, err
	}
	return client.ReadTextFile(ctx, req)
}

func (c *hostAwareAgentClient) ReadFileChunk(ctx context.Context, req grpcclient.ReadFileChunkRequest) (grpcclient.ReadFileChunkResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ReadFileChunkResult{}, err
	}
	return client.ReadFileChunk(ctx, req)
}

func (c *hostAwareAgentClient) WriteFileChunk(ctx context.Context, req grpcclient.WriteFileChunkRequest) (grpcclient.WriteFileChunkResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.WriteFileChunkResult{}, err
	}
	return client.WriteFileChunk(ctx, req)
}

func (c *hostAwareAgentClient) WriteTextFile(ctx context.Context, req grpcclient.WriteTextFileRequest) (grpcclient.WriteTextFileResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.WriteTextFileResult{}, err
	}
	return client.WriteTextFile(ctx, req)
}

func (c *hostAwareAgentClient) CreateDirectory(ctx context.Context, req grpcclient.CreateDirectoryRequest) (grpcclient.CreateDirectoryResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.CreateDirectoryResult{}, err
	}
	return client.CreateDirectory(ctx, req)
}

func (c *hostAwareAgentClient) DeleteFile(ctx context.Context, req grpcclient.DeleteFileRequest) (grpcclient.DeleteFileResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.DeleteFileResult{}, err
	}
	return client.DeleteFile(ctx, req)
}

func (c *hostAwareAgentClient) RenameFile(ctx context.Context, req grpcclient.RenameFileRequest) (grpcclient.RenameFileResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.RenameFileResult{}, err
	}
	return client.RenameFile(ctx, req)
}

func (c *hostAwareAgentClient) ListServices(ctx context.Context, req grpcclient.ListServicesRequest) (grpcclient.ListServicesResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ListServicesResult{}, err
	}
	return client.ListServices(ctx, req)
}

func (c *hostAwareAgentClient) StartService(ctx context.Context, req grpcclient.ServiceActionRequest) (grpcclient.ServiceActionResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ServiceActionResult{}, err
	}
	return client.StartService(ctx, req)
}

func (c *hostAwareAgentClient) StopService(ctx context.Context, req grpcclient.ServiceActionRequest) (grpcclient.ServiceActionResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ServiceActionResult{}, err
	}
	return client.StopService(ctx, req)
}

func (c *hostAwareAgentClient) RestartService(ctx context.Context, req grpcclient.ServiceActionRequest) (grpcclient.ServiceActionResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ServiceActionResult{}, err
	}
	return client.RestartService(ctx, req)
}

func (c *hostAwareAgentClient) ListDockerContainers(ctx context.Context) (grpcclient.ListDockerContainersResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ListDockerContainersResult{}, err
	}
	return client.ListDockerContainers(ctx)
}

func (c *hostAwareAgentClient) StartDockerContainer(ctx context.Context, req grpcclient.DockerContainerActionRequest) (grpcclient.DockerContainerActionResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.DockerContainerActionResult{}, err
	}
	return client.StartDockerContainer(ctx, req)
}

func (c *hostAwareAgentClient) StopDockerContainer(ctx context.Context, req grpcclient.DockerContainerActionRequest) (grpcclient.DockerContainerActionResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.DockerContainerActionResult{}, err
	}
	return client.StopDockerContainer(ctx, req)
}

func (c *hostAwareAgentClient) RestartDockerContainer(ctx context.Context, req grpcclient.DockerContainerActionRequest) (grpcclient.DockerContainerActionResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.DockerContainerActionResult{}, err
	}
	return client.RestartDockerContainer(ctx, req)
}

func (c *hostAwareAgentClient) ListDockerImages(ctx context.Context) (grpcclient.ListDockerImagesResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ListDockerImagesResult{}, err
	}
	return client.ListDockerImages(ctx)
}

func (c *hostAwareAgentClient) ListCronTasks(ctx context.Context) (grpcclient.ListCronTasksResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.ListCronTasksResult{}, err
	}
	return client.ListCronTasks(ctx)
}

func (c *hostAwareAgentClient) CreateCronTask(ctx context.Context, req grpcclient.CreateCronTaskRequest) (grpcclient.CreateCronTaskResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.CreateCronTaskResult{}, err
	}
	return client.CreateCronTask(ctx, req)
}

func (c *hostAwareAgentClient) UpdateCronTask(ctx context.Context, req grpcclient.UpdateCronTaskRequest) (grpcclient.UpdateCronTaskResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.UpdateCronTaskResult{}, err
	}
	return client.UpdateCronTask(ctx, req)
}

func (c *hostAwareAgentClient) DeleteCronTask(ctx context.Context, req grpcclient.DeleteCronTaskRequest) (grpcclient.DeleteCronTaskResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.DeleteCronTaskResult{}, err
	}
	return client.DeleteCronTask(ctx, req)
}

func (c *hostAwareAgentClient) SetCronTaskEnabled(ctx context.Context, req grpcclient.SetCronTaskEnabledRequest) (grpcclient.SetCronTaskEnabledResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return grpcclient.SetCronTaskEnabledResult{}, err
	}
	return client.SetCronTaskEnabled(ctx, req)
}

func (c *hostAwareAgentClient) Enroll(ctx context.Context, token, hostname, agentVersion string, capabilities []string) (*grpcclient.EnrollmentResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return nil, err
	}
	return client.Enrollment().Enroll(ctx, token, hostname, agentVersion, capabilities)
}

func (c *hostAwareAgentClient) Revoke(ctx context.Context, enrollmentID, reason string) error {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return err
	}
	return client.Enrollment().Revoke(ctx, enrollmentID, reason)
}

func (c *hostAwareAgentClient) RotateCertificates(ctx context.Context, enrollmentID string, reuseKey bool) (*grpcclient.CertificateRotationResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return nil, err
	}
	return client.Enrollment().RotateCertificates(ctx, enrollmentID, reuseKey)
}

func (c *hostAwareAgentClient) ValidateCertificate(ctx context.Context, clientCert string) (*grpcclient.CertificateValidationResult, error) {
	client, err := c.clientForContext(ctx)
	if err != nil {
		return nil, err
	}
	return client.Enrollment().ValidateCertificate(ctx, clientCert)
}

func (c *hostAwareAgentClient) Enrollment() grpcclient.EnrollmentClient {
	client, err := c.clientForContext(context.Background())
	if err != nil {
		return nil
	}
	return client.Enrollment()
}

func (c *hostAwareAgentClient) clientForContext(ctx context.Context) (grpcclient.AgentClient, error) {
	hostID, ok := hostctx.HostID(ctx)
	if !ok || c.hostRepo == nil {
		return c.defaultClient, nil
	}

	host, err := c.hostRepo.GetByID(ctx, hostID)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if host == nil {
		return nil, apperror.ErrHostNotFound
	}
	if host.Status == HostStatusDisabled {
		return nil, apperror.ErrHostDisabled
	}

	target := net.JoinHostPort(host.Address, fmt.Sprintf("%d", host.Port))

	c.mu.RLock()
	cached := c.clients[target]
	c.mu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached = c.clients[target]; cached != nil {
		return cached, nil
	}

	client := grpcclient.New(target, c.timeout)
	c.clients[target] = client
	return client, nil
}
