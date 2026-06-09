package service

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	agentv1 "github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient/pb/proto/agent/v1"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/hostctx"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"google.golang.org/grpc"
)

func TestHostAwareAgentClientRoutesToSelectedHost(t *testing.T) {
	defaultTarget := startSystemOverviewServer(t, "default-node")
	selectedTarget := startSystemOverviewServer(t, "selected-node")

	selectedHostAddr, selectedHostPort := splitHostPort(t, selectedTarget)
	repo := &stubHostRepository{
		hosts: map[int64]*model.Host{
			7: {
				ID:      7,
				Name:    "selected-node",
				Address: selectedHostAddr,
				Port:    selectedHostPort,
				Status:  HostStatusOnline,
			},
		},
	}

	client := NewHostAwareAgentClient(defaultTarget, 2*time.Second, repo)

	defaultOverview, err := client.GetSystemOverview(context.Background())
	if err != nil {
		t.Fatalf("GetSystemOverview() default error = %v", err)
	}
	if defaultOverview.Hostname != "default-node" {
		t.Fatalf("expected default-node, got %s", defaultOverview.Hostname)
	}

	selectedCtx := hostctx.WithHostID(context.Background(), 7)
	selectedOverview, err := client.GetSystemOverview(selectedCtx)
	if err != nil {
		t.Fatalf("GetSystemOverview() selected error = %v", err)
	}
	if selectedOverview.Hostname != "selected-node" {
		t.Fatalf("expected selected-node, got %s", selectedOverview.Hostname)
	}
}

type stubHostRepository struct {
	repository.HostRepository
	hosts map[int64]*model.Host
}

func (r *stubHostRepository) Create(context.Context, *model.Host) error  { return nil }
func (r *stubHostRepository) List(context.Context) ([]model.Host, error) { return nil, nil }
func (r *stubHostRepository) UpdateHealth(context.Context, int64, int16, string, *time.Time) error {
	return nil
}

func (r *stubHostRepository) GetByID(_ context.Context, id int64) (*model.Host, error) {
	if host, ok := r.hosts[id]; ok {
		return host, nil
	}
	return nil, nil
}

type testSystemService struct {
	agentv1.UnimplementedSystemServiceServer
	hostname string
}

func (s *testSystemService) GetSystemOverview(context.Context, *agentv1.GetSystemOverviewRequest) (*agentv1.GetSystemOverviewResponse, error) {
	return &agentv1.GetSystemOverviewResponse{
		Error: okError(),
		Overview: &agentv1.SystemOverview{
			Hostname: s.hostname,
			Os:       "Linux",
			Kernel:   "6.8.0",
			Uptime:   "1h",
			Cpu: &agentv1.CPUInfo{
				Model:        "test-cpu",
				LogicalCores: 8,
				UsagePercent: 10,
			},
			Memory: &agentv1.MemoryInfo{
				TotalBytes:   1024,
				UsedBytes:    256,
				UsagePercent: 25,
			},
			Disks: []*agentv1.DiskInfo{{
				MountPoint:   "/",
				TotalBytes:   1024,
				UsedBytes:    512,
				UsagePercent: 50,
			}},
		},
	}, nil
}

func (s *testSystemService) GetRealtimeResource(context.Context, *agentv1.GetRealtimeResourceRequest) (*agentv1.GetRealtimeResourceResponse, error) {
	return &agentv1.GetRealtimeResourceResponse{
		Error: okError(),
		Resource: &agentv1.RealtimeResource{
			CpuUsagePercent:    10,
			MemoryUsagePercent: 25,
			DiskUsagePercent:   50,
		},
	}, nil
}

func startSystemOverviewServer(t *testing.T, hostname string) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	server := grpc.NewServer()
	agentv1.RegisterSystemServiceServer(server, &testSystemService{hostname: hostname})

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func splitHostPort(t *testing.T, target string) (string, int) {
	t.Helper()

	host, portRaw, err := net.SplitHostPort(target)
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	var port int
	if _, err := fmt.Sscanf(portRaw, "%d", &port); err != nil {
		t.Fatalf("invalid port %q: %v", portRaw, err)
	}
	return host, port
}
