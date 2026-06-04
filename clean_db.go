package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
)

func main() {
	fmt.Println("Starting Database Cleanup...")

	_ = godotenv.Load() // Load local .env if present

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "shiftmaster")
	sslMode := getEnv("DB_SSLMODE", "disable")



	// Since pkg/database needs a config object, we'll construct a minimal valid one.
	// Actually, database.New uses the config struct, so let's just make one directly.
	dbCfg := config.DatabaseConfig{
		Host:              dbHost,
		Port:              dbPort,
		User:              dbUser,
		Password:          dbPass,
		DBName:            dbName,
		SSLMode:           sslMode,
		MaxOpenConns:      10,
		MinConns:          2,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   time.Minute * 30,
		ConnectTimeout:    time.Second * 5,
		HealthCheckPeriod: time.Minute,
	}

	db, err := database.New(dbCfg)
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

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
