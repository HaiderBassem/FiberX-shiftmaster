package main

import (
	"context"
	"log"
	"os"

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

	sqlContent, err := os.ReadFile("database_update.sql")
	if err != nil {
		log.Fatalf("Error reading sql file: %v", err)
	}

	_, err = db.Exec(ctx, string(sqlContent))
	if err != nil {
		log.Fatalf("Error executing migration: %v", err)
	}

	log.Println("Migrations executed successfully!")
}
