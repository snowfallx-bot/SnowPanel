package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/handler"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RouterDeps struct {
	Logger           *zap.Logger
	DB               *gorm.DB
	AgentClient      grpcclient.AgentClient
	AuthService      service.AuthService
	DashboardService service.DashboardService
}

func NewRouter(deps RouterDeps) *gin.Engine {
	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Recover(deps.Logger),
		middleware.AccessLog(deps.Logger),
	)

	healthHandler := handler.NewHealthHandler(deps.DB)
	systemHandler := handler.NewSystemHandler(time.Now)
	authHandler := handler.NewAuthHandler(deps.AuthService)
	dashboardHandler := handler.NewDashboardHandler(deps.DashboardService)

	router.GET("/health", healthHandler.Health)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", systemHandler.Ping)
		v1.POST("/auth/login", authHandler.Login)

		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(deps.AuthService))
		{
			protected.GET("/auth/me", authHandler.Me)
			protected.GET("/dashboard/summary", dashboardHandler.Summary)
		}
	}

	return router
}
