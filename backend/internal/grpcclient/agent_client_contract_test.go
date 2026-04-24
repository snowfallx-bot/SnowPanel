package grpcclient

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	agentv1 "github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient/pb/proto/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

	for _, serviceName := range []string{"HealthService", "SystemService", "FileService"} {
		if files.Services().ByName(protoreflect.Name(serviceName)) == nil {
			t.Fatalf("expected service descriptor %s to exist", serviceName)
		}
	}
}

type protoContractOptions struct {
	healthTransportCode codes.Code
	listFilesError      *agentv1.Error
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

func (s *protoContractHealthService) Check(context.Context, *agentv1.HealthCheckRequest) (*agentv1.HealthCheckResponse, error) {
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
