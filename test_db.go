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

	rows, err := conn.Query(ctx, "SELECT id, status, updated_at FROM shift_swaps")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		found = true
		var id, status string
		var updatedAt time.Time
		if err := rows.Scan(&id, &status, &updatedAt); err != nil {
			log.Fatalf("Scan failed: %v\n", err)
		}
		fmt.Printf("Swap ID: %s, Status: %s, Date: %v\n", id, status, updatedAt)
	}
	if !found {
		fmt.Println("No shift swaps found.")
	}
}
