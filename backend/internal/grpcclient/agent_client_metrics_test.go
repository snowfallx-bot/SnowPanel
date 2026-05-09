package grpcclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	agentv1 "github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient/pb/proto/agent/v1"
	appmetrics "github.com/snowfallx-bot/SnowPanel/backend/internal/metrics"
	"google.golang.org/grpc/codes"
)

func TestClientObserveAgentMetricsOnSuccess(t *testing.T) {
	registry := prometheus.NewRegistry()
	originalMetrics := agentMetrics
	agentMetrics = appmetrics.New(registry)
	defer func() { agentMetrics = originalMetrics }()

	target := startProtoContractServer(t, protoContractOptions{})
	client := New(target, 2*time.Second)

	_, err := client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}

	successCount := testutil.ToFloat64(agentMetrics.AgentRequestsTotal.WithLabelValues("health.check", "success", "false"))
	if successCount != 1 {
		t.Fatalf("expected success counter 1, got %f", successCount)
	}
}

func TestClientObserveAgentMetricsOnStructuredError(t *testing.T) {
	registry := prometheus.NewRegistry()
	originalMetrics := agentMetrics
	agentMetrics = appmetrics.New(registry)
	defer func() { agentMetrics = originalMetrics }()

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

	errorCount := testutil.ToFloat64(agentMetrics.AgentRequestsTotal.WithLabelValues("file.list", "error", "false"))
	if errorCount != 1 {
		t.Fatalf("expected structured error counter 1, got %f", errorCount)
	}
}

func TestClientObserveAgentMetricsOnTransportError(t *testing.T) {
	registry := prometheus.NewRegistry()
	originalMetrics := agentMetrics
	agentMetrics = appmetrics.New(registry)
	defer func() { agentMetrics = originalMetrics }()

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

	errorCount := testutil.ToFloat64(agentMetrics.AgentRequestsTotal.WithLabelValues("health.check", "error", "true"))
	if errorCount != 1 {
		t.Fatalf("expected transport error counter 1, got %f", errorCount)
	}
}
