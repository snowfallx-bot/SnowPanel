package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/handler"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
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
	FileService      service.FileService
	ServiceManager   service.ServiceManagerService
	DockerService    service.DockerService
	CronService      service.CronService
	AuditService     service.AuditService
	TaskService      service.TaskService
	LoginAttempts    security.LoginAttemptGuard
}

func NewRouter(deps RouterDeps) *gin.Engine {
	router := gin.New()
	router.Use(
		middleware.CORS(),
		middleware.RequestID(),
		middleware.Recover(deps.Logger),
		middleware.AccessLog(deps.Logger),
	)

	healthHandler := handler.NewHealthHandler(deps.DB, deps.AgentClient)
	systemHandler := handler.NewSystemHandler(time.Now)
	authHandler := handler.NewAuthHandler(deps.AuthService, deps.AuditService, deps.LoginAttempts)
	dashboardHandler := handler.NewDashboardHandler(deps.DashboardService)
	fileHandler := handler.NewFileHandler(deps.FileService, deps.AuditService)
	serviceHandler := handler.NewServiceHandler(deps.ServiceManager, deps.AuditService)
	dockerHandler := handler.NewDockerHandler(deps.DockerService, deps.AuditService)
	cronHandler := handler.NewCronHandler(deps.CronService, deps.AuditService)
	auditHandler := handler.NewAuditHandler(deps.AuditService)
	taskHandler := handler.NewTaskHandler(deps.TaskService, deps.AuditService)

	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Readiness)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", systemHandler.Ping)
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/refresh", authHandler.Refresh)

		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(deps.AuthService))
		{
			protected.GET("/auth/me", authHandler.Me)
			protected.POST("/auth/logout", authHandler.Logout)
			protected.POST("/auth/change-password", authHandler.ChangePassword)
			protected.GET("/dashboard/summary", dashboardHandler.Summary)
			files := protected.Group("/files")
			{
				files.GET("/list", middleware.RequirePermission("files.read"), fileHandler.ListFiles)
				files.GET("/download", middleware.RequirePermission("files.read"), fileHandler.DownloadFile)
				files.POST("/upload", middleware.RequirePermission("files.write"), fileHandler.UploadFile)
				files.POST("/read", middleware.RequirePermission("files.read"), fileHandler.ReadTextFile)
				files.POST("/write", middleware.RequirePermission("files.write"), fileHandler.WriteTextFile)
				files.POST("/rename", middleware.RequirePermission("files.write"), fileHandler.RenameFile)
				files.POST("/mkdir", middleware.RequirePermission("files.write"), fileHandler.CreateDirectory)
				files.DELETE("/delete", middleware.RequirePermission("files.write"), fileHandler.DeleteFile)
			}

			services := protected.Group("/services")
			{
				services.GET("", middleware.RequirePermission("services.read"), serviceHandler.ListServices)
				services.POST("/:name/start", middleware.RequirePermission("services.manage"), serviceHandler.StartService)
				services.POST("/:name/stop", middleware.RequirePermission("services.manage"), serviceHandler.StopService)
				services.POST("/:name/restart", middleware.RequirePermission("services.manage"), serviceHandler.RestartService)
			}

			docker := protected.Group("/docker")
			{
				docker.GET("/containers", middleware.RequirePermission("docker.read"), dockerHandler.ListContainers)
				docker.POST("/containers/:id/start", middleware.RequirePermission("docker.manage"), dockerHandler.StartContainer)
				docker.POST("/containers/:id/stop", middleware.RequirePermission("docker.manage"), dockerHandler.StopContainer)
				docker.POST("/containers/:id/restart", middleware.RequirePermission("docker.manage"), dockerHandler.RestartContainer)
				docker.GET("/images", middleware.RequirePermission("docker.read"), dockerHandler.ListImages)
			}

			cron := protected.Group("/cron")
			{
				cron.GET("", middleware.RequirePermission("cron.read"), cronHandler.ListTasks)
				cron.POST("", middleware.RequirePermission("cron.manage"), cronHandler.CreateTask)
				cron.PUT("/:id", middleware.RequirePermission("cron.manage"), cronHandler.UpdateTask)
				cron.DELETE("/:id", middleware.RequirePermission("cron.manage"), cronHandler.DeleteTask)
				cron.POST("/:id/enable", middleware.RequirePermission("cron.manage"), cronHandler.EnableTask)
				cron.POST("/:id/disable", middleware.RequirePermission("cron.manage"), cronHandler.DisableTask)
			}

			audit := protected.Group("/audit")
			{
				audit.GET("/logs", middleware.RequirePermission("audit.read"), auditHandler.ListLogs)
			}

			tasks := protected.Group("/tasks")
			{
				tasks.GET("", middleware.RequirePermission("tasks.read"), taskHandler.ListTasks)
				tasks.GET("/:id", middleware.RequirePermission("tasks.read"), taskHandler.GetTaskDetail)
				tasks.POST("/docker/restart", middleware.RequirePermission("tasks.manage"), taskHandler.CreateDockerRestartTask)
				tasks.POST("/services/restart", middleware.RequirePermission("tasks.manage"), taskHandler.CreateServiceRestartTask)
				tasks.POST("/:id/cancel", middleware.RequirePermission("tasks.manage"), taskHandler.CancelTask)
				tasks.POST("/:id/retry", middleware.RequirePermission("tasks.manage"), taskHandler.RetryTask)
			}
		}
	}

	return router
}
