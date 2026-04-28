package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://cpper:0770@localhost:5432/shiftmaster?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	// Fetch the employee ID
	var empID string
	err = conn.QueryRow(ctx, "SELECT id FROM employees LIMIT 1").Scan(&empID)
	
	rows, err := conn.Query(ctx, "SELECT id, title, type FROM notifications WHERE recipient_id=$1", empID)
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, ntype string
		if err := rows.Scan(&id, &title, &ntype); err != nil {
			log.Fatalf("Scan failed: %v\n", err)
		}
		fmt.Printf("Notif - ID: %s, Title: %s, Type: %s\n", id, title, ntype)
	}
}
