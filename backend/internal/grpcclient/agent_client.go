package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	agentv1 "github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient/pb/proto/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	agentUnavailableCode    int32 = 3001
	agentInvalidPayloadCode int32 = 3002
)

type AgentError struct {
	Code     int32
	Message  string
	Detail   string
	GRPCCode codes.Code
	Cause    error
}

func (e *AgentError) Error() string {
	if e == nil {
		return "agent error"
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "core agent request failed"
	}

	detail := strings.TrimSpace(e.Detail)
	if detail == "" {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %v", message, e.Cause)
		}
		return message
	}
	return fmt.Sprintf("%s: %s", message, detail)
}

func (e *AgentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *AgentError) IsTransport() bool {
	return e != nil && e.Cause != nil
}

type SystemOverview struct {
	Hostname           string
	OS                 string
	Kernel             string
	Uptime             string
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	DiskUsagePercent   float64
}

type RealtimeResource struct {
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	DiskUsagePercent   float64
	LoadAverage1m      float64
	LoadAverage5m      float64
	LoadAverage15m     float64
}

type FileEntry struct {
	Name           string
	Path           string
	IsDir          bool
	Size           uint64
	ModifiedAtUnix int64
}

type ListFilesRequest struct {
	Path string
}

type ListFilesResult struct {
	CurrentPath string
	Entries     []FileEntry
}

type ReadTextFileRequest struct {
	Path     string
	MaxBytes int64
	Encoding string
}

type ReadTextFileResult struct {
	Path      string
	Content   string
	Size      uint64
	Truncated bool
	Encoding  string
}

type WriteTextFileRequest struct {
	Path              string
	Content           string
	CreateIfNotExists bool
	Truncate          bool
	Encoding          string
}

type WriteTextFileResult struct {
	Path         string
	WrittenBytes uint64
}

type CreateDirectoryRequest struct {
	Path          string
	CreateParents bool
}

type CreateDirectoryResult struct {
	Path string
}

type DeleteFileRequest struct {
	Path      string
	Recursive bool
}

type DeleteFileResult struct {
	Path string
}

type ServiceInfo struct {
	Name        string
	DisplayName string
	Status      string
}

type ListServicesRequest struct {
	Keyword string
}

type ListServicesResult struct {
	Services []ServiceInfo
}

type ServiceActionRequest struct {
	Name string
}

type ServiceActionResult struct {
	Name   string
	Status string
}

type DockerContainerInfo struct {
	ID     string
	Name   string
	Image  string
	State  string
	Status string
}

type ListDockerContainersResult struct {
	Containers []DockerContainerInfo
}

type DockerContainerActionRequest struct {
	ID string
}

type DockerContainerActionResult struct {
	ID    string
	State string
}

type DockerImageInfo struct {
	ID       string
	RepoTags []string
	Size     uint64
}

type ListDockerImagesResult struct {
	Images []DockerImageInfo
}

type CronTask struct {
	ID         string
	Expression string
	Command    string
	Enabled    bool
}

type ListCronTasksResult struct {
	Tasks []CronTask
}

type CreateCronTaskRequest struct {
	Expression string
	Command    string
	Enabled    bool
}

type CreateCronTaskResult struct {
	Task CronTask
}

type UpdateCronTaskRequest struct {
	ID         string
	Expression string
	Command    string
	Enabled    bool
}

type UpdateCronTaskResult struct {
	Task CronTask
}

type DeleteCronTaskRequest struct {
	ID string
}

type DeleteCronTaskResult struct {
	ID string
}

type SetCronTaskEnabledRequest struct {
	ID      string
	Enabled bool
}

type SetCronTaskEnabledResult struct {
	Task CronTask
}

