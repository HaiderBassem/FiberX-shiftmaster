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
	// The push service must exist before the notification service: every
	// persisted notification is delivered in real time through it.
	pushService := notification.NewPushService(notifRepo, employeeRepo, cfg.VAPID)
	notifService := service.NewNotificationService(notifRepo, pushService)
	emailService := service.NewEmailService(cfg.GraphAPI)
	employeeService := service.NewEmployeeService(employeeRepo, departmentRepo, authService)
	scheduleService := service.NewScheduleService(scheduleRepo, employeeRepo, shiftRepo, leaveRepo, notifService, emailService, db)

	leaveService := service.NewLeaveService(leaveRepo, employeeRepo, departmentRepo, scheduleRepo, shiftRepo, leaveBalanceRepo, leaveTypeRepo, notifService, emailService)
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

	// The AI assistant runs entirely over the services above, driven by a
	// language model hosted on this machine. Bringing the runtime up is
	// asynchronous and never blocks startup: if the model fails to load, every
	// other part of ShiftMaster is unaffected and the assistant reports an
	// honest state instead of disappearing.
	var assistantRuntime *llm.Runtime
	var assistantLLM llm.Client
	if cfg.Assistant.Enabled() {
		assistantRuntime = llm.NewRuntime(llm.RuntimeConfig{
			BaseURL:        cfg.Assistant.BaseURL,
			APIKey:         cfg.Assistant.RuntimeKey,
			Managed:        cfg.Assistant.Managed,
			ServerBin:      cfg.Assistant.ServerBin,
			ModelPath:      cfg.Assistant.ModelPath,
			ModelName:      cfg.Assistant.Model,
			ContextSize:    cfg.Assistant.ContextSize,
			GPULayers:      cfg.Assistant.GPULayers,
			Threads:        cfg.Assistant.Threads,
			Parallel:       cfg.Assistant.Parallel,
			LogPath:        cfg.Assistant.RuntimeLogPath,
			StartupTimeout: cfg.Assistant.StartupTimeout,
			QueueWait:      cfg.Assistant.QueueWait,
			Warm:           cfg.Assistant.Warm,
		})
		assistantRuntime.Start(context.Background())
		assistantLLM = llm.NewLocal(llm.LocalConfig{
			BaseURL:        cfg.Assistant.BaseURL,
			Model:          cfg.Assistant.Model,
			APIKey:         cfg.Assistant.RuntimeKey,
			Temperature:    cfg.Assistant.Temperature,
			TopP:           cfg.Assistant.TopP,
			MaxTokens:      cfg.Assistant.MaxTokens,
			Timeout:        cfg.Assistant.Timeout,
			MaxRetries:     cfg.Assistant.MaxRetries,
			GroundingRetry: cfg.Assistant.GroundingRetry,
		}, assistantRuntime)
	}

	assistantService := assistant.NewService(&assistant.Deps{
		Cfg:              cfg.Assistant,
		LLM:              assistantLLM,
		Runtime:          assistantRuntime,
		DB:               db,
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

	// Once the model is up, prime its prompt cache with the real tool
	// catalogue so the first person to ask something does not wait for it.
	// Detached: this must never delay or fail startup.
	if assistantRuntime != nil {
		go func() {
			warmCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			if assistantRuntime.WaitReady(warmCtx) {
				assistantService.WarmCatalogue(warmCtx)
			}
		}()
	}

	// --- Initialize Handlers ---
	// Cookie security must follow the scheme users actually connect with, not
	// the environment name: a Secure cookie over plain HTTP is silently
	// discarded by the browser, which took every protected image down with it.
	secureCookies := cfg.CookieSecure()
	if cfg.Server.IsProduction() && !secureCookies {
		log.Printf("WARN: cookies are issued without the Secure attribute (COOKIE_SECURE is unset and CORS_ALLOWED_ORIGINS lists no public https:// origin); set COOKIE_SECURE=true if users reach the site over HTTPS")
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
	announcementHandler := handlers.NewAnnouncementHandler(announcementRepo, announcementService, cfg.Upload)
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

	// Stop the model runtime after the HTTP server so in-flight turns finish
	// first. A managed model server is terminated with its whole process group;
	// an externally managed one is simply left alone.
	if assistantRuntime != nil {
		_ = assistantRuntime.Close()
	}

	log.Println("Server stopped")
}
