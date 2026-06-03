package main

import (
	"context"
	"fmt"
	"os"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
)

func main() {
	cfg, _ := config.Load()
	db, err := database.New(cfg.Database)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
		return
	}
	defer db.Close()

	content, err := os.ReadFile("internal/database/migrations/027_shift_handovers.sql")
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
		return
	}

	_, err = db.Exec(context.Background(), string(content))
	if err != nil {
		fmt.Printf("Migration error: %v\n", err)
		return
	}
	fmt.Println("Migration 027 applied successfully")
}
