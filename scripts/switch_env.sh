#!/usr/bin/env bash

# switch_env.sh - Select active environment from opt/envs/

TARGET=$1

# If no argument provided, try auto-detection
if [ -z "$TARGET" ]; then
    CURRENT_HOST=$(hostname)
    if [ -f "opt/envs/.env_$CURRENT_HOST" ] || [ -f "opt/envs/.env_$CURRENT_HOST.gpg" ]; then
        TARGET="$CURRENT_HOST"
        echo "Detected Host: $CURRENT_HOST -> Selecting '$TARGET'"
    else
        echo "Unknown host: $CURRENT_HOST (File opt/envs/.env_$CURRENT_HOST not found)"
        echo "Example usage: ./scripts/switch_env.sh zenbook"
        echo "Currently available environments in opt/envs/:"
        ls opt/envs/
        exit 1
    fi
fi

SOURCE_FILE="opt/envs/.env_${TARGET}"

if [ ! -f "$SOURCE_FILE" ]; then
    if [ -f "${SOURCE_FILE}.gpg" ]; then
        echo "Source file '$SOURCE_FILE' missing, but encrypted version found. Unlocking..."
        ./scripts/vault.sh unlock "$SOURCE_FILE"
    else
        echo "Error: Source file '$SOURCE_FILE' does not exist."
        exit 1
    fi
fi

cp "$SOURCE_FILE" .env
ENV_NAME=$(grep "^ENV_NAME=" .env | cut -d'=' -f2)
[ -z "$ENV_NAME" ] && ENV_NAME="$TARGET"

echo "Switched environment to '$ENV_NAME' (source: $SOURCE_FILE)"
grep "DB_HOST" .env || true
grep "DB_PORT" .env || true
grep "DB_NAME" .env || true