type AgentClient interface {
	CheckHealth(ctx context.Context) (string, error)
	GetSystemOverview(ctx context.Context) (SystemOverview, error)
	GetRealtimeResource(ctx context.Context) (RealtimeResource, error)
	ListFiles(ctx context.Context, req ListFilesRequest) (ListFilesResult, error)
	ReadTextFile(ctx context.Context, req ReadTextFileRequest) (ReadTextFileResult, error)
	WriteTextFile(ctx context.Context, req WriteTextFileRequest) (WriteTextFileResult, error)
	CreateDirectory(ctx context.Context, req CreateDirectoryRequest) (CreateDirectoryResult, error)
	DeleteFile(ctx context.Context, req DeleteFileRequest) (DeleteFileResult, error)
	ListServices(ctx context.Context, req ListServicesRequest) (ListServicesResult, error)
	StartService(ctx context.Context, req ServiceActionRequest) (ServiceActionResult, error)
	StopService(ctx context.Context, req ServiceActionRequest) (ServiceActionResult, error)
	RestartService(ctx context.Context, req ServiceActionRequest) (ServiceActionResult, error)
	ListDockerContainers(ctx context.Context) (ListDockerContainersResult, error)
	StartDockerContainer(ctx context.Context, req DockerContainerActionRequest) (DockerContainerActionResult, error)
	StopDockerContainer(ctx context.Context, req DockerContainerActionRequest) (DockerContainerActionResult, error)
	RestartDockerContainer(ctx context.Context, req DockerContainerActionRequest) (DockerContainerActionResult, error)
	ListDockerImages(ctx context.Context) (ListDockerImagesResult, error)
	ListCronTasks(ctx context.Context) (ListCronTasksResult, error)
	CreateCronTask(ctx context.Context, req CreateCronTaskRequest) (CreateCronTaskResult, error)
	UpdateCronTask(ctx context.Context, req UpdateCronTaskRequest) (UpdateCronTaskResult, error)
	DeleteCronTask(ctx context.Context, req DeleteCronTaskRequest) (DeleteCronTaskResult, error)
	SetCronTaskEnabled(ctx context.Context, req SetCronTaskEnabledRequest) (SetCronTaskEnabledResult, error)
}

type Client struct {
	target  string
	timeout time.Duration
}

func New(target string, timeout time.Duration) *Client {
	return &Client{
		target:  target,
		timeout: timeout,
	}
}

func (c *Client) Target() string {
	return c.target
}

func (c *Client) Timeout() time.Duration {
	return c.timeout
}

func (c *Client) CheckHealth(ctx context.Context) (string, error) {
	result := ""
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewHealthServiceClient(conn)
		resp, err := client.Check(callCtx, &agentv1.HealthCheckRequest{})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		result = strings.TrimSpace(resp.GetStatus())
		if result == "" {
			result = "UNKNOWN"
		}
		return nil
	})
	return result, err
}

func (c *Client) GetSystemOverview(ctx context.Context) (SystemOverview, error) {
	result := SystemOverview{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewSystemServiceClient(conn)
		resp, err := client.GetSystemOverview(callCtx, &agentv1.GetSystemOverviewRequest{})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		overview := resp.GetOverview()
		if overview == nil {
			return invalidPayloadError("system overview payload is empty")
		}

		result = SystemOverview{
			Hostname:           overview.GetHostname(),
			OS:                 overview.GetOs(),
			Kernel:             overview.GetKernel(),
			Uptime:             overview.GetUptime(),
			CPUUsagePercent:    overview.GetCpu().GetUsagePercent(),
			MemoryUsagePercent: overview.GetMemory().GetUsagePercent(),
			DiskUsagePercent:   extractDiskUsagePercent(overview.GetDisks()),
		}
		return nil
	})
	return result, err
}

