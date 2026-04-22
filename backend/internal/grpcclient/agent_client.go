package grpcclient

import (
	"context"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("grpc client is not initialized yet; generate proto code and wire real grpc transport")

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

type AgentClient interface {
	GetSystemOverview(ctx context.Context) (SystemOverview, error)
	GetRealtimeResource(ctx context.Context) (RealtimeResource, error)
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

func (c *Client) GetSystemOverview(ctx context.Context) (SystemOverview, error) {
	_ = ctx
	return SystemOverview{}, ErrNotImplemented
}

func (c *Client) GetRealtimeResource(ctx context.Context) (RealtimeResource, error) {
	_ = ctx
	return RealtimeResource{}, ErrNotImplemented
}
