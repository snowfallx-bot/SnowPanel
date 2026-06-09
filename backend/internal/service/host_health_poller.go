package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type HostHealthPoller struct {
	hostService HostService
	interval    time.Duration
	logger      *zap.Logger
}

func NewHostHealthPoller(hostService HostService, interval time.Duration, logger *zap.Logger) *HostHealthPoller {
	if logger == nil {
		logger = zap.NewNop()
	}
	if interval <= 0 {
		interval = time.Minute
	}
	return &HostHealthPoller{
		hostService: hostService,
		interval:    interval,
		logger:      logger,
	}
}

func (p *HostHealthPoller) Start(ctx context.Context) {
	if p == nil || p.hostService == nil {
		return
	}
	go p.run(ctx)
}

func (p *HostHealthPoller) run(ctx context.Context) {
	p.checkOnce(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("host health poller stopped")
			return
		case <-ticker.C:
			p.checkOnce(ctx)
		}
	}
}

func (p *HostHealthPoller) checkOnce(ctx context.Context) {
	result, err := p.hostService.ListHosts(ctx)
	if err != nil {
		p.logger.Warn("failed to list hosts for health polling", zap.Error(err))
		return
	}

	for _, host := range result.Items {
		if host.Status == "disabled" {
			continue
		}
		if _, err := p.hostService.CheckHost(ctx, host.ID); err != nil {
			p.logger.Warn(
				"host health poll failed",
				zap.Int64("host_id", host.ID),
				zap.String("host", host.Name),
				zap.Error(err),
			)
			continue
		}
		p.logger.Debug(
			"host health poll succeeded",
			zap.Int64("host_id", host.ID),
			zap.String("host", host.Name),
		)
	}
}