func (c *Client) GetRealtimeResource(ctx context.Context) (RealtimeResource, error) {
	result := RealtimeResource{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewSystemServiceClient(conn)
		resp, err := client.GetRealtimeResource(callCtx, &agentv1.GetRealtimeResourceRequest{})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		resource := resp.GetResource()
		if resource == nil {
			return invalidPayloadError("realtime resource payload is empty")
		}

		result = RealtimeResource{
			CPUUsagePercent:    resource.GetCpuUsagePercent(),
			MemoryUsagePercent: resource.GetMemoryUsagePercent(),
			DiskUsagePercent:   resource.GetDiskUsagePercent(),
			LoadAverage1m:      resource.GetLoadAverage_1M(),
			LoadAverage5m:      resource.GetLoadAverage_5M(),
			LoadAverage15m:     resource.GetLoadAverage_15M(),
		}
		return nil
	})
	return result, err
}

func (c *Client) ListFiles(ctx context.Context, req ListFilesRequest) (ListFilesResult, error) {
	result := ListFilesResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewFileServiceClient(conn)
		resp, err := client.ListFiles(callCtx, &agentv1.ListFilesRequest{
			Path: req.Path,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		entries := make([]FileEntry, 0, len(resp.GetEntries()))
		for _, entry := range resp.GetEntries() {
			entries = append(entries, FileEntry{
				Name:           entry.GetName(),
				Path:           entry.GetPath(),
				IsDir:          entry.GetIsDir(),
				Size:           entry.GetSize(),
				ModifiedAtUnix: entry.GetModifiedAtUnix(),
			})
		}

		result = ListFilesResult{
			CurrentPath: resp.GetCurrentPath(),
			Entries:     entries,
		}
		return nil
	})
	return result, err
}

func (c *Client) ReadTextFile(ctx context.Context, req ReadTextFileRequest) (ReadTextFileResult, error) {
	result := ReadTextFileResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewFileServiceClient(conn)
		resp, err := client.ReadTextFile(callCtx, &agentv1.ReadTextFileRequest{
			Path:     req.Path,
			MaxBytes: req.MaxBytes,
			Encoding: req.Encoding,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		result = ReadTextFileResult{
			Path:      resp.GetPath(),
			Content:   resp.GetContent(),
			Size:      resp.GetSize(),
			Truncated: resp.GetTruncated(),
			Encoding:  resp.GetEncoding(),
		}
		return nil
	})
	return result, err
}

func (c *Client) WriteTextFile(ctx context.Context, req WriteTextFileRequest) (WriteTextFileResult, error) {
	result := WriteTextFileResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewFileServiceClient(conn)
		resp, err := client.WriteTextFile(callCtx, &agentv1.WriteTextFileRequest{
			Path:              req.Path,
			Content:           req.Content,
			CreateIfNotExists: req.CreateIfNotExists,
			Truncate:          req.Truncate,
			Encoding:          req.Encoding,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		result = WriteTextFileResult{
			Path:         resp.GetPath(),
			WrittenBytes: resp.GetWrittenBytes(),
		}
		return nil
	})
	return result, err
}

func (c *Client) CreateDirectory(
	ctx context.Context,
	req CreateDirectoryRequest,
) (CreateDirectoryResult, error) {
	result := CreateDirectoryResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewFileServiceClient(conn)
		resp, err := client.CreateDirectory(callCtx, &agentv1.CreateDirectoryRequest{
			Path:          req.Path,
			CreateParents: req.CreateParents,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		result = CreateDirectoryResult{
			Path: resp.GetPath(),
		}
		return nil
	})
	return result, err
}

func (c *Client) DeleteFile(ctx context.Context, req DeleteFileRequest) (DeleteFileResult, error) {
	result := DeleteFileResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewFileServiceClient(conn)
		resp, err := client.DeleteFile(callCtx, &agentv1.DeleteFileRequest{
			Path:      req.Path,
			Recursive: req.Recursive,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		result = DeleteFileResult{
			Path: resp.GetPath(),
		}
		return nil
	})
	return result, err
}

