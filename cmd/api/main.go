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
	"shiftmaster-backend/internal/assistant"
	"shiftmaster-backend/internal/assistant/llm"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/middleware"
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
	fiberxDataRepo := repository.NewFiberxDataRepository(db)
	securityRepo := repository.NewSecurityRepository(db)
	itemReqRepo := repository.NewItemRequestRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	provinceRepo := repository.NewProvinceRepository(db)
	assistantRepo := repository.NewAssistantRepository(db)

	// --- Initialize Services ---
	// IP-level blocking and per-account lockout both come from configuration rather
	// than hardcoded constants, so an operator can tune them without a rebuild.
	securityService := service.NewSecurityService(securityRepo, cfg.Security.MaxLoginAttempts*2, cfg.Security.LockoutDuration())
	authService := service.NewAuthService(
		employeeRepo,
		securityService,
		cfg.JWT.BcryptCost,
		cfg.Security.MaxLoginAttempts,
		cfg.Security.LockoutDuration(),
	)
	notifService := service.NewNotificationService(notifRepo)
	emailService := service.NewEmailService(cfg.GraphAPI)
	employeeService := service.NewEmployeeService(employeeRepo, departmentRepo, authService)
	scheduleService := service.NewScheduleService(scheduleRepo, employeeRepo, shiftRepo, leaveRepo, notifService, emailService, db)

	pushService := notification.NewPushService(notifRepo, employeeRepo, cfg.VAPID)

	leaveService := service.NewLeaveService(leaveRepo, employeeRepo, departmentRepo, scheduleRepo, shiftRepo, leaveBalanceRepo, leaveTypeRepo, notifService, emailService, pushService)
	swapService := service.NewSwapService(swapRepo, scheduleRepo, employeeRepo, taskRepo, notifService, emailService, db)
	taskService := service.NewTaskService(taskRepo, boardRepo, employeeRepo, scheduleRepo)
	auditService := service.NewAuditService(auditRepo)
	leaveTypeService := service.NewLeaveTypeService(leaveTypeRepo)
	infoTableService := service.NewInfoTableService(infoTableRepo, employeeRepo)
	helpDocService := service.NewHelpDocumentService(helpDocRepo, employeeRepo)
	announcementService := service.NewAnnouncementService(announcementRepo, employeeRepo, emailService, pushService)
	moduleAccessService := service.NewModuleAccessService(moduleAccessRepo, employeeRepo)
	fiberxDataService := service.NewFiberxDataService(fiberxDataRepo, employeeRepo)
	itemReqService := service.NewItemRequestService(itemReqRepo, employeeRepo, departmentRepo, emailService)
	provinceService := service.NewProvinceService(provinceRepo)

	// The AI assistant runs entirely over the services above; without an API
	// key it stays dormant and its endpoints report themselves unavailable.
	assistantService := assistant.NewService(&assistant.Deps{
		Cfg: cfg.Assistant,
		LLM: llm.NewAnthropic(llm.AnthropicConfig{
			APIKey:     cfg.Assistant.APIKey,
			Model:      cfg.Assistant.Model,
			BaseURL:    cfg.Assistant.BaseURL,
			MaxTokens:  cfg.Assistant.MaxTokens,
			Timeout:    cfg.Assistant.Timeout,
			MaxRetries: cfg.Assistant.MaxRetries,
		}),
		AssistantRepo:    assistantRepo,
		EmployeeRepo:     employeeRepo,
		DepartmentRepo:   departmentRepo,
		ShiftRepo:        shiftRepo,
		ScheduleRepo:     scheduleRepo,
		LeaveRepo:        leaveRepo,
		LeaveTypeRepo:    leaveTypeRepo,
		TaskRepo:         taskRepo,
		NotifRepo:        notifRepo,
		AnnouncementRepo: announcementRepo,
		HandoverRepo:     handoverRepo,
		TicketRepo:       ticketRepo,
		ItemReqRepo:      itemReqRepo,
		HelpDocRepo:      helpDocRepo,
		FiberxRepo:       fiberxDataRepo,
		InfoTableRepo:    infoTableRepo,
		ServiceRepo:      serviceRepo,
		ProvinceRepo:     provinceRepo,
		AuditLogRepo:     auditRepo,
		AuthService:      authService,
		LeaveService:     leaveService,
		ScheduleService:  scheduleService,
		TaskService:      taskService,
		SwapService:      swapService,
		InfoTableService: infoTableService,
		HelpDocService:   helpDocService,
		FiberxService:    fiberxDataService,
		ItemReqService:   itemReqService,
		ProvinceService:  provinceService,
		AuditService:     auditService,
	})

	// --- Initialize Handlers ---
	// Cookie security must follow the scheme users actually connect with, not
	// the environment name: a Secure cookie over plain HTTP is silently
	// discarded by the browser, which took every protected image down with it.
	secureCookies := cfg.CookieSecure()
	if cfg.Server.IsProduction() && !secureCookies {
		log.Printf("WARN: cookies are issued without the Secure attribute because the deployment serves plain HTTP; put the site behind HTTPS when possible")
	}
	authHandler := handlers.NewAuthHandler(authService, employeeService, cfg.JWT, secureCookies)
	empHandler := handlers.NewEmployeeHandler(employeeService, leaveBalanceRepo, taskRepo, leaveRepo, departmentRepo, cfg.Upload)
	deptHandler := handlers.NewDepartmentHandler(departmentRepo, employeeRepo)
	shiftHandler := handlers.NewShiftHandler(shiftRepo)
	scheduleHandler := handlers.NewScheduleHandler(scheduleService)
	leaveHandler := handlers.NewLeaveHandler(leaveService)
	swapHandler := handlers.NewSwapHandler(swapService)
	taskHandler := handlers.NewTaskHandler(taskService)
	notifHandler := handlers.NewNotificationHandler(notifService, cfg.JWT)
	auditHandler := handlers.NewAuditHandler(auditService)
	leaveTypeHandler := handlers.NewLeaveTypeHandler(leaveTypeService)
	infoTableHandler := handlers.NewInfoTableHandler(infoTableService)
	helpDocHandler := handlers.NewHelpDocumentHandler(helpDocService)
	announcementHandler := handlers.NewAnnouncementHandler(announcementRepo, announcementService)
	pushHandler := handlers.NewPushHandler(notifRepo, cfg.VAPID)
	handoverHandler := handlers.NewHandoverHandler(handoverRepo, employeeRepo, shiftRepo, scheduleRepo, notifService)
	uploadHandler := handlers.NewUploadHandler(cfg.Upload)
	moduleAccessHandler := handlers.NewModuleAccessHandler(moduleAccessService)
	fiberxDataHandler := handlers.NewFiberxDataHandler(fiberxDataService)
	securityHandler := handlers.NewSecurityHandler(securityService)
	itemReqHandler := handlers.NewItemRequestHandler(itemReqService)
	ticketHandler := handlers.NewTicketHandler(ticketRepo)
	serviceHandler := handlers.NewServiceHandler(serviceRepo, db)
	provinceHandler := handlers.NewProvinceHandler(provinceService)
	assistantHandler := handlers.NewAssistantHandler(assistantService)

	// --- Setup Gin Engine ---
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Without this, gin trusts every upstream and derives ClientIP from any
	// X-Forwarded-For the caller supplies. Both the IP blocker and the failed-login
	// throttle key off ClientIP, so a spoofable value would let an attacker evade
	// blocking by rotating the header, and let them get someone else's address
	// blocked by forging it.
	if err := r.SetTrustedProxies(cfg.Security.TrustedProxies); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}

	// The WebSocket upgrader refuses every connection until this is set.
	notification.ConfigureOrigins(cfg.CORS.AllowedOrigins)

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           time.Duration(cfg.CORS.MaxAge) * time.Second,
	}))

	// Security Middleware (IP Blocker)
	r.Use(middleware.IPBlockerMiddleware(securityService))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		if err := db.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Setup API routes
	SetupRouter(r, cfg.JWT.Secret, cfg.JWT.Issuer, departmentRepo,
		authHandler, empHandler, deptHandler, shiftHandler,
		scheduleHandler, leaveHandler, swapHandler, taskHandler, notifHandler, auditHandler, leaveTypeHandler, infoTableHandler, helpDocHandler, announcementHandler, pushHandler, handoverHandler, uploadHandler, moduleAccessHandler, fiberxDataHandler, securityHandler, itemReqHandler, ticketHandler, serviceHandler, provinceHandler, assistantHandler,
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

	// Start upcoming leave reminders background worker (runs every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := leaveService.SendUpcomingLeaveReminders(ctx); err != nil {
				log.Printf("Failed to send upcoming leave reminders: %v", err)
			}
			cancel()
		}
	}()

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
