package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sqlBytes, err := ioutil.ReadFile("internal/database/migrations/022_add_employee_ui_preferences.sql")
	if err != nil {
		log.Fatalf("Failed to read migration: %v", err)
	}

	_, err = db.Pool().Exec(context.Background(), string(sqlBytes))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration 022 applied successfully.")
}
