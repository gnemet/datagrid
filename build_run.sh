#!/bin/bash
# Build and Run script for Datagrid Test App

echo "Cleaning up port 8085..."
lsof -ti :8085 | xargs kill -9 2>/dev/null || true

echo "Initializing Database..."
psql -h localhost -p 5433 -U root -d db01 -f database/init_db.sql

echo "Building Application..."
go build -o bin/testapp cmd/testapp/main.go

echo "Starting Application..."
./bin/testapp
