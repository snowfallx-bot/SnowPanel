package service

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	agentv1 "github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient/pb/proto/agent/v1"
	"google.golang.org/grpc"
)

func TestDashboardService_GetSummaryViaGRPC(t *testing.T) {
	target := startFakeAgentServer(t)
	client := grpcclient.New(target, 2*time.Second)
	service := NewDashboardService(client)

	summary, err := service.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}

	if summary.Hostname != "test-node" {
		t.Fatalf("unexpected hostname: %s", summary.Hostname)
	}
	if summary.CPUUsage != 23.5 {
		t.Fatalf("unexpected cpu usage: %f", summary.CPUUsage)
	}
	if summary.DiskUsage != 61 {
		t.Fatalf("unexpected disk usage: %f", summary.DiskUsage)
	}
}

func TestFileService_ListFilesViaGRPC(t *testing.T) {
	target := startFakeAgentServer(t)
	client := grpcclient.New(target, 2*time.Second)
	service := NewFileService(client)

	result, err := service.ListFiles(context.Background(), dto.ListFilesQuery{Path: "/tmp"})
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	if result.CurrentPath != "/tmp" {
		t.Fatalf("unexpected current path: %s", result.CurrentPath)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("unexpected entries count: %d", len(result.Entries))
	}
	if result.Entries[0].Name != "demo.txt" {
		t.Fatalf("unexpected file name: %s", result.Entries[0].Name)
	}
}

func TestFileService_DownloadFileViaGRPC(t *testing.T) {
	target := startFakeAgentServer(t)
	client := grpcclient.New(target, 2*time.Second)
	service := NewFileService(client)

	downloaded := make([]byte, 0, 64)
	result, err := service.DownloadFile(
		context.Background(),
		dto.DownloadFileQuery{Path: "/tmp/demo.txt"},
		func(chunk []byte) error {
			downloaded = append(downloaded, chunk...)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("DownloadFile() error = %v", err)
	}
	if result.Path != "/tmp/demo.txt" {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if result.TotalSize != 21 {
		t.Fatalf("unexpected total size: %d", result.TotalSize)
	}
	if result.DownloadedBytes != 21 {
		t.Fatalf("unexpected downloaded bytes: %d", result.DownloadedBytes)
	}
	if string(downloaded) != "hello from fake agent" {
		t.Fatalf("unexpected content: %s", string(downloaded))
	}
}

func TestServiceManagerService_ListServicesViaGRPC(t *testing.T) {
	target := startFakeAgentServer(t)
	client := grpcclient.New(target, 2*time.Second)
	service := NewServiceManagerService(client)

	result, err := service.ListServices(context.Background(), dto.ListServicesQuery{})
	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}

	if len(result.Services) != 1 {
		t.Fatalf("unexpected services count: %d", len(result.Services))
	}
	if result.Services[0].Name != "nginx.service" {
		t.Fatalf("unexpected service name: %s", result.Services[0].Name)
	}
}

func TestDockerService_ListContainersViaGRPC(t *testing.T) {
	target := startFakeAgentServer(t)
	client := grpcclient.New(target, 2*time.Second)
	service := NewDockerService(client)

	result, err := service.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}

	if len(result.Containers) != 1 {
		t.Fatalf("unexpected containers count: %d", len(result.Containers))
	}
	if result.Containers[0].ID != "abc123" {
		t.Fatalf("unexpected container id: %s", result.Containers[0].ID)
	}
}

func TestCronService_ListTasksViaGRPC(t *testing.T) {
	target := startFakeAgentServer(t)
	client := grpcclient.New(target, 2*time.Second)
	service := NewCronService(client)

	result, err := service.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("unexpected tasks count: %d", len(result.Tasks))
	}
	if result.Tasks[0].Expression != "*/5 * * * *" {
		t.Fatalf("unexpected cron expression: %s", result.Tasks[0].Expression)
	}
}

