package main

import (
	"context"
	"fmt"
	"log"

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

	empRepo := repository.NewEmployeeRepository(db)
	shiftRepo := repository.NewShiftRepository(db)

	ctx := context.Background()

	// Get a test employee (the first one)
	emps, _ := empRepo.GetAll(ctx)
	if len(emps) == 0 {
		log.Fatal("no employees")
	}
	emp := emps[0]
	fmt.Printf("Testing with Employee: %s %s (Dept: %v)\n", emp.FirstName, emp.LastName, emp.DepartmentID)

	if emp.DepartmentID == nil {
		log.Fatal("Employee has no department")
	}

	deptEmps, _ := empRepo.GetByDepartment(ctx, *emp.DepartmentID)
	fmt.Printf("Total employees in dept: %d\n", len(deptEmps))

	shifts, _ := shiftRepo.GetAll(ctx, emp.DepartmentID)
	for _, s := range shifts {
		fmt.Printf("Shift: %s (%s) ID: %s\n", s.Name, s.ShiftCode, s.ID)
	}

	for _, e := range deptEmps {
		fmt.Printf("Emp: %s %s, DefaultShiftID: %v\n", e.FirstName, e.LastName, e.DefaultShiftID)
	}
}
