package main

import (
	"context"
	"fmt"
	"log"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
)

func main() {
	cfg, _ := config.Load()
	db, _ := database.New(cfg.Database)
	defer db.Close()

	rows, err := db.Pool().Query(context.Background(), "SELECT id, title, url FROM external_links")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var id, title, url string
		rows.Scan(&id, &title, &url)
		fmt.Printf("Link: %s | %s | %s\n", id, title, url)
	}
}