func (c *Client) ListServices(ctx context.Context, req ListServicesRequest) (ListServicesResult, error) {
	result := ListServicesResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewServiceManagerServiceClient(conn)
		resp, err := client.ListServices(callCtx, &agentv1.ListServicesRequest{
			Keyword: req.Keyword,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		services := make([]ServiceInfo, 0, len(resp.GetServices()))
		for _, item := range resp.GetServices() {
			services = append(services, ServiceInfo{
				Name:        item.GetName(),
				DisplayName: item.GetDisplayName(),
				Status:      item.GetStatus(),
			})
		}
		result = ListServicesResult{Services: services}
		return nil
	})
	return result, err
}

func (c *Client) StartService(ctx context.Context, req ServiceActionRequest) (ServiceActionResult, error) {
	return c.runServiceAction(ctx, req.Name, func(
		callCtx context.Context,
		client agentv1.ServiceManagerServiceClient,
		request *agentv1.ServiceActionRequest,
	) (*agentv1.ServiceActionResponse, error) {
		return client.StartService(callCtx, request)
	})
}

func (c *Client) StopService(ctx context.Context, req ServiceActionRequest) (ServiceActionResult, error) {
	return c.runServiceAction(ctx, req.Name, func(
		callCtx context.Context,
		client agentv1.ServiceManagerServiceClient,
		request *agentv1.ServiceActionRequest,
	) (*agentv1.ServiceActionResponse, error) {
		return client.StopService(callCtx, request)
	})
}

func (c *Client) RestartService(ctx context.Context, req ServiceActionRequest) (ServiceActionResult, error) {
	return c.runServiceAction(ctx, req.Name, func(
		callCtx context.Context,
		client agentv1.ServiceManagerServiceClient,
		request *agentv1.ServiceActionRequest,
	) (*agentv1.ServiceActionResponse, error) {
		return client.RestartService(callCtx, request)
	})
}

func (c *Client) ListDockerContainers(ctx context.Context) (ListDockerContainersResult, error) {
	result := ListDockerContainersResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewDockerServiceClient(conn)
		resp, err := client.ListContainers(callCtx, &agentv1.ListDockerContainersRequest{})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		containers := make([]DockerContainerInfo, 0, len(resp.GetContainers()))
		for _, item := range resp.GetContainers() {
			containers = append(containers, DockerContainerInfo{
				ID:     item.GetId(),
				Name:   item.GetName(),
				Image:  item.GetImage(),
				State:  item.GetState(),
				Status: item.GetStatus(),
			})
		}
		result = ListDockerContainersResult{Containers: containers}
		return nil
	})
	return result, err
}

func (c *Client) StartDockerContainer(
	ctx context.Context,
	req DockerContainerActionRequest,
) (DockerContainerActionResult, error) {
	return c.runDockerAction(ctx, req.ID, func(
		callCtx context.Context,
		client agentv1.DockerServiceClient,
		request *agentv1.DockerContainerActionRequest,
	) (*agentv1.DockerContainerActionResponse, error) {
		return client.StartContainer(callCtx, request)
	})
}

func (c *Client) StopDockerContainer(
	ctx context.Context,
	req DockerContainerActionRequest,
) (DockerContainerActionResult, error) {
	return c.runDockerAction(ctx, req.ID, func(
		callCtx context.Context,
		client agentv1.DockerServiceClient,
		request *agentv1.DockerContainerActionRequest,
	) (*agentv1.DockerContainerActionResponse, error) {
		return client.StopContainer(callCtx, request)
	})
}

func (c *Client) RestartDockerContainer(
	ctx context.Context,
	req DockerContainerActionRequest,
) (DockerContainerActionResult, error) {
	return c.runDockerAction(ctx, req.ID, func(
		callCtx context.Context,
		client agentv1.DockerServiceClient,
		request *agentv1.DockerContainerActionRequest,
	) (*agentv1.DockerContainerActionResponse, error) {
		return client.RestartContainer(callCtx, request)
	})
}

