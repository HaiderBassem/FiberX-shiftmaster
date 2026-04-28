package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
	"shiftmaster-backend/pkg/database"
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

	// We need a dummy database.DB object. It wraps *pgxpool.Pool, but here we just have a connection.
	// Since repository needs *database.DB, which has a pool, we need to initialize it properly.
	// We can just execute the raw SQL that `SendNotification` does, using the actual data from a leave.
}
