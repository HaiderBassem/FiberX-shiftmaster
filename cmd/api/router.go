package main

import (
	"github.com/gin-gonic/gin"

	"shiftmaster-backend/cmd/api/handlers"
	"shiftmaster-backend/internal/middleware"
)

// SetupRouter configures all API routes on the Gin engine.
func SetupRouter(
	r *gin.Engine,
	jwtSecret string,
	authH *handlers.AuthHandler,
	empH *handlers.EmployeeHandler,
	deptH *handlers.DepartmentHandler,
	shiftH *handlers.ShiftHandler,
	scheduleH *handlers.ScheduleHandler,
	leaveH *handlers.LeaveHandler,
	swapH *handlers.SwapHandler,
	taskH *handlers.TaskHandler,
	notifH *handlers.NotificationHandler,
	auditH *handlers.AuditHandler,
) {
	api := r.Group("/api")

	// --- Public routes ---
	auth := api.Group("/auth")
	{
		auth.POST("/login", authH.Login)
	}

	// --- Protected routes (JWT required) ---
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(jwtSecret))
	{
		// Auth
		protectedAuth := protected.Group("/auth")
		{
			protectedAuth.GET("/me", authH.Me)
			protectedAuth.POST("/change-password", authH.ChangePassword)
		}

		// Employees
		employees := protected.Group("/employees")
		{
			employees.GET("", empH.List)
			employees.GET("/:id", empH.GetByID)
		}

		// Departments (read)
		departments := protected.Group("/departments")
		{
			departments.GET("", deptH.List)
			departments.GET("/:id", deptH.GetByID)
		}

		// Shifts (read)
		shifts := protected.Group("/shifts")
		{
			shifts.GET("", shiftH.List)
			shifts.GET("/:id", shiftH.GetByID)
		}

		// Schedules
		schedules := protected.Group("/schedules")
		{
			schedules.GET("/daily", scheduleH.DailyShifts)
			schedules.GET("/employee/:id", scheduleH.EmployeeShifts)
			schedules.GET("/replacements", scheduleH.AvailableReplacements)

			scheduleShifts := schedules.Group("/shifts")
			{
				scheduleShifts.POST("/:id/check-in", scheduleH.CheckIn)
				scheduleShifts.POST("/:id/check-out", scheduleH.CheckOut)
			}
		}

		// Leaves
		leaves := protected.Group("/leaves")
		{
			leaves.POST("", leaveH.Request)
			leaves.GET("/me", leaveH.MyLeaves)
			leaves.GET("/pending", leaveH.PendingForApproval)
		}

		// Swaps
		swaps := protected.Group("/swaps")
		{
			swaps.POST("", swapH.Request)
			swaps.GET("/me", swapH.MyRequests)
			swaps.GET("/pending", swapH.PendingForMe)
			swaps.GET("/eligible-targets", swapH.EligibleTargets)
			swaps.POST("/:id/respond", swapH.Respond)
			swaps.POST("/:id/cancel", swapH.Cancel)
		}

		// Tasks
		tasks := protected.Group("/tasks")
		{
			tasks.GET("/schedules", taskH.ListSchedules)
			tasks.GET("/assignments", taskH.DailyAssignments)
			tasks.GET("/assignments/me", taskH.MyTasks)
			tasks.GET("/my-week", taskH.MyWeeklyTasks)
			tasks.GET("/eligible-assignees", taskH.EligibleAssignees)
			tasks.GET("/boards", taskH.ListBoards)
			tasks.GET("/boards/stats", taskH.GetBoardStats)
			tasks.GET("/boards/eligible-employees", taskH.GetBoardEligibleEmployees)
			tasks.GET("/boards/:id/view", taskH.GetBoardView)
			tasks.POST("/executions/:id/start", taskH.StartExecution)
			tasks.PATCH("/executions/:id/status", taskH.UpdateExecutionStatus)
			tasks.POST("/executions/:id/complete", taskH.CompleteExecution)
		}

		// Notifications
		notifs := protected.Group("/notifications")
		{
			notifs.GET("", notifH.List)
			notifs.GET("/unread", notifH.Unread)
			notifs.GET("/unread/count", notifH.UnreadCount)
			notifs.POST("/:id/read", notifH.MarkAsRead)
			notifs.POST("/read-all", notifH.MarkAllAsRead)
		}

		// Activity history (audit logs)
		protected.GET("/activity", auditH.ListActivity)

		// --- Supervisor routes (manager + admin + team_leader) ---
		// These are for approval workflows where team leaders also participate
		supervisor := protected.Group("")
		supervisor.Use(middleware.RequireRole("manager", "admin", "team_leader"))
		{
			// Leave approvals
			supervisor.GET("/leaves/coverage-preview", leaveH.CoveragePreview)
			supervisor.POST("/leaves/:id/approve/team-leader", leaveH.ApproveByTeamLeader)
			supervisor.POST("/leaves/:id/approve/manager", leaveH.ApproveByManager)
			supervisor.POST("/leaves/:id/reject", leaveH.Reject)

			// Swap approvals
			// (Swap approvals are handled in a team_leader-only route group below.)

			// Task management (create, assign, update, delete)
			supervisor.POST("/tasks/schedules", taskH.CreateSchedule)
			supervisor.PUT("/tasks/schedules/:id", taskH.UpdateSchedule)
			supervisor.PATCH("/tasks/schedules/:id/toggle", taskH.ToggleActive)
			supervisor.DELETE("/tasks/schedules/:id", taskH.DeleteSchedule)
			supervisor.POST("/tasks/assign", taskH.Assign)
			supervisor.DELETE("/tasks/assignments/:id", taskH.DeleteAssignment)

			// Board management (create, update, delete) — team_leader, manager, admin
			supervisor.POST("/tasks/boards", taskH.CreateBoard)
			supervisor.PUT("/tasks/boards/:id", taskH.UpdateBoard)
			supervisor.DELETE("/tasks/boards/:id", taskH.DeleteBoard)

			// Employee creation (team_leader, manager, admin can create)
			supervisor.POST("/employees", empH.Create)
			// Employee editing/deactivation/deletion (team_leader, manager, admin)
			supervisor.PUT("/employees/:id", empH.Update)
			// Employee deactivation/deletion (team_leader, manager, admin)
			supervisor.PATCH("/employees/:id/status", empH.UpdateStatus)
			supervisor.DELETE("/employees/:id", empH.Delete)

			// Schedule editing (manual create/update for a day)
			supervisor.POST("/schedules/shifts/set", scheduleH.SetEmployeeShift)

			// Shift management (team_leader + manager + admin)
			supervisor.POST("/shifts", shiftH.Create)
			supervisor.PUT("/shifts/:id", shiftH.Update)
			supervisor.DELETE("/shifts/:id", shiftH.Delete)
		}

		// Phase 3: swap approvals are team_leader only (skip manager).
		tlOnly := protected.Group("")
		tlOnly.Use(middleware.RequireRole("team_leader"))
		{
			tlOnly.GET("/swaps/pending/manager", swapH.PendingForManager)
			tlOnly.POST("/swaps/:id/approve", swapH.Approve)
			tlOnly.POST("/swaps/:id/reject", swapH.Reject)
		}

		// --- Admin only routes (manager + admin) ---
		// These are for CRUD and system management
		admin := protected.Group("")
		admin.Use(middleware.RequireRole("manager", "admin"))
		{
			// Auth admin
			admin.POST("/auth/reset-password/:id", authH.ResetPassword)

			// Department management (admin only)
			// Managers can view departments but cannot create/update/delete them.

			// Schedule management
			admin.POST("/schedules/:id/publish", scheduleH.Publish)
			admin.POST("/schedules/shifts/:id/replace", scheduleH.AssignReplacement)

		}

		adminOnly := protected.Group("")
		adminOnly.Use(middleware.RequireRole("admin"))
		{
			adminDept := adminOnly.Group("/departments")
			{
				adminDept.POST("", deptH.Create)
				adminDept.PUT("/:id", deptH.Update)
				adminDept.DELETE("/:id", deptH.Delete)
			}
		}
	}
}
