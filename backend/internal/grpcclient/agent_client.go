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

type AgentClient interface {
	GetSystemOverview(ctx context.Context) (SystemOverview, error)
	GetRealtimeResource(ctx context.Context) (RealtimeResource, error)
	ListFiles(ctx context.Context, req ListFilesRequest) (ListFilesResult, error)
	ReadTextFile(ctx context.Context, req ReadTextFileRequest) (ReadTextFileResult, error)
	WriteTextFile(ctx context.Context, req WriteTextFileRequest) (WriteTextFileResult, error)
	CreateDirectory(ctx context.Context, req CreateDirectoryRequest) (CreateDirectoryResult, error)
	DeleteFile(ctx context.Context, req DeleteFileRequest) (DeleteFileResult, error)
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

func (c *Client) ListFiles(ctx context.Context, req ListFilesRequest) (ListFilesResult, error) {
	_ = ctx
	_ = req
	return ListFilesResult{}, ErrNotImplemented
}

func (c *Client) ReadTextFile(ctx context.Context, req ReadTextFileRequest) (ReadTextFileResult, error) {
	_ = ctx
	_ = req
	return ReadTextFileResult{}, ErrNotImplemented
}

func (c *Client) WriteTextFile(ctx context.Context, req WriteTextFileRequest) (WriteTextFileResult, error) {
	_ = ctx
	_ = req
	return WriteTextFileResult{}, ErrNotImplemented
}

func (c *Client) CreateDirectory(ctx context.Context, req CreateDirectoryRequest) (CreateDirectoryResult, error) {
	_ = ctx
	_ = req
	return CreateDirectoryResult{}, ErrNotImplemented
}

func (c *Client) DeleteFile(ctx context.Context, req DeleteFileRequest) (DeleteFileResult, error) {
	_ = ctx
	_ = req
	return DeleteFileResult{}, ErrNotImplemented
}
