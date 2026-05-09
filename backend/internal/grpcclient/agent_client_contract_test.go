package grpcclient

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	agentv1 "github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient/pb/proto/agent/v1"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/requestctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestClientCheckHealthViaProtoContract(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{})
	client := New(target, 2*time.Second)

	statusValue, err := client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if statusValue != "SERVING" {
		t.Fatalf("unexpected health status: %s", statusValue)
	}
}

func TestClientCheckHealthPropagatesRequestID(t *testing.T) {
	requestIDCh := make(chan string, 1)
	target := startProtoContractServer(t, protoContractOptions{
		healthObserver: func(ctx context.Context) {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				requestIDCh <- ""
				return
			}
			values := md.Get(requestIDMetadataKey)
			if len(values) == 0 {
				requestIDCh <- ""
				return
			}
			requestIDCh <- values[0]
		},
	})
	client := New(target, 2*time.Second)

	ctx := requestctx.WithRequestID(context.Background(), "req-chain-123")
	_, err := client.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}

	select {
	case observed := <-requestIDCh:
		if observed != "req-chain-123" {
			t.Fatalf("expected propagated request id req-chain-123, got %q", observed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for propagated request id")
	}
}

func TestClientGetRealtimeResourceViaProtoContract(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{})
	client := New(target, 2*time.Second)

	resource, err := client.GetRealtimeResource(context.Background())
	if err != nil {
		t.Fatalf("GetRealtimeResource() error = %v", err)
	}

	if resource.CPUUsagePercent != 18.25 {
		t.Fatalf("unexpected cpu usage: %f", resource.CPUUsagePercent)
	}
	if resource.MemoryUsagePercent != 67.5 {
		t.Fatalf("unexpected memory usage: %f", resource.MemoryUsagePercent)
	}
	if resource.DiskUsagePercent != 71.75 {
		t.Fatalf("unexpected disk usage: %f", resource.DiskUsagePercent)
	}
	if resource.LoadAverage1m != 0.9 || resource.LoadAverage5m != 0.6 || resource.LoadAverage15m != 0.3 {
		t.Fatalf("unexpected load averages: %+v", resource)
	}
}

func TestClientListFilesPreservesProtoFields(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{})
	client := New(target, 2*time.Second)

	result, err := client.ListFiles(context.Background(), ListFilesRequest{Path: "/srv/app"})
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	if result.CurrentPath != "/srv/app" {
		t.Fatalf("unexpected current path: %s", result.CurrentPath)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("unexpected entries count: %d", len(result.Entries))
	}

	configEntry := result.Entries[0]
	if configEntry.Name != "config.yaml" || configEntry.Path != "/srv/app/config.yaml" {
		t.Fatalf("unexpected first entry: %+v", configEntry)
	}
	if configEntry.IsDir {
		t.Fatalf("expected file entry, got directory: %+v", configEntry)
	}
	if configEntry.Size != 256 || configEntry.ModifiedAtUnix != 1710000001 {
		t.Fatalf("unexpected file metadata: %+v", configEntry)
	}

	logsEntry := result.Entries[1]
	if !logsEntry.IsDir || logsEntry.Name != "logs" {
		t.Fatalf("unexpected second entry: %+v", logsEntry)
	}
}

