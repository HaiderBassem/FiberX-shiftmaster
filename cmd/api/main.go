package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"shiftmaster-backend/cmd/api/handlers"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/notification"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
	"shiftmaster-backend/pkg/database"
)

func main() {
	// Load configuration (reads .env automatically)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize the Database Connection
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// --- Initialize Repositories ---
	employeeRepo := repository.NewEmployeeRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	shiftRepo := repository.NewShiftRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	leaveRepo := repository.NewLeaveRepository(db)
	swapRepo := repository.NewSwapRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	boardRepo := repository.NewBoardRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	leaveBalanceRepo := repository.NewLeaveBalanceRepository(db)
	leaveTypeRepo := repository.NewLeaveTypeRepository(db)
	infoTableRepo := repository.NewInfoTableRepository(db)
	helpDocRepo := repository.NewHelpDocumentRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db.Pool())
	handoverRepo := repository.NewHandoverRepository(db)
	moduleAccessRepo := repository.NewModuleAccessRepository(db)

	// --- Initialize Services ---
	authService := service.NewAuthService(employeeRepo, cfg.JWT.BcryptCost)
	notifService := service.NewNotificationService(notifRepo)
	emailService := service.NewEmailService(cfg.GraphAPI)
	employeeService := service.NewEmployeeService(employeeRepo, departmentRepo, authService)
	scheduleService := service.NewScheduleService(scheduleRepo, employeeRepo, shiftRepo, notifService, db)
	
	pushService := notification.NewPushService(notifRepo, cfg.VAPID)
	
	leaveService := service.NewLeaveService(leaveRepo, employeeRepo, scheduleRepo, leaveBalanceRepo, leaveTypeRepo, notifService, emailService, pushService)
	swapService := service.NewSwapService(swapRepo, scheduleRepo, employeeRepo, taskRepo, notifService, emailService, db)
	taskService := service.NewTaskService(taskRepo, boardRepo, employeeRepo, scheduleRepo)
	auditService := service.NewAuditService(auditRepo)
	leaveTypeService := service.NewLeaveTypeService(leaveTypeRepo)
	infoTableService := service.NewInfoTableService(infoTableRepo, employeeRepo)
	helpDocService := service.NewHelpDocumentService(helpDocRepo, employeeRepo)
	announcementService := service.NewAnnouncementService(announcementRepo, employeeRepo, emailService, pushService)
	moduleAccessService := service.NewModuleAccessService(moduleAccessRepo, employeeRepo)

	// --- Initialize Handlers ---
	authHandler := handlers.NewAuthHandler(authService, employeeService, cfg.JWT.Secret, cfg.JWT.AccessExpireMin, cfg.JWT.RefreshExpireDays)
	empHandler := handlers.NewEmployeeHandler(employeeService, leaveBalanceRepo, taskRepo, leaveRepo, departmentRepo)
	deptHandler := handlers.NewDepartmentHandler(departmentRepo, employeeRepo)
	shiftHandler := handlers.NewShiftHandler(shiftRepo)
	scheduleHandler := handlers.NewScheduleHandler(scheduleService)
	leaveHandler := handlers.NewLeaveHandler(leaveService)
	swapHandler := handlers.NewSwapHandler(swapService)
	taskHandler := handlers.NewTaskHandler(taskService)
	notifHandler := handlers.NewNotificationHandler(notifService)
	auditHandler := handlers.NewAuditHandler(auditService)
	leaveTypeHandler := handlers.NewLeaveTypeHandler(leaveTypeService)
	infoTableHandler := handlers.NewInfoTableHandler(infoTableService)
	helpDocHandler := handlers.NewHelpDocumentHandler(helpDocService)
	announcementHandler := handlers.NewAnnouncementHandler(announcementRepo, announcementService)
	pushHandler := handlers.NewPushHandler(notifRepo, cfg.VAPID)
	handoverHandler := handlers.NewHandoverHandler(handoverRepo, employeeRepo, shiftRepo, scheduleRepo, notifService)
	uploadHandler := handlers.NewUploadHandler()
	moduleAccessHandler := handlers.NewModuleAccessHandler(moduleAccessService)

	// --- Setup Gin Engine ---
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           time.Duration(cfg.CORS.MaxAge) * time.Second,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		if err := db.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Setup API routes
	SetupRouter(r, cfg.JWT.Secret, departmentRepo,
		authHandler, empHandler, deptHandler, shiftHandler,
		scheduleHandler, leaveHandler, swapHandler, taskHandler, notifHandler, auditHandler, leaveTypeHandler, infoTableHandler, helpDocHandler, announcementHandler, pushHandler, handoverHandler, uploadHandler, moduleAccessHandler,
	)

	// --- Start HTTP Server ---
	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("ShiftMaster API running in %s mode on %s", cfg.Server.Env, cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Start pool monitor in background
	go db.Monitor(context.Background(), time.Minute)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
