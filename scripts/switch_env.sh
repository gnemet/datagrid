#!/usr/bin/env bash

# switch_env.sh - Select active environment from opt/envs/

TARGET=$1

# If no argument provided, try auto-detection
if [ -z "$TARGET" ]; then
    CURRENT_HOST=$(hostname)
    if [ -f "config/envs/.env_$CURRENT_HOST" ]; then
        TARGET="$CURRENT_HOST"
        echo "Detected Host: $CURRENT_HOST -> Selecting '$TARGET'"
    else
        echo "Unknown host: $CURRENT_HOST (File config/envs/.env_$CURRENT_HOST not found)"
        echo "Example usage: ./scripts/switch_env.sh LOCAL"
        echo "Currently available environments in config/envs/:"
        ls config/envs/
        exit 1
    fi
fi

SOURCE_FILE="config/envs/.env_${TARGET}"

if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' does not exist."
    exit 1
fi

cp "$SOURCE_FILE" .env
echo "Switched environment to '$TARGET' (copied $SOURCE_FILE to root .env)"
