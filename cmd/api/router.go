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
	leaveTypeH *handlers.LeaveTypeHandler,
	infoTableH *handlers.InfoTableHandler,
	helpDocH *handlers.HelpDocumentHandler,
	announcementH *handlers.AnnouncementHandler,
	pushH *handlers.PushHandler,
) {
	api := r.Group("/api")

	// --- Public routes ---
	auth := api.Group("/auth")
	{
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.Refresh)
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
			employees.PUT("/:id/password", empH.UpdatePassword)
			employees.PUT("/:id/help-permission", empH.UpdateHelpPermission)
			employees.PUT("/:id/announcement-permission", empH.UpdateAnnouncementPermission)
			employees.PUT("/:id/table-permission", empH.UpdateTablePermission)
			employees.PUT("/:id/preferences", empH.UpdatePreferences)
		}

		// Departments (read)
		departments := protected.Group("/departments")
		{
			departments.GET("", deptH.List)
			departments.GET("/my-managed", deptH.MyManaged)
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

		// Leave Types (read)
		leaveTypes := protected.Group("/leave-types")
		{
			leaveTypes.GET("", leaveTypeH.GetAll)
			leaveTypes.GET("/:id", leaveTypeH.GetByID)
		}

		// Swaps
		swaps := protected.Group("/swaps")
		{
			swaps.POST("", swapH.Request)
			swaps.GET("/me", swapH.MyRequests)
			swaps.GET("/pending/for-me", swapH.PendingForMe)
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
			tasks.GET("/boards/:id/recurring", taskH.ListRecurringByBoard)
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

		// Info Tables (Dynamic Tables)
		infoTables := protected.Group("/info-tables")
		{
			infoTables.GET("", infoTableH.GetVisibleTables)
			infoTables.POST("", infoTableH.CreateTable)
			infoTables.GET("/:id/rows", infoTableH.GetTableRows)
			infoTables.POST("/:id/rows", infoTableH.CreateTableRow)
			infoTables.PUT("/:id/rows/:rowId", infoTableH.UpdateTableRow)
			infoTables.DELETE("/:id/rows/:rowId", infoTableH.DeleteTableRow)
			
			// Access Management
			infoTables.GET("/:id/access", infoTableH.GetAccessLists)
			infoTables.POST("/:id/access", infoTableH.AddEmployeeAccess)
			infoTables.DELETE("/:id/access/employee/:employeeId", infoTableH.RemoveEmployeeAccess)
			infoTables.POST("/:id/department-access", infoTableH.ShareWithDepartment)
		}

		// Help Documents
		helpDocs := protected.Group("/help-docs")
		{
			helpDocs.GET("", helpDocH.GetVisibleDocuments)
			helpDocs.GET("/:id", helpDocH.GetDocument)
			helpDocs.POST("", helpDocH.CreateDocument)
			helpDocs.PUT("/:id", helpDocH.UpdateDocument)
			helpDocs.DELETE("/:id", helpDocH.DeleteDocument)
			
			// Access Management
			helpDocs.GET("/:id/access", helpDocH.GetAccessList)
			helpDocs.POST("/:id/access", helpDocH.SetEmployeeAccess)
		}

		// --- Supervisor routes (manager + admin + team_leader) ---
		// READ access and approval workflows
		supervisor := protected.Group("")
		supervisor.Use(middleware.RequireRole("manager", "admin", "team_leader"))
		{
			// Leave approvals (all supervisors can approve)
			supervisor.GET("/leaves/coverage-preview", leaveH.CoveragePreview)
			supervisor.POST("/leaves/:id/approve/team-leader", leaveH.ApproveByTeamLeader)
			supervisor.POST("/leaves/:id/approve/manager", leaveH.ApproveByManager)
			supervisor.POST("/leaves/:id/reject", leaveH.Reject)
			supervisor.POST("/leaves/:id/cancel-approval", leaveH.CancelApproval)

			// Swap approvals (all supervisors can approve/reject)
			supervisor.GET("/swaps/pending/manager", swapH.PendingForManager)
			supervisor.POST("/swaps/:id/approve", swapH.Approve)
			supervisor.POST("/swaps/:id/reject", swapH.Reject)
			supervisor.POST("/swaps/:id/cancel-approval", swapH.CancelApproval)
			supervisor.GET("/swaps/history", swapH.SwapHistory)

			// Task history (read — all supervisors can view)
			supervisor.GET("/tasks/history", taskH.TaskHistory)

			// Leave history (read — all supervisors can view approval/rejection history)
			supervisor.GET("/leaves/history", leaveH.LeaveHistory)

			// Pending leaves with employee details for approval dashboard
			supervisor.GET("/leaves/pending/rich", leaveH.PendingRich)
		}

		// --- Team Leader + Admin WRITE routes ---
		// Only team_leader and admin can modify schedules, tasks, and employees
		// Manager is VIEW ONLY
		tlWrite := protected.Group("")
		tlWrite.Use(middleware.RequireRole("team_leader", "admin"))
		{
			// Task management (create, assign, update, delete)
			tlWrite.POST("/tasks/schedules", taskH.CreateSchedule)
			tlWrite.PUT("/tasks/schedules/:id", taskH.UpdateSchedule)
			tlWrite.PATCH("/tasks/schedules/:id/toggle", taskH.ToggleActive)
			tlWrite.DELETE("/tasks/schedules/:id", taskH.DeleteSchedule)
			tlWrite.POST("/tasks/assign", taskH.Assign)
			tlWrite.DELETE("/tasks/assignments/:id", taskH.DeleteAssignment)

			// Recurring assignments (create/delete)
			tlWrite.POST("/tasks/recurring-assign", taskH.RecurringAssign)
			tlWrite.DELETE("/tasks/recurring-assignments/:id", taskH.DeleteRecurringAssignment)
			tlWrite.POST("/tasks/recurring-assignments/remove", taskH.DeleteRecurringAssignmentByKey)

			// Board management (create, update, delete)
			tlWrite.POST("/tasks/boards", taskH.CreateBoard)
			tlWrite.PUT("/tasks/boards/:id", taskH.UpdateBoard)
			tlWrite.DELETE("/tasks/boards/:id", taskH.DeleteBoard)

			// Employee creation/editing/deletion
			tlWrite.POST("/employees", empH.Create)
			tlWrite.PUT("/employees/:id", empH.Update)
			tlWrite.PATCH("/employees/:id/status", empH.UpdateStatus)
			tlWrite.DELETE("/employees/:id", empH.Delete)

			// Schedule editing (manual create/update for a day)
			tlWrite.POST("/schedules/shifts/set", scheduleH.SetEmployeeShift)
			tlWrite.DELETE("/schedules/shifts/:id", scheduleH.DeleteEmployeeShift)

			// Shift management
			tlWrite.POST("/shifts", shiftH.Create)
			tlWrite.PUT("/shifts/:id", shiftH.Update)
			tlWrite.DELETE("/shifts/:id", shiftH.Delete)
		}



		// Push Notifications
		push := protected.Group("/push")
		{
			push.GET("/public-key", pushH.GetPublicKey)
			push.POST("/subscribe", pushH.Subscribe)
		}

		// Audit Logs (Admin only)
		audit := protected.Group("/audit")
		{
			audit.GET("", auditH.List)
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
			
			// Leave Types management
			admin.POST("/leave-types", leaveTypeH.Create)
			admin.PUT("/leave-types/:id", leaveTypeH.Update)
			admin.DELETE("/leave-types/:id", leaveTypeH.Delete)

		}

		adminOnly := protected.Group("")
		adminOnly.Use(middleware.RequireRole("admin"))
		{
			adminDept := adminOnly.Group("/departments")
			{
				adminDept.POST("", deptH.Create)
				adminDept.PUT("/:id", deptH.Update)
				adminDept.DELETE("/:id", deptH.Delete)
				adminDept.POST("/:id/managers", deptH.AddManager)
				adminDept.DELETE("/:id/managers/:manager_id", deptH.RemoveManager)
			}
		}

		// Announcements
		announcements := protected.Group("/announcements")
		{
			announcements.GET("/active", announcementH.GetActive)
			announcements.GET("", announcementH.GetAll)
			announcements.POST("", announcementH.Create)
			announcements.DELETE("/:id", announcementH.Delete)
			announcements.PUT("/:id/activate", announcementH.SetActive)
		}

	}
}