func startFakeAgentServer(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	server := grpc.NewServer()
	agentv1.RegisterSystemServiceServer(server, &fakeSystemService{})
	agentv1.RegisterFileServiceServer(server, &fakeFileService{})
	agentv1.RegisterServiceManagerServiceServer(server, &fakeServiceManagerService{})
	agentv1.RegisterDockerServiceServer(server, &fakeDockerService{})
	agentv1.RegisterCronServiceServer(server, &fakeCronService{})

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func okError() *agentv1.Error {
	return &agentv1.Error{
		Code:    0,
		Message: "ok",
		Detail:  "",
	}
}

type fakeSystemService struct {
	agentv1.UnimplementedSystemServiceServer
}

func (s *fakeSystemService) GetSystemOverview(
	context.Context,
	*agentv1.GetSystemOverviewRequest,
) (*agentv1.GetSystemOverviewResponse, error) {
	return &agentv1.GetSystemOverviewResponse{
		Error: okError(),
		Overview: &agentv1.SystemOverview{
			Hostname: "test-node",
			Os:       "Linux",
			Kernel:   "6.8.0",
			Uptime:   "1h",
			Cpu: &agentv1.CPUInfo{
				Model:        "test-cpu",
				LogicalCores: 8,
				UsagePercent: 20,
			},
			Memory: &agentv1.MemoryInfo{
				TotalBytes:   1024,
				UsedBytes:    512,
				UsagePercent: 50,
			},
			Disks: []*agentv1.DiskInfo{
				{
					MountPoint:   "/",
					TotalBytes:   1000,
					UsedBytes:    500,
					UsagePercent: 50,
				},
			},
		},
	}, nil
}

func (s *fakeSystemService) GetRealtimeResource(
	context.Context,
	*agentv1.GetRealtimeResourceRequest,
) (*agentv1.GetRealtimeResourceResponse, error) {
	return &agentv1.GetRealtimeResourceResponse{
		Error: okError(),
		Resource: &agentv1.RealtimeResource{
			CpuUsagePercent:    23.5,
			MemoryUsagePercent: 51.2,
			DiskUsagePercent:   61,
			LoadAverage_1M:     0.5,
			LoadAverage_5M:     0.4,
			LoadAverage_15M:    0.3,
		},
	}, nil
}

type fakeFileService struct {
	agentv1.UnimplementedFileServiceServer
}

func (s *fakeFileService) ListFiles(
	context.Context,
	*agentv1.ListFilesRequest,
) (*agentv1.ListFilesResponse, error) {
	return &agentv1.ListFilesResponse{
		Error:       okError(),
		CurrentPath: "/tmp",
		Entries: []*agentv1.FileEntry{
			{
				Name:           "demo.txt",
				Path:           "/tmp/demo.txt",
				IsDir:          false,
				Size:           128,
				ModifiedAtUnix: 1710000000,
			},
		},
	}, nil
}

func (s *fakeFileService) ReadTextFile(
	context.Context,
	*agentv1.ReadTextFileRequest,
) (*agentv1.ReadTextFileResponse, error) {
	return &agentv1.ReadTextFileResponse{
		Error:     okError(),
		Path:      "/tmp/demo.txt",
		Content:   "hello from fake agent",
		Size:      21,
		Truncated: false,
		Encoding:  "utf-8",
	}, nil
}

func (s *fakeFileService) ReadFileChunk(
	_ context.Context,
	req *agentv1.ReadFileChunkRequest,
) (*agentv1.ReadFileChunkResponse, error) {
	content := []byte("hello from fake agent")
	offset := req.GetOffset()
	if offset > uint64(len(content)) {
		offset = uint64(len(content))
	}
	chunkSize := int(req.GetLimit())
	if chunkSize <= 0 {
		chunkSize = 8
	}
	end := int(offset) + chunkSize
	if end > len(content) {
		end = len(content)
	}
	chunk := append([]byte(nil), content[offset:end]...)
	eof := end >= len(content)

	return &agentv1.ReadFileChunkResponse{
		Error:     okError(),
		Path:      "/tmp/demo.txt",
		Offset:    offset,
		Chunk:     chunk,
		TotalSize: uint64(len(content)),
		Eof:       eof,
	}, nil
}

type fakeServiceManagerService struct {
	agentv1.UnimplementedServiceManagerServiceServer
}

func (s *fakeServiceManagerService) ListServices(
	context.Context,
	*agentv1.ListServicesRequest,
) (*agentv1.ListServicesResponse, error) {
	return &agentv1.ListServicesResponse{
		Error: okError(),
		Services: []*agentv1.ServiceInfo{
			{
				Name:        "nginx.service",
				DisplayName: "Nginx",
				Status:      "active",
			},
		},
	}, nil
}

type fakeDockerService struct {
	agentv1.UnimplementedDockerServiceServer
}

func (s *fakeDockerService) ListContainers(
	context.Context,
	*agentv1.ListDockerContainersRequest,
) (*agentv1.ListDockerContainersResponse, error) {
	return &agentv1.ListDockerContainersResponse{
		Error: okError(),
		Containers: []*agentv1.DockerContainerInfo{
			{
				Id:     "abc123",
				Name:   "snowpanel-core-agent",
				Image:  "snowpanel-core-agent:latest",
				State:  "running",
				Status: "Up 1 minute",
			},
		},
	}, nil
}

type fakeCronService struct {
	agentv1.UnimplementedCronServiceServer
}

func (s *fakeCronService) ListCronTasks(
	context.Context,
	*agentv1.ListCronTasksRequest,
) (*agentv1.ListCronTasksResponse, error) {
	return &agentv1.ListCronTasksResponse{
		Error: okError(),
		Tasks: []*agentv1.CronTask{
			{
				Id:         "task-1",
				Expression: "*/5 * * * *",
				Command:    "backup",
				Enabled:    true,
			},
		},
	}, nil
}
