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

	rows, err := db.Query(context.Background(), "SELECT id, shift_code, name FROM shifts")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Shifts in DB:")
	for rows.Next() {
		var id, code, name string
		rows.Scan(&id, &code, &name)
		fmt.Printf("ID: %s, Code: %s, Name: %s\n", id, code, name)
	}
}
