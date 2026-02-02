#!/bin/bash
# Build and Run script for Datagrid Test App

echo "Cleaning up port 8085..."
lsof -ti :8085 | xargs kill -9 2>/dev/null || true

echo "Initializing Database..."
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f database/init_db.sql

echo "Building Application..."
go build -o bin/testapp cmd/testapp/main.go

echo "Starting Application..."
./bin/testapp
