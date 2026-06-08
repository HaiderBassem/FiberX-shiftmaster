package main

import (
	"context"
	"fmt"
	"log"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/pkg/database"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load()
	if err != nil { log.Fatal(err) }
	db, err := database.New(cfg.Database)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	ctx := context.Background()

	// Try to add the missing columns if they don't exist
	db.Exec(ctx, "ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_manage_help_docs BOOLEAN DEFAULT false;")
	db.Exec(ctx, "ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_manage_fiberx_data BOOLEAN DEFAULT false;")
	db.Exec(ctx, "ALTER TABLE employees ADD COLUMN IF NOT EXISTS ui_preferences JSONB DEFAULT '{}';")

	empRepo := repository.NewEmployeeRepository(db)
	repo := repository.NewFiberxDataRepository(db)
	svc := service.NewFiberxDataService(repo, empRepo)

	// Pick a document to share
	rows, err := db.Query(ctx, "SELECT id, department_id, created_by FROM fiberx_data LIMIT 1")
	if err != nil { log.Fatal(err) }
	var docID, docDeptID, docCreatedBy uuid.UUID
	for rows.Next() {
		rows.Scan(&docID, &docDeptID, &docCreatedBy)
	}
	rows.Close()

	fmt.Printf("Doc ID: %s, Dept: %s, CreatedBy: %s\n", docID, docDeptID, docCreatedBy)

	// Test Team Leader logic
	targetDeptID, _ := uuid.NewRandom()

	// 1. Team Leader with departmentID matched
	err = svc.SetDepartmentShare(ctx, docID, targetDeptID, "read", &docDeptID, docCreatedBy, "team_leader")
	if err != nil {
		fmt.Printf("Service Error Team Leader: %v\n", err)
	} else {
		fmt.Printf("Service Success Team Leader!\n")
	}

	// 2. Admin with departmentID matched
	err = svc.SetDepartmentShare(ctx, docID, targetDeptID, "read", &docDeptID, docCreatedBy, "admin")
	if err != nil {
		fmt.Printf("Service Error Admin: %v\n", err)
	} else {
		fmt.Printf("Service Success Admin!\n")
	}
	
	// 3. Manager with departmentID matched
	err = svc.SetDepartmentShare(ctx, docID, targetDeptID, "read", &docDeptID, docCreatedBy, "manager")
	if err != nil {
		fmt.Printf("Service Error Manager: %v\n", err)
	} else {
		fmt.Printf("Service Success Manager!\n")
	}
}
