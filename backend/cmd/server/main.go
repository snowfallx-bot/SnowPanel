package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/database"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/logger"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

func main() {
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

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		zapLogger.Fatal("failed to connect postgres", logger.Err(err))
	}

	userRepo := repository.NewUserRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, cfg.Auth)
	if err := authService.EnsureDefaultAdmin(context.Background()); err != nil {
		zapLogger.Fatal("failed to ensure default admin", logger.Err(err))
	}

	agentClient := grpcclient.New(cfg.AgentTarget, cfg.AgentTimeout)
	dashboardService := service.NewDashboardService(agentClient)
	fileService := service.NewFileService(agentClient)
	serviceManager := service.NewServiceManagerService(agentClient)
	dockerService := service.NewDockerService(agentClient)
	cronService := service.NewCronService(agentClient)
	taskService := service.NewTaskService(taskRepo, dockerService, serviceManager)
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
			AgentClient:      agentClient,
			AuthService:      authService,
			DashboardService: dashboardService,
			FileService:      fileService,
			ServiceManager:   serviceManager,
			DockerService:    dockerService,
			CronService:      cronService,
			AuditService:     auditService,
			TaskService:      taskService,
			LoginAttempts:    loginAttempts,
		}),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.Printf("backend listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("backend stopped unexpectedly: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = database.Close(shutdownCtx, db)
}
