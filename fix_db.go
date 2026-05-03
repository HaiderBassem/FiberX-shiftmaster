package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://cpper:0770@localhost:5432/shiftmaster?sslmode=disable"
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	query := `
CREATE OR REPLACE FUNCTION audit_employee_shifts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_logs (employee_id, action, table_name, record_id, new_data, created_at)
        VALUES (NEW.created_by, 'INSERT', 'employee_shifts', NEW.id, row_to_json(NEW), CURRENT_TIMESTAMP);
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_logs (employee_id, action, table_name, record_id, old_data, new_data, created_at)
        VALUES (NEW.created_by, 'UPDATE', 'employee_shifts', NEW.id, row_to_json(OLD), row_to_json(NEW), CURRENT_TIMESTAMP);
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_logs (employee_id, action, table_name, record_id, old_data, created_at)
        VALUES (OLD.created_by, 'DELETE', 'employee_shifts', OLD.id, row_to_json(OLD), CURRENT_TIMESTAMP);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v\n", err)
	}

	fmt.Println("Trigger function updated successfully!")
}
