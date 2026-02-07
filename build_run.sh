#!/bin/bash

# build_run.sh - Root entry point
# Optional environment switch (e.g., ./build_run.sh LOCAL)

if [ ! -z "$1" ] && [[ "$1" != -* ]]; then
    if [ -f "opt/envs/.env_$1" ]; then
        echo "Switching to environment: $1"
        ./scripts/switch_env.sh "$1"
        shift
    else
        echo "Error: Environment config 'opt/envs/.env_$1' not found."
        exit 1
    fi
else
    echo "Using current .env configuration..."
fi

if [ ! -f ".env" ]; then
    echo "Error: .env file not found. Run ./scripts/switch_env.sh <ENV> or restore from vault."
    exit 1
fi

set -a; source .env; set +a

# Verify critical vars if needed, for now just echo
echo "Environment: $ENV_NAME"

# PORT Cleanup: find port in config.yaml and kill process using it
PORT=$(grep -A 1 "server:" config.yaml | grep "port:" | sed 's/.*port: //' | tr -d ' "')
if [ ! -z "$PORT" ]; then
    echo "Cleaning up port $PORT..."
    
    # Try multiple methods to ensure the port is free
    PID=$(lsof -t -i:$PORT 2>/dev/null)
    if [ ! -z "$PID" ]; then
        echo "Killing PID $PID..."
        kill -9 $PID 2>/dev/null || true
        
        # Wait up to 5 seconds for the port to be free
        for i in {1..10}; do
            if ! lsof -i:$PORT >/dev/null 2>&1; then
                echo "Port $PORT is now free."
                break
            fi
            sleep 0.5
        done
    fi
fi

go run cmd/testapp/main.go "$@"
