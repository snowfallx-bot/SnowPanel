package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/api"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/database"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/logger"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

func main() {
	cfg := config.Load()
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
	authService := service.NewAuthService(userRepo, cfg.Auth)
	if err := authService.EnsureDefaultAdmin(context.Background()); err != nil {
		zapLogger.Fatal("failed to ensure default admin", logger.Err(err))
	}

	agentClient := grpcclient.New(cfg.AgentTarget, cfg.AgentTimeout)
	dashboardService := service.NewDashboardService(agentClient)
	fileService := service.NewFileService(agentClient)

	server := &http.Server{
		Addr: cfg.Server.Address(),
		Handler: api.NewRouter(api.RouterDeps{
			Logger:           zapLogger,
			DB:               db,
			AgentClient:      agentClient,
			AuthService:      authService,
			DashboardService: dashboardService,
			FileService:      fileService,
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