func (c *Client) ListDockerImages(ctx context.Context) (ListDockerImagesResult, error) {
	result := ListDockerImagesResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewDockerServiceClient(conn)
		resp, err := client.ListImages(callCtx, &agentv1.ListDockerImagesRequest{})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		images := make([]DockerImageInfo, 0, len(resp.GetImages()))
		for _, item := range resp.GetImages() {
			images = append(images, DockerImageInfo{
				ID:       item.GetId(),
				RepoTags: item.GetRepoTags(),
				Size:     item.GetSize(),
			})
		}
		result = ListDockerImagesResult{Images: images}
		return nil
	})
	return result, err
}

func (c *Client) ListCronTasks(ctx context.Context) (ListCronTasksResult, error) {
	result := ListCronTasksResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewCronServiceClient(conn)
		resp, err := client.ListCronTasks(callCtx, &agentv1.ListCronTasksRequest{})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}

		tasks := make([]CronTask, 0, len(resp.GetTasks()))
		for _, item := range resp.GetTasks() {
			tasks = append(tasks, CronTask{
				ID:         item.GetId(),
				Expression: item.GetExpression(),
				Command:    item.GetCommand(),
				Enabled:    item.GetEnabled(),
			})
		}
		result = ListCronTasksResult{Tasks: tasks}
		return nil
	})
	return result, err
}

func (c *Client) CreateCronTask(ctx context.Context, req CreateCronTaskRequest) (CreateCronTaskResult, error) {
	result := CreateCronTaskResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewCronServiceClient(conn)
		resp, err := client.CreateCronTask(callCtx, &agentv1.CreateCronTaskRequest{
			Expression: req.Expression,
			Command:    req.Command,
			Enabled:    req.Enabled,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		task, err := cronTaskFromProto(resp.GetTask())
		if err != nil {
			return err
		}
		result = CreateCronTaskResult{Task: task}
		return nil
	})
	return result, err
}

func (c *Client) UpdateCronTask(ctx context.Context, req UpdateCronTaskRequest) (UpdateCronTaskResult, error) {
	result := UpdateCronTaskResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewCronServiceClient(conn)
		resp, err := client.UpdateCronTask(callCtx, &agentv1.UpdateCronTaskRequest{
			Id:         req.ID,
			Expression: req.Expression,
			Command:    req.Command,
			Enabled:    req.Enabled,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		task, err := cronTaskFromProto(resp.GetTask())
		if err != nil {
			return err
		}
		result = UpdateCronTaskResult{Task: task}
		return nil
	})
	return result, err
}

func (c *Client) DeleteCronTask(ctx context.Context, req DeleteCronTaskRequest) (DeleteCronTaskResult, error) {
	result := DeleteCronTaskResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewCronServiceClient(conn)
		resp, err := client.DeleteCronTask(callCtx, &agentv1.DeleteCronTaskRequest{
			Id: req.ID,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		result = DeleteCronTaskResult{ID: resp.GetId()}
		return nil
	})
	return result, err
}

func (c *Client) SetCronTaskEnabled(
	ctx context.Context,
	req SetCronTaskEnabledRequest,
) (SetCronTaskEnabledResult, error) {
	result := SetCronTaskEnabledResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewCronServiceClient(conn)
		resp, err := client.SetCronTaskEnabled(callCtx, &agentv1.SetCronTaskEnabledRequest{
			Id:      req.ID,
			Enabled: req.Enabled,
		})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		task, err := cronTaskFromProto(resp.GetTask())
		if err != nil {
			return err
		}
		result = SetCronTaskEnabledResult{Task: task}
		return nil
	})
	return result, err
}

func (c *Client) runServiceAction(
	ctx context.Context,
	name string,
	rpc func(
		callCtx context.Context,
		client agentv1.ServiceManagerServiceClient,
		request *agentv1.ServiceActionRequest,
	) (*agentv1.ServiceActionResponse, error),
) (ServiceActionResult, error) {
	result := ServiceActionResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewServiceManagerServiceClient(conn)
		resp, err := rpc(callCtx, client, &agentv1.ServiceActionRequest{Name: name})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		result = ServiceActionResult{
			Name:   resp.GetName(),
			Status: resp.GetStatus(),
		}
		return nil
	})
	return result, err
}

