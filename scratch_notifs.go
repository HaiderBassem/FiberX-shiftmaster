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
		log.Fatal(err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(context.Background(), "SELECT id, recipient_id, title, message, created_at FROM notifications ORDER BY created_at DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Recent Notifications:")
	for rows.Next() {
		var id, recipient_id, title, message, created_at string
		rows.Scan(&id, &recipient_id, &title, &message, &created_at)
		fmt.Printf("Time: %s\nTitle: %s\nMessage: %s\n---\n", created_at, title, message)
	}
}
