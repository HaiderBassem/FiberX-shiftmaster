#!/bin/bash
set -e

echo "Clearing the database schema..."
psql "postgres://cpper:0770@localhost:5432/shiftmaster?sslmode=disable" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO cpper; GRANT ALL ON SCHEMA public TO public;"

echo "Running complete database migration and seed..."
psql "postgres://cpper:0770@localhost:5432/shiftmaster?sslmode=disable" -f "internal/database/database.sql"

echo "Database seeded successfully!"
