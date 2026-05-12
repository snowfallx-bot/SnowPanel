package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/database"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/logger"
	appmetrics "github.com/snowfallx-bot/SnowPanel/backend/internal/metrics"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/observability"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

func main() {
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid runtime config: %v", err)
	}

	zapLogger, err := logger.New(cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		_ = zapLogger.Sync()
	}()

	tracingEnabled := cfg.Tracing.Enabled
	tracingShutdown, err := observability.InitTracing(rootCtx, cfg.Tracing, cfg.AppEnv)
	if err != nil {
		tracingEnabled = false
		zapLogger.Warn("otel tracing disabled", logger.Err(err))
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := tracingShutdown(shutdownCtx); shutdownErr != nil {
				zapLogger.Warn("failed to shutdown otel tracing", logger.Err(shutdownErr))
			}
		}()
	}

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		zapLogger.Fatal("failed to connect postgres", logger.Err(err))
	}

	userRepo := repository.NewUserRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	backupRepo := repository.NewBackupRepository(db)
	systemSettingRepo := repository.NewSystemSettingRepository(db)
	settingsEncryptor, err := newSettingsEncryptor(cfg.Security)
	if err != nil {
		zapLogger.Fatal("invalid settings encryption config", logger.Err(err))
	}
	if err := validateEncryptedSettingsStartup(rootCtx, systemSettingRepo, settingsEncryptor); err != nil {
		zapLogger.Fatal("invalid encrypted settings config", logger.Err(err))
	}
	auditService := service.NewAuditServiceWithOptions(auditRepo, service.AuditServiceOptions{
		RetentionDays: cfg.Audit.RetentionDays,
		ExportMaxRows: cfg.Audit.ExportMaxRows,
	})
	authService := service.NewAuthService(userRepo, cfg.Auth)
	if err := authService.EnsureDefaultAdmin(rootCtx); err != nil {
		zapLogger.Fatal("failed to ensure default admin", logger.Err(err))
	}

	agentClient := grpcclient.NewWithAuth(cfg.AgentTarget, cfg.AgentTimeout, grpcclient.AuthConfig{
		Mode:        cfg.AgentAuth.Mode,
		SharedToken: cfg.AgentAuth.SharedToken,
	})
	dashboardService := service.NewDashboardService(agentClient)
	fileService := service.NewFileService(agentClient)
	serviceManager := service.NewServiceManagerService(agentClient)
	dockerService := service.NewDockerService(agentClient)
	cronService := service.NewCronService(agentClient)
	backupService := service.NewBackupService(backupRepo)
	metricsSet := appmetrics.Default()
	taskService := service.NewTaskServiceWithOptions(
		taskRepo,
		dockerService,
		serviceManager,
		service.TaskServiceOptions{
			AsyncExecution: !cfg.TaskWorker.Enabled,
			MaxAttempts:    cfg.TaskWorker.MaxAttempts,
			Metrics:        metricsSet,
			BackupService:  backupService,
		},
	)
	if cfg.TaskWorker.Enabled {
		go taskService.RunWorker(rootCtx, service.TaskWorkerOptions{
			WorkerID:      cfg.TaskWorker.WorkerID,
			Concurrency:   cfg.TaskWorker.Concurrency,
			LeaseDuration: cfg.TaskWorker.LeaseDuration,
			PollInterval:  cfg.TaskWorker.PollInterval,
		})
		zapLogger.Info("durable task worker enabled")
	}
	var loginAttempts security.LoginAttemptGuard = security.NewLoginAttemptLimiter(security.LoginAttemptLimiterOptions{
		MaxFailures:   cfg.Auth.LoginMaxFailures,
		FailureWindow: cfg.Auth.LoginFailureWindow,
		LockDuration:  cfg.Auth.LoginLockDuration,
	})
	var redisClient *redis.Client
	if cfg.Auth.LoginAttemptStore == "redis" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Address(),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = redisClient.Ping(pingCtx).Err()
		cancel()
		if err != nil {
			zapLogger.Warn(
				"redis login limiter unavailable, fallback to in-memory limiter",
				logger.Err(err),
			)
			_ = redisClient.Close()
			redisClient = nil
		} else {
			zapLogger.Info("redis-backed login limiter enabled")
			loginAttempts = security.NewRedisLoginAttemptLimiter(
				redisClient,
				security.RedisLoginAttemptLimiterOptions{
					MaxFailures:   cfg.Auth.LoginMaxFailures,
					FailureWindow: cfg.Auth.LoginFailureWindow,
					LockDuration:  cfg.Auth.LoginLockDuration,
					KeyPrefix:     cfg.Auth.LoginAttemptPrefix,
				},
			)
		}
	}
	defer func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
	}()

	server := &http.Server{
		Addr: cfg.Server.Address(),
		Handler: api.NewRouter(api.RouterDeps{
			Logger:           zapLogger,
			DB:               db,
			TracingEnabled:   tracingEnabled,
			TracingSvcName:   cfg.Tracing.ServiceName,
			AgentClient:      agentClient,
			AuthService:      authService,
			DashboardService: dashboardService,
			FileService:      fileService,
			ServiceManager:   serviceManager,
			DockerService:    dockerService,
			CronService:      cronService,
			AuditService:     auditService,
			TaskService:      taskService,
			BackupService:    backupService,
			LoginAttempts:    loginAttempts,
		}),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.Printf("backend listening on %s", server.Addr)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("backend stopped unexpectedly: %v", err)
		}
	case <-rootCtx.Done():
		zapLogger.Info("backend shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		zapLogger.Warn("failed to shutdown http server", logger.Err(err))
	}
	_ = database.Close(shutdownCtx, db)
}

func newSettingsEncryptor(cfg config.SecurityConfig) (*security.Encryptor, error) {
	if cfg.EncryptionKey == "" {
		return nil, nil
	}
	return security.NewEncryptor(cfg.EncryptionKey, cfg.EncryptionKeyID)
}

func validateEncryptedSettingsStartup(
	ctx context.Context,
	repo repository.SystemSettingRepository,
	encryptor *security.Encryptor,
) error {
	hasEncryptedSettings, err := repo.HasEncryptedSettings(ctx)
	if err != nil {
		return fmt.Errorf("check encrypted settings: %w", err)
	}
	if hasEncryptedSettings && encryptor == nil {
		return errors.New("SNOWPANEL_ENCRYPTION_KEY cannot be empty when encrypted settings exist")
	}
	return nil
}
