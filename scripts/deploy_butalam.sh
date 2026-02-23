#!/bin/bash
set -e

# deploy_butalam.sh - Build and deploy Datagrid to butalam environment
# Modeled after jiramntr's deploy pattern.

# 1. Prepare Environment
echo "Preparing environment..."
./scripts/switch_env.sh IT-2057-butalam

ENV_FILE=".env"
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: Environment file $ENV_FILE not found."
    exit 1
fi

# Parse deployment variables from .env
REMOTE_SSH_VAL=$(grep "^REMOTE_SSH=" "$ENV_FILE" | cut -d'=' -f2- | tr -d '"' | tr -d '\r')
REMOTE_PWD_VAL=$(grep "^REMOTE_PWD=" "$ENV_FILE" | cut -d'=' -f2- | tr -d '"' | tr -d '\r')
DB_USER_VAL=$(grep "^DB_USER=" "$ENV_FILE" | cut -d'=' -f2- | tr -d '"' | tr -d '\r')
DB_PASSWORD_VAL=$(grep "^DB_PASSWORD=" "$ENV_FILE" | cut -d'=' -f2- | tr -d '"' | tr -d '\r')
DB_NAME_VAL=$(grep "^DB_NAME=" "$ENV_FILE" | cut -d'=' -f2- | tr -d '"' | tr -d '\r')

# Deployment settings
REMOTE_SSH_CMD=${REMOTE_SSH_VAL:-"ssh nemetg@sys-butalam01"}
REMOTE_PWD=${REMOTE_PWD_VAL}

# Parse REMOTE_SSH to separate options from target
SSH_TARGET=$(echo "$REMOTE_SSH_CMD" | awk '{print $NF}')
SSH_OPTS=$(echo "$REMOTE_SSH_CMD" | sed 's/^ssh //' | sed "s/ $SSH_TARGET$//")

echo "Deploy target: $SSH_TARGET"

if [ -z "$REMOTE_PWD" ]; then
    SSH_CMD="ssh $SSH_OPTS"
    SCP_CMD="scp $SSH_OPTS"
else
    SSH_CMD="sshpass -p '$REMOTE_PWD' ssh -o StrictHostKeyChecking=no $SSH_OPTS"
    SCP_CMD="sshpass -p '$REMOTE_PWD' scp -o StrictHostKeyChecking=no $SSH_OPTS"
fi

# 2. Build Linux Binary
echo "Building Linux binary (amd64)..."
rm -rf dist/
mkdir -p dist/butalam/bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/butalam/bin/datagrid-server ./cmd/cursorapp/main.go

# 3. Prepare Package
echo "Packaging assets..."
# Hygiene: Strip deployment-only variables for the production .env
grep -vE "REMOTE_SSH|REMOTE_PWD|DB_SSH_TUNNEL|Deployment" .env > dist/butalam/.env

# Add CATALOG_PATH (os.ReadFile doesn't support globs)
FIRST_CATALOG=$(ls internal/data/catalog/*.json 2>/dev/null | head -1)
if [ -n "$FIRST_CATALOG" ]; then
    echo "" >> dist/butalam/.env
    echo "CATALOG_PATH=$FIRST_CATALOG" >> dist/butalam/.env
fi

cp config.yaml dist/butalam/
cp -r ui dist/butalam/
mkdir -p dist/butalam/internal/data
cp -r internal/data/catalog dist/butalam/internal/data/
# cursorapp expects templates at pkg/datagrid/ui/ (Go module path)
mkdir -p dist/butalam/pkg/datagrid
cp -r ui dist/butalam/pkg/datagrid/ui
cp -r deploy dist/butalam/
mkdir -p dist/butalam/logs

# Create a simple runner script for the remote
cat > dist/butalam/run.sh << 'EOF'
#!/bin/bash
set -a
source .env
set +a
./bin/datagrid-server
EOF
chmod +x dist/butalam/run.sh

# 4. Transfer and Deploy
echo "Transferring package to $SSH_TARGET..."
tar -czf dist/butalam.tar.gz -C dist/butalam .
eval $SCP_CMD dist/butalam.tar.gz "$SSH_TARGET:/tmp/datagrid_deploy.tar.gz"

echo "Executing remote deployment..."
eval $SSH_CMD "$SSH_TARGET" "REMOTE_PWD='$REMOTE_PWD' DB_USER_VAL='$DB_USER_VAL' DB_PASSWORD_VAL='$DB_PASSWORD_VAL' DB_NAME_VAL='$DB_NAME_VAL' bash -s" << 'REMOTE_EOF'
    set -e
    # Authenticate sudo once
    echo "$REMOTE_PWD" | sudo -S true 2>/dev/null

    # Create system user if not exists
    if ! id datagrid &>/dev/null; then
        echo "Creating datagrid system user..."
        sudo useradd -r -s /bin/false datagrid
    fi

    echo "Preparing directories..."
    sudo mkdir -p /opt/datagrid
    sudo chown -R $USER:$USER /opt/datagrid
    
    # Stop existing service (if running)
    sudo systemctl stop datagrid 2>/dev/null || true

    # Extract
    tar -xzf /tmp/datagrid_deploy.tar.gz -C /opt/datagrid
    rm /tmp/datagrid_deploy.tar.gz
    
    cd /opt/datagrid

    echo "Checking remote database and role..."
    
    # Role management
    sudo -u postgres psql -d postgres -c "DO \$BODY\$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER_VAL') THEN CREATE ROLE $DB_USER_VAL WITH LOGIN PASSWORD '$DB_PASSWORD_VAL' SUPERUSER; ELSE ALTER ROLE $DB_USER_VAL WITH PASSWORD '$DB_PASSWORD_VAL' SUPERUSER; END IF; END \$BODY\$;" || true
    
    # Create Database if not exists
    sudo -u postgres psql -d postgres -t -c "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME_VAL'" | grep -q 1 || sudo -u postgres psql -d postgres -c "CREATE DATABASE $DB_NAME_VAL OWNER $DB_USER_VAL;" || true

    # Set ownership for service user
    sudo chown -R datagrid:datagrid /opt/datagrid

    # Install and restart systemd service
    echo "Installing systemd service..."
    sudo cp deploy/datagrid.service /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable datagrid
    sudo systemctl restart datagrid
    
    sleep 2
    sudo systemctl status datagrid --no-pager || true

    echo "✅ Deployment successful at /opt/datagrid"
REMOTE_EOF

# 5. Health Check
PORT=$(grep "port:" config.yaml | head -1 | awk '{print $2}' | tr -d '"')
HEALTH_URL="http://$(echo $SSH_TARGET | cut -d'@' -f2):${PORT:-8085}"
echo "Health check: $HEALTH_URL ..."
curl -I -s --noproxy "*" "$HEALTH_URL" || echo "Warning: Could not connect to $HEALTH_URL (service may still be starting)"

echo "Done!"

# 6. Cleanup (keep dist/ directory, remove contents)
echo "Cleaning up build artifacts..."
rm -rf dist/*
