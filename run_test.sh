#!/bin/bash

# run_test.sh - Run Go tests and verify environment
set -a; source .env; set +a

echo "Running Datagrid Unit Tests..."
go test -v ./...

echo ""
echo "Verifying Database Connection..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT count(*) FROM $DB_SCHEMA.personnel;"

echo ""
echo "Building Test Application..."
go build -o testapp ./cmd/testapp/main.go
if [ $? -eq 0 ]; then
    echo "Build Successful."
else
    echo "Build Failed."
    exit 1
fi
