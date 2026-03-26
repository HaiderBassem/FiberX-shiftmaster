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

	// --- Auto-apply task_boards migration (non-destructive) ---
	if _, err := db.Pool().Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS task_boards (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(100) NOT NULL,
			description TEXT,
			recurrence_type VARCHAR(10) NOT NULL DEFAULT 'weekly' CHECK (recurrence_type IN ('daily', 'weekly')),
			is_active BOOLEAN DEFAULT true,
			created_by UUID REFERENCES employees(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name='task_schedules' AND column_name='board_id'
			) THEN
				ALTER TABLE task_schedules ADD COLUMN board_id UUID REFERENCES task_boards(id) ON DELETE CASCADE;
			END IF;
		END $$;
		DO $$ BEGIN
			BEGIN
				ALTER TABLE task_schedules ALTER COLUMN schedule_type DROP NOT NULL;
			EXCEPTION WHEN others THEN
				-- keep startup resilient if schema already differs
				NULL;
			END;
		END $$;
		-- Add started_at to task_executions
		DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name='task_executions' AND column_name='started_at'
			) THEN
				ALTER TABLE task_executions ADD COLUMN started_at TIMESTAMP;
			END IF;
		END $$;
	`); err != nil {
		log.Printf("WARNING: task_boards migration: %v", err)
	} else {
		log.Println("task_boards migration applied successfully")
	}

	// --- Ensure swap_status enum supports intermediate approval state ---
	if _, err := db.Pool().Exec(context.Background(), `
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'swap_status') THEN
				IF NOT EXISTS (
					SELECT 1
					FROM pg_enum e
					JOIN pg_type t ON t.oid = e.enumtypid
					WHERE t.typname = 'swap_status'
					  AND e.enumlabel = 'employee_accepted'
				) THEN
					ALTER TYPE swap_status ADD VALUE 'employee_accepted';
				END IF;
			END IF;
		END $$;
	`); err != nil {
		log.Printf("WARNING: swap_status enum migration: %v", err)
	} else {
		log.Println("swap_status enum migration applied successfully")
	}

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

	// --- Initialize Services ---
	authService := service.NewAuthService(employeeRepo, cfg.JWT.BcryptCost)
	notifService := service.NewNotificationService(notifRepo)
	employeeService := service.NewEmployeeService(employeeRepo, departmentRepo, authService)
	scheduleService := service.NewScheduleService(scheduleRepo, employeeRepo, shiftRepo, notifService, db)
	leaveService := service.NewLeaveService(leaveRepo, employeeRepo, scheduleRepo, notifService)
	swapService := service.NewSwapService(swapRepo, scheduleRepo, employeeRepo, taskRepo, notifService, db)
	taskService := service.NewTaskService(taskRepo, boardRepo, employeeRepo, scheduleRepo)
	auditService := service.NewAuditService(auditRepo)

	// --- Initialize Handlers ---
	authHandler := handlers.NewAuthHandler(authService, employeeService, cfg.JWT.Secret, cfg.JWT.AccessExpireMin, cfg.JWT.RefreshExpireDays)
	empHandler := handlers.NewEmployeeHandler(employeeService)
	deptHandler := handlers.NewDepartmentHandler(departmentRepo, employeeRepo)
	shiftHandler := handlers.NewShiftHandler(shiftRepo)
	scheduleHandler := handlers.NewScheduleHandler(scheduleService)
	leaveHandler := handlers.NewLeaveHandler(leaveService)
	swapHandler := handlers.NewSwapHandler(swapService)
	taskHandler := handlers.NewTaskHandler(taskService)
	notifHandler := handlers.NewNotificationHandler(notifService)
	auditHandler := handlers.NewAuditHandler(auditService)

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
	SetupRouter(r, cfg.JWT.Secret,
		authHandler, empHandler, deptHandler, shiftHandler,
		scheduleHandler, leaveHandler, swapHandler, taskHandler, notifHandler, auditHandler,
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
