package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/repository"
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

	repo := repository.NewTicketRepository(db)
	deptID := uuid.MustParse("7546a31c-bb20-4cfd-97aa-ecc206f6b12e")
	
	tickets, err := repo.GetTicketsForDepartment(context.Background(), deptID)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: %d tickets found\n", len(tickets))
	}
}
