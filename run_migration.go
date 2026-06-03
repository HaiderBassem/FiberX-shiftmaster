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

	migrations := []string{
		"internal/database/migrations/025_leave_balances.sql",
		"internal/database/migrations/026_monthly_hourly_leaves.sql",
	}

	for _, m := range migrations {
		sqlBytes, err := ioutil.ReadFile(m)
		if err != nil {
			log.Fatalf("Failed to read migration %s: %v", m, err)
		}

		_, err = db.Pool().Exec(context.Background(), string(sqlBytes))
		if err != nil {
			log.Fatalf("Migration %s failed: %v", m, err)
		}
		fmt.Printf("Migration %s applied successfully.\n", m)
	}
}