func TestClientListFilesMapsStructuredAgentError(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{
		listFilesError: &agentv1.Error{
			Code:    4001,
			Message: "path access denied",
			Detail:  "normalized path '/root' is outside configured safe roots",
		},
	})
	client := New(target, 2*time.Second)

	_, err := client.ListFiles(context.Background(), ListFilesRequest{Path: "/root"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var agentErr *AgentError
	if !errors.As(err, &agentErr) {
		t.Fatalf("expected AgentError, got %T", err)
	}
	if agentErr.Code != 4001 {
		t.Fatalf("unexpected error code: %d", agentErr.Code)
	}
	if agentErr.Message != "path access denied" {
		t.Fatalf("unexpected error message: %s", agentErr.Message)
	}
	if agentErr.Detail != "normalized path '/root' is outside configured safe roots" {
		t.Fatalf("unexpected error detail: %s", agentErr.Detail)
	}
	if agentErr.IsTransport() {
		t.Fatal("expected structured proto error, got transport error")
	}
	if !strings.Contains(agentErr.Error(), agentErr.Detail) {
		t.Fatalf("expected error string to include detail, got %q", agentErr.Error())
	}
}

func TestClientFileOperationsPreserveProtoFields(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{})
	client := New(target, 2*time.Second)
	ctx := context.Background()

	readText, err := client.ReadTextFile(ctx, ReadTextFileRequest{
		Path:     "/srv/app/config.yaml",
		MaxBytes: 64,
		Encoding: "utf-8",
	})
	if err != nil {
		t.Fatalf("ReadTextFile() error = %v", err)
	}
	if readText.Path != "/srv/app/config.yaml" || readText.Content != "contract text content" ||
		readText.Size != 2048 || !readText.Truncated || readText.Encoding != "utf-8" {
		t.Fatalf("unexpected read text result: %+v", readText)
	}

	readChunk, err := client.ReadFileChunk(ctx, ReadFileChunkRequest{
		Path:   "/srv/app/archive.bin",
		Offset: 32,
		Limit:  4,
	})
	if err != nil {
		t.Fatalf("ReadFileChunk() error = %v", err)
	}
	if readChunk.Path != "/srv/app/archive.bin" || readChunk.Offset != 32 ||
		string(readChunk.Chunk) != "chunk:/srv/app/archive.bin" ||
		readChunk.TotalSize != 4096 || !readChunk.EOF {
		t.Fatalf("unexpected read chunk result: %+v", readChunk)
	}

	writeChunk, err := client.WriteFileChunk(ctx, WriteFileChunkRequest{
		Path:              "/srv/app/archive.bin",
		Offset:            32,
		Chunk:             []byte("contract chunk"),
		CreateIfNotExists: true,
		Truncate:          true,
	})
	if err != nil {
		t.Fatalf("WriteFileChunk() error = %v", err)
	}
	if writeChunk.Path != "/srv/app/archive.bin" || writeChunk.Offset != 32 ||
		writeChunk.WrittenBytes != uint64(len("contract chunk")) || writeChunk.TotalSize != 46 {
		t.Fatalf("unexpected write chunk result: %+v", writeChunk)
	}

	writeText, err := client.WriteTextFile(ctx, WriteTextFileRequest{
		Path:              "/srv/app/config.yaml",
		Content:           "contract text content",
		CreateIfNotExists: true,
		Truncate:          true,
		Encoding:          "utf-8",
	})
	if err != nil {
		t.Fatalf("WriteTextFile() error = %v", err)
	}
	if writeText.Path != "/srv/app/config.yaml" || writeText.WrittenBytes != uint64(len("contract text content")) {
		t.Fatalf("unexpected write text result: %+v", writeText)
	}

	created, err := client.CreateDirectory(ctx, CreateDirectoryRequest{
		Path:          "/srv/app/cache",
		CreateParents: true,
	})
	if err != nil {
		t.Fatalf("CreateDirectory() error = %v", err)
	}
	if created.Path != "/srv/app/cache:parents" {
		t.Fatalf("unexpected create directory result: %+v", created)
	}

	deleted, err := client.DeleteFile(ctx, DeleteFileRequest{
		Path:      "/srv/app/cache",
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if deleted.Path != "/srv/app/cache:recursive" {
		t.Fatalf("unexpected delete file result: %+v", deleted)
	}

	renamed, err := client.RenameFile(ctx, RenameFileRequest{
		SourcePath: "/srv/app/config.yaml",
		TargetPath: "/srv/app/config.old.yaml",
	})
	if err != nil {
		t.Fatalf("RenameFile() error = %v", err)
	}
	if renamed.SourcePath != "/srv/app/config.yaml" ||
		renamed.TargetPath != "/srv/app/config.old.yaml" ||
		renamed.MovedBytes != 42 {
		t.Fatalf("unexpected rename file result: %+v", renamed)
	}
}

func TestClientServiceDockerAndCronContracts(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{})
	client := New(target, 2*time.Second)
	ctx := context.Background()

	services, err := client.ListServices(ctx, ListServicesRequest{Keyword: "ssh"})
	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}
	if len(services.Services) != 2 || services.Services[0].Name != "sshd" ||
		services.Services[0].DisplayName != "SSH daemon for ssh" ||
		services.Services[0].Status != "running" {
		t.Fatalf("unexpected services result: %+v", services)
	}

	for _, tc := range []struct {
		name string
		call func(context.Context, ServiceActionRequest) (ServiceActionResult, error)
		want string
	}{
		{name: "start", call: client.StartService, want: "started"},
		{name: "stop", call: client.StopService, want: "stopped"},
		{name: "restart", call: client.RestartService, want: "restarted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.call(ctx, ServiceActionRequest{Name: "sshd"})
			if err != nil {
				t.Fatalf("%s service error = %v", tc.name, err)
			}
			if result.Name != "sshd" || result.Status != tc.want {
				t.Fatalf("unexpected service action result: %+v", result)
			}
		})
	}

	containers, err := client.ListDockerContainers(ctx)
	if err != nil {
		t.Fatalf("ListDockerContainers() error = %v", err)
	}
	if len(containers.Containers) != 2 || containers.Containers[0].ID != "container-1" ||
		containers.Containers[0].Name != "snowpanel-backend" ||
		containers.Containers[0].Image != "snowpanel/backend:test" ||
		containers.Containers[0].State != "running" ||
		containers.Containers[0].Status != "Up 1 minute" {
		t.Fatalf("unexpected containers result: %+v", containers)
	}

	for _, tc := range []struct {
		name string
		call func(context.Context, DockerContainerActionRequest) (DockerContainerActionResult, error)
		want string
	}{
		{name: "start", call: client.StartDockerContainer, want: "running"},
		{name: "stop", call: client.StopDockerContainer, want: "exited"},
		{name: "restart", call: client.RestartDockerContainer, want: "restarted"},
	} {
		t.Run("docker "+tc.name, func(t *testing.T) {
			result, err := tc.call(ctx, DockerContainerActionRequest{ID: "container-1"})
			if err != nil {
				t.Fatalf("%s docker container error = %v", tc.name, err)
			}
			if result.ID != "container-1" || result.State != tc.want {
				t.Fatalf("unexpected docker action result: %+v", result)
			}
		})
	}

	images, err := client.ListDockerImages(ctx)
	if err != nil {
		t.Fatalf("ListDockerImages() error = %v", err)
	}
	if len(images.Images) != 1 || images.Images[0].ID != "image-1" ||
		len(images.Images[0].RepoTags) != 2 ||
		images.Images[0].RepoTags[0] != "snowpanel/backend:test" ||
		images.Images[0].RepoTags[1] != "snowpanel/backend:latest" ||
		images.Images[0].Size != 123456 {
		t.Fatalf("unexpected images result: %+v", images)
	}

	tasks, err := client.ListCronTasks(ctx)
	if err != nil {
		t.Fatalf("ListCronTasks() error = %v", err)
	}
	if len(tasks.Tasks) != 1 || tasks.Tasks[0].ID != "cron-1" ||
		tasks.Tasks[0].Expression != "*/5 * * * *" ||
		tasks.Tasks[0].Command != "snowpanel check" ||
		!tasks.Tasks[0].Enabled {
		t.Fatalf("unexpected cron list result: %+v", tasks)
	}

	createdTask, err := client.CreateCronTask(ctx, CreateCronTaskRequest{
		Expression: "0 * * * *",
		Command:    "snowpanel rotate",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateCronTask() error = %v", err)
	}
	if createdTask.Task.ID != "created-cron" || createdTask.Task.Expression != "0 * * * *" ||
		createdTask.Task.Command != "snowpanel rotate" || !createdTask.Task.Enabled {
		t.Fatalf("unexpected create cron result: %+v", createdTask)
	}

	updatedTask, err := client.UpdateCronTask(ctx, UpdateCronTaskRequest{
		ID:         "cron-1",
		Expression: "30 * * * *",
		Command:    "snowpanel sync",
		Enabled:    false,
	})
	if err != nil {
		t.Fatalf("UpdateCronTask() error = %v", err)
	}
	if updatedTask.Task.ID != "cron-1" || updatedTask.Task.Expression != "30 * * * *" ||
		updatedTask.Task.Command != "snowpanel sync" || updatedTask.Task.Enabled {
		t.Fatalf("unexpected update cron result: %+v", updatedTask)
	}

	deletedTask, err := client.DeleteCronTask(ctx, DeleteCronTaskRequest{ID: "cron-1"})
	if err != nil {
		t.Fatalf("DeleteCronTask() error = %v", err)
	}
	if deletedTask.ID != "cron-1" {
		t.Fatalf("unexpected delete cron result: %+v", deletedTask)
	}

	enabledTask, err := client.SetCronTaskEnabled(ctx, SetCronTaskEnabledRequest{
		ID:      "cron-1",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("SetCronTaskEnabled() error = %v", err)
	}
	if enabledTask.Task.ID != "cron-1" || !enabledTask.Task.Enabled {
		t.Fatalf("unexpected set cron enabled result: %+v", enabledTask)
	}
}

func TestClientCheckHealthMapsTransportError(t *testing.T) {
	target := startProtoContractServer(t, protoContractOptions{healthTransportCode: codes.Unimplemented})
	client := New(target, 2*time.Second)

	_, err := client.CheckHealth(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var agentErr *AgentError
	if !errors.As(err, &agentErr) {
		t.Fatalf("expected AgentError, got %T", err)
	}
	if !agentErr.IsTransport() {
		t.Fatal("expected transport error")
	}
	if agentErr.Code != agentUnavailableCode {
		t.Fatalf("unexpected transport error code: %d", agentErr.Code)
	}
	if agentErr.GRPCCode != codes.Unimplemented {
		t.Fatalf("unexpected gRPC code: %s", agentErr.GRPCCode)
	}
	if agentErr.Message != "core agent rpc is not implemented" {
		t.Fatalf("unexpected transport message: %s", agentErr.Message)
	}
	if !strings.Contains(agentErr.Detail, "health check not implemented") {
		t.Fatalf("unexpected transport detail: %s", agentErr.Detail)
	}
}

func TestGeneratedGoProtoDescriptorsExposeCriticalServices(t *testing.T) {
	files := agentv1.File_proto_agent_v1_agent_proto
	descriptor := files.Messages().ByName(protoreflect.Name("PathSafetyContext"))
	if descriptor == nil {
		t.Fatal("expected PathSafetyContext descriptor to exist")
	}
	if descriptor.Fields().ByName(protoreflect.Name("allowed_roots")) == nil {
		t.Fatal("expected allowed_roots field to exist")
	}

	for _, serviceName := range []string{
		"HealthService",
		"SystemService",
		"FileService",
		"ServiceManagerService",
		"DockerService",
		"CronService",
	} {
		if files.Services().ByName(protoreflect.Name(serviceName)) == nil {
			t.Fatalf("expected service descriptor %s to exist", serviceName)
		}
	}
}

type protoContractOptions struct {
	healthTransportCode codes.Code
	listFilesError      *agentv1.Error
	healthObserver      func(context.Context)
}

func startProtoContractServer(t *testing.T, opts protoContractOptions) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	server := grpc.NewServer()
	agentv1.RegisterHealthServiceServer(server, &protoContractHealthService{opts: opts})
	agentv1.RegisterSystemServiceServer(server, &protoContractSystemService{})
	agentv1.RegisterFileServiceServer(server, &protoContractFileService{opts: opts})
	agentv1.RegisterServiceManagerServiceServer(server, &protoContractServiceManagerService{})
	agentv1.RegisterDockerServiceServer(server, &protoContractDockerService{})
	agentv1.RegisterCronServiceServer(server, &protoContractCronService{})

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func okProtoError() *agentv1.Error {
	return &agentv1.Error{Code: 0, Message: "ok", Detail: ""}
}

type protoContractHealthService struct {
	agentv1.UnimplementedHealthServiceServer
	opts protoContractOptions
}

func (s *protoContractHealthService) Check(ctx context.Context, _ *agentv1.HealthCheckRequest) (*agentv1.HealthCheckResponse, error) {
	if s.opts.healthObserver != nil {
		s.opts.healthObserver(ctx)
	}
	if s.opts.healthTransportCode != codes.OK {
		return nil, status.Error(s.opts.healthTransportCode, "health check not implemented by fake contract server")
	}
	return &agentv1.HealthCheckResponse{Error: okProtoError(), Status: "SERVING"}, nil
}

type protoContractSystemService struct {
	agentv1.UnimplementedSystemServiceServer
}

func (s *protoContractSystemService) GetSystemOverview(context.Context, *agentv1.GetSystemOverviewRequest) (*agentv1.GetSystemOverviewResponse, error) {
	return &agentv1.GetSystemOverviewResponse{
		Error: okProtoError(),
		Overview: &agentv1.SystemOverview{
			Hostname: "contract-node",
			Os:       "Linux",
			Kernel:   "6.8.12",
			Uptime:   "2h30m",
			Cpu: &agentv1.CPUInfo{
				Model:        "contract-cpu",
				LogicalCores: 8,
				UsagePercent: 12.5,
			},
			Memory: &agentv1.MemoryInfo{
				TotalBytes:   4096,
				UsedBytes:    2048,
				UsagePercent: 50,
			},
			Disks: []*agentv1.DiskInfo{{
				MountPoint:   "/",
				TotalBytes:   10000,
				UsedBytes:    5000,
				UsagePercent: 50,
			}},
		},
	}, nil
}

func (s *protoContractSystemService) GetRealtimeResource(context.Context, *agentv1.GetRealtimeResourceRequest) (*agentv1.GetRealtimeResourceResponse, error) {
	return &agentv1.GetRealtimeResourceResponse{
		Error: okProtoError(),
		Resource: &agentv1.RealtimeResource{
			CpuUsagePercent:    18.25,
			MemoryUsagePercent: 67.5,
			DiskUsagePercent:   71.75,
			LoadAverage_1M:     0.9,
			LoadAverage_5M:     0.6,
			LoadAverage_15M:    0.3,
		},
	}, nil
}

type protoContractFileService struct {
	agentv1.UnimplementedFileServiceServer
	opts protoContractOptions
}

func (s *protoContractFileService) ListFiles(context.Context, *agentv1.ListFilesRequest) (*agentv1.ListFilesResponse, error) {
	if s.opts.listFilesError != nil {
		return &agentv1.ListFilesResponse{Error: s.opts.listFilesError}, nil
	}

	return &agentv1.ListFilesResponse{
		Error:       okProtoError(),
		CurrentPath: "/srv/app",
		Entries: []*agentv1.FileEntry{
			{
				Name:           "config.yaml",
				Path:           "/srv/app/config.yaml",
				IsDir:          false,
				Size:           256,
				ModifiedAtUnix: 1710000001,
			},
			{
				Name:           "logs",
				Path:           "/srv/app/logs",
				IsDir:          true,
				Size:           0,
				ModifiedAtUnix: 1710000002,
			},
		},
	}, nil
}

func (s *protoContractFileService) ReadTextFile(_ context.Context, req *agentv1.ReadTextFileRequest) (*agentv1.ReadTextFileResponse, error) {
	return &agentv1.ReadTextFileResponse{
		Error:     okProtoError(),
		Path:      req.GetPath(),
		Content:   "contract text content",
		Size:      2048,
		Truncated: req.GetMaxBytes() == 64,
		Encoding:  req.GetEncoding(),
	}, nil
}

func (s *protoContractFileService) ReadFileChunk(_ context.Context, req *agentv1.ReadFileChunkRequest) (*agentv1.ReadFileChunkResponse, error) {
	return &agentv1.ReadFileChunkResponse{
		Error:     okProtoError(),
		Path:      req.GetPath(),
		Offset:    req.GetOffset(),
		Chunk:     []byte("chunk:" + req.GetPath()),
		TotalSize: 4096,
		Eof:       req.GetLimit() == 4,
	}, nil
}

func (s *protoContractFileService) WriteFileChunk(_ context.Context, req *agentv1.WriteFileChunkRequest) (*agentv1.WriteFileChunkResponse, error) {
	return &agentv1.WriteFileChunkResponse{
		Error:        okProtoError(),
		Path:         req.GetPath(),
		Offset:       req.GetOffset(),
		WrittenBytes: uint64(len(req.GetChunk())),
		TotalSize:    req.GetOffset() + uint64(len(req.GetChunk())),
	}, nil
}

func (s *protoContractFileService) WriteTextFile(_ context.Context, req *agentv1.WriteTextFileRequest) (*agentv1.WriteTextFileResponse, error) {
	return &agentv1.WriteTextFileResponse{
		Error:        okProtoError(),
		Path:         req.GetPath(),
		WrittenBytes: uint64(len(req.GetContent())),
	}, nil
}

func (s *protoContractFileService) CreateDirectory(_ context.Context, req *agentv1.CreateDirectoryRequest) (*agentv1.CreateDirectoryResponse, error) {
	path := req.GetPath()
	if req.GetCreateParents() {
		path += ":parents"
	}
	return &agentv1.CreateDirectoryResponse{Error: okProtoError(), Path: path}, nil
}

func (s *protoContractFileService) DeleteFile(_ context.Context, req *agentv1.DeleteFileRequest) (*agentv1.DeleteFileResponse, error) {
	path := req.GetPath()
	if req.GetRecursive() {
		path += ":recursive"
	}
	return &agentv1.DeleteFileResponse{Error: okProtoError(), Path: path}, nil
}

func (s *protoContractFileService) RenameFile(_ context.Context, req *agentv1.RenameFileRequest) (*agentv1.RenameFileResponse, error) {
	return &agentv1.RenameFileResponse{
		Error:      okProtoError(),
		SourcePath: req.GetSourcePath(),
		TargetPath: req.GetTargetPath(),
		MovedBytes: 42,
	}, nil
}

type protoContractServiceManagerService struct {
	agentv1.UnimplementedServiceManagerServiceServer
}

func (s *protoContractServiceManagerService) ListServices(_ context.Context, req *agentv1.ListServicesRequest) (*agentv1.ListServicesResponse, error) {
	return &agentv1.ListServicesResponse{
		Error: okProtoError(),
		Services: []*agentv1.ServiceInfo{
			{Name: "sshd", DisplayName: "SSH daemon for " + req.GetKeyword(), Status: "running"},
			{Name: "postgresql", DisplayName: "PostgreSQL", Status: "inactive"},
		},
	}, nil
}

func (s *protoContractServiceManagerService) StartService(ctx context.Context, req *agentv1.ServiceActionRequest) (*agentv1.ServiceActionResponse, error) {
	return serviceActionResponse(req.GetName(), "started"), nil
}

func (s *protoContractServiceManagerService) StopService(ctx context.Context, req *agentv1.ServiceActionRequest) (*agentv1.ServiceActionResponse, error) {
	return serviceActionResponse(req.GetName(), "stopped"), nil
}

func (s *protoContractServiceManagerService) RestartService(ctx context.Context, req *agentv1.ServiceActionRequest) (*agentv1.ServiceActionResponse, error) {
	return serviceActionResponse(req.GetName(), "restarted"), nil
}

func serviceActionResponse(name string, status string) *agentv1.ServiceActionResponse {
	return &agentv1.ServiceActionResponse{Error: okProtoError(), Name: name, Status: status}
}

type protoContractDockerService struct {
	agentv1.UnimplementedDockerServiceServer
}

func (s *protoContractDockerService) ListContainers(context.Context, *agentv1.ListDockerContainersRequest) (*agentv1.ListDockerContainersResponse, error) {
	return &agentv1.ListDockerContainersResponse{
		Error: okProtoError(),
		Containers: []*agentv1.DockerContainerInfo{
			{
				Id:     "container-1",
				Name:   "snowpanel-backend",
				Image:  "snowpanel/backend:test",
				State:  "running",
				Status: "Up 1 minute",
			},
			{
				Id:     "container-2",
				Name:   "snowpanel-frontend",
				Image:  "snowpanel/frontend:test",
				State:  "exited",
				Status: "Exited",
			},
		},
	}, nil
}

func (s *protoContractDockerService) StartContainer(ctx context.Context, req *agentv1.DockerContainerActionRequest) (*agentv1.DockerContainerActionResponse, error) {
	return dockerActionResponse(req.GetId(), "running"), nil
}

func (s *protoContractDockerService) StopContainer(ctx context.Context, req *agentv1.DockerContainerActionRequest) (*agentv1.DockerContainerActionResponse, error) {
	return dockerActionResponse(req.GetId(), "exited"), nil
}

func (s *protoContractDockerService) RestartContainer(ctx context.Context, req *agentv1.DockerContainerActionRequest) (*agentv1.DockerContainerActionResponse, error) {
	return dockerActionResponse(req.GetId(), "restarted"), nil
}

func (s *protoContractDockerService) ListImages(context.Context, *agentv1.ListDockerImagesRequest) (*agentv1.ListDockerImagesResponse, error) {
	return &agentv1.ListDockerImagesResponse{
		Error: okProtoError(),
		Images: []*agentv1.DockerImageInfo{
			{
				Id:       "image-1",
				RepoTags: []string{"snowpanel/backend:test", "snowpanel/backend:latest"},
				Size:     123456,
			},
		},
	}, nil
}

func dockerActionResponse(id string, state string) *agentv1.DockerContainerActionResponse {
	return &agentv1.DockerContainerActionResponse{Error: okProtoError(), Id: id, State: state}
}

type protoContractCronService struct {
	agentv1.UnimplementedCronServiceServer
}

func (s *protoContractCronService) ListCronTasks(context.Context, *agentv1.ListCronTasksRequest) (*agentv1.ListCronTasksResponse, error) {
	return &agentv1.ListCronTasksResponse{
		Error: okProtoError(),
		Tasks: []*agentv1.CronTask{
			{Id: "cron-1", Expression: "*/5 * * * *", Command: "snowpanel check", Enabled: true},
		},
	}, nil
}

func (s *protoContractCronService) CreateCronTask(_ context.Context, req *agentv1.CreateCronTaskRequest) (*agentv1.CreateCronTaskResponse, error) {
	return &agentv1.CreateCronTaskResponse{
		Error: okProtoError(),
		Task: &agentv1.CronTask{
			Id:         "created-cron",
			Expression: req.GetExpression(),
			Command:    req.GetCommand(),
			Enabled:    req.GetEnabled(),
		},
	}, nil
}

func (s *protoContractCronService) UpdateCronTask(_ context.Context, req *agentv1.UpdateCronTaskRequest) (*agentv1.UpdateCronTaskResponse, error) {
	return &agentv1.UpdateCronTaskResponse{
		Error: okProtoError(),
		Task: &agentv1.CronTask{
			Id:         req.GetId(),
			Expression: req.GetExpression(),
			Command:    req.GetCommand(),
			Enabled:    req.GetEnabled(),
		},
	}, nil
}

func (s *protoContractCronService) DeleteCronTask(_ context.Context, req *agentv1.DeleteCronTaskRequest) (*agentv1.DeleteCronTaskResponse, error) {
	return &agentv1.DeleteCronTaskResponse{Error: okProtoError(), Id: req.GetId()}, nil
}

func (s *protoContractCronService) SetCronTaskEnabled(_ context.Context, req *agentv1.SetCronTaskEnabledRequest) (*agentv1.SetCronTaskEnabledResponse, error) {
	return &agentv1.SetCronTaskEnabledResponse{
		Error: okProtoError(),
		Task: &agentv1.CronTask{
			Id:         req.GetId(),
			Expression: "*/5 * * * *",
			Command:    "snowpanel check",
			Enabled:    req.GetEnabled(),
		},
	}, nil
}
