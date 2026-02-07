#!/bin/bash
set -e

VERSION=$(grep 'version:' config.yaml | awk '{print $2}' | tr -d '"')
DIST_NAME="datagrid-$VERSION"
DIST_DIR="dist/$DIST_NAME"

echo "Packing Datagrid $VERSION..."

# Clean
rm -rf dist
mkdir -p $DIST_DIR

# Build
echo "Building binary..."
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $DIST_DIR/cursorapp ./cmd/cursorapp/main.go

# Copy Assets
echo "Copying assets..."
cp config.yaml $DIST_DIR/
cp -r ui $DIST_DIR/
cp -r pkg/datagrid/ui "$DIST_DIR/pkg/datagrid/"
cp -r scripts $DIST_DIR/
cp -r deploy $DIST_DIR/

# Create Archive
echo "Creating archive..."
cd dist
tar -czvf $DIST_NAME.tar.gz $DIST_NAME

echo "Pack created at dist/$DIST_NAME.tar.gz"
