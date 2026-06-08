package main

import (
	"context"
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

	// 1. Create module_departments
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS module_departments (
			module_name VARCHAR(50) NOT NULL,
			department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
			granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (module_name, department_id)
		)
	`)
	if err != nil {
		log.Fatalf("Error creating module_departments: %v", err)
	}

	// 2. Create module_exclusions
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS module_exclusions (
			module_name VARCHAR(50) NOT NULL,
			employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
			excluded_by UUID REFERENCES employees(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (module_name, employee_id)
		)
	`)
	if err != nil {
		log.Fatalf("Error creating module_exclusions: %v", err)
	}

	log.Println("Migrations executed successfully!")
}
