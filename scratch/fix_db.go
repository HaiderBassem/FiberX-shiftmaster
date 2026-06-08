package main

import (
	"context"
	"fmt"
	"log"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	queries := []string{
		"ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_manage_help_docs BOOLEAN DEFAULT false;",
		"ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_manage_fiberx_data BOOLEAN DEFAULT false;",
		"ALTER TABLE employees ADD COLUMN IF NOT EXISTS ui_preferences JSONB DEFAULT '{}'::jsonb;",
		"ALTER TABLE fiberx_data_employee_access ADD COLUMN IF NOT EXISTS granted_by UUID REFERENCES employees(id) ON DELETE SET NULL;",
		"ALTER TABLE fiberx_data_department_shares ADD COLUMN IF NOT EXISTS granted_by UUID REFERENCES employees(id) ON DELETE SET NULL;",
	}

	for _, q := range queries {
		_, err := db.Exec(ctx, q)
		if err != nil {
			fmt.Printf("Warning: Query failed: %s\nError: %v\n", q, err)
		} else {
			fmt.Printf("Success: %s\n", q)
		}
	}
	fmt.Println("Database migration completed!")
}
