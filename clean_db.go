package main

import (
	"context"
	"fmt"
	"log"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
)

func main() {
	fmt.Println("Starting Database Cleanup...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// List of tables to wipe completely
	tablesToDelete := []string{
		"leaves",
		"employee_leave_balances",
		"shift_swaps",
		"shift_handovers",
		"task_executions",
		"task_assignments",
		"notifications",
		"push_subscriptions",
		"audit_logs",
		"daily_stats",
		"announcements",
		"help_document_access",
		"help_documents",
		"info_table_rows",
		"info_table_employee_access",
		"info_table_department_access",
		"info_tables",
	}

	for _, table := range tablesToDelete {
		fmt.Printf("Truncating table: %s...\n", table)
		// Using CASCADE to ensure dependent rows in other tables in this list are also wiped if any constraints exist.
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE;", table)
		_, err := db.Exec(ctx, query)
		if err != nil {
			log.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}

	fmt.Println("✅ Database cleanup completed successfully!")
	fmt.Println("Note: Departments, Employees, Shifts, and Employee Schedules were NOT deleted.")
}
