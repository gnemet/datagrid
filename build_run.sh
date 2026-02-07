#!/bin/bash

# build_run.sh - Root entry point
# Optional environment switch (e.g., ./build_run.sh LOCAL)

# Environment switching logic
if [ ! -z "$1" ] && [[ "$1" != -* ]]; then
    if [ -f "config/envs/.env_$1" ]; then
        echo "Switching to environment: $1"
        ./scripts/switch_env.sh "$1"
        shift
    fi
fi

if [ ! -f ".env" ]; then
    echo "No .env found. Attempting host auto-detection..."
    ./scripts/switch_env.sh
    if [ ! -f ".env" ]; then
        echo "Error: .env file not found and auto-detection failed."
        exit 1
    fi
fi

# Run tests from the new pkg location
go test ./pkg/datagrid/... || exit 1

set -a; source .env; set +a
echo "Environment: $ENV_NAME"

# PORT Selection: prioritize environment variable, then config.yaml
CLEANUP_PORT=${PORT:-$(grep -A 1 "server:" config.yaml | grep "port:" | sed 's/.*port: //' | tr -d ' "')}

echo "Cleaning up ports (Range 8085-8090 + $CLEANUP_PORT)..."
for P in {8085..8090}; do
    fuser -k $P/tcp 2>/dev/null || true
done

if [ ! -z "$CLEANUP_PORT" ]; then
    fuser -k $CLEANUP_PORT/tcp 2>/dev/null || true
fi

# App selection logic
APP_TARGET="cursor"
if [ ! -z "$1" ] && [[ "$1" != -* ]]; then
    case $1 in
        cursor|test|validate|dbtest)
            APP_TARGET=$1
            shift
            ;;
    esac
fi

case $APP_TARGET in
    cursor)   GO_TARGET="cmd/cursorapp/main.go" ;;
    test)     GO_TARGET="cmd/testapp/main.go"   ;;
    validate) GO_TARGET="cmd/validate/main.go" ;;
    dbtest)   GO_TARGET="cmd/dbtest/main.go"   ;;
    *)        GO_TARGET="cmd/cursorapp/main.go" ;;
esac

echo "Launching Application: $APP_TARGET ($GO_TARGET)"
go run "$GO_TARGET" "$@"