func (c *Client) runDockerAction(
	ctx context.Context,
	id string,
	rpc func(
		callCtx context.Context,
		client agentv1.DockerServiceClient,
		request *agentv1.DockerContainerActionRequest,
	) (*agentv1.DockerContainerActionResponse, error),
) (DockerContainerActionResult, error) {
	result := DockerContainerActionResult{}
	err := c.invoke(ctx, func(callCtx context.Context, conn *grpc.ClientConn) error {
		client := agentv1.NewDockerServiceClient(conn)
		resp, err := rpc(callCtx, client, &agentv1.DockerContainerActionRequest{Id: id})
		if err != nil {
			return err
		}
		if err := responseError(resp.GetError()); err != nil {
			return err
		}
		result = DockerContainerActionResult{
			ID:    resp.GetId(),
			State: resp.GetState(),
		}
		return nil
	})
	return result, err
}

func (c *Client) invoke(ctx context.Context, call func(context.Context, *grpc.ClientConn) error) error {
	callCtx := ctx
	cancel := func() {}
	if c.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	defer cancel()

	conn, err := grpc.DialContext(
		callCtx,
		c.target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return transportError(err)
	}
	defer conn.Close()

	if err := call(callCtx, conn); err != nil {
		var agentErr *AgentError
		if errors.As(err, &agentErr) {
			return agentErr
		}
		return transportError(err)
	}

	return nil
}

func responseError(err *agentv1.Error) error {
	if err == nil {
		return invalidPayloadError("missing error envelope")
	}
	if err.GetCode() == 0 {
		return nil
	}

	message := strings.TrimSpace(err.GetMessage())
	if message == "" {
		message = "core agent request failed"
	}

	return &AgentError{
		Code:     err.GetCode(),
		Message:  message,
		Detail:   strings.TrimSpace(err.GetDetail()),
		GRPCCode: codes.OK,
	}
}

func transportError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return &AgentError{
			Code:     agentUnavailableCode,
			Message:  "core agent unavailable",
			Detail:   err.Error(),
			GRPCCode: codes.Unknown,
			Cause:    err,
		}
	}

	message := "core agent unavailable"
	switch st.Code() {
	case codes.InvalidArgument:
		message = "invalid request to core agent"
	case codes.PermissionDenied:
		message = "core agent permission denied"
	case codes.Unimplemented:
		message = "core agent rpc is not implemented"
	}

	return &AgentError{
		Code:     agentUnavailableCode,
		Message:  message,
		Detail:   st.Message(),
		GRPCCode: st.Code(),
		Cause:    err,
	}
}

func invalidPayloadError(detail string) error {
	return &AgentError{
		Code:     agentInvalidPayloadCode,
		Message:  "invalid core agent response",
		Detail:   detail,
		GRPCCode: codes.Internal,
	}
}

func extractDiskUsagePercent(disks []*agentv1.DiskInfo) float64 {
	if len(disks) == 0 {
		return 0
	}

	for _, disk := range disks {
		if disk.GetMountPoint() == "/" {
			return disk.GetUsagePercent()
		}
	}

	maxUsage := 0.0
	for _, disk := range disks {
		maxUsage = math.Max(maxUsage, disk.GetUsagePercent())
	}
	return maxUsage
}

func cronTaskFromProto(task *agentv1.CronTask) (CronTask, error) {
	if task == nil {
		return CronTask{}, invalidPayloadError("missing cron task payload")
	}
	return CronTask{
		ID:         task.GetId(),
		Expression: task.GetExpression(),
		Command:    task.GetCommand(),
		Enabled:    task.GetEnabled(),
	}, nil
}
