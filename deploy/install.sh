#!/bin/bash
set -e

APP_NAME="datagrid"
INSTALL_DIR="/opt/$APP_NAME"
USER_NAME="datagrid"

echo "Installing $APP_NAME..."

# 1. Create User
if ! id "$USER_NAME" &>/dev/null; then
    echo "Creating user $USER_NAME..."
    sudo useradd -r -s /bin/false $USER_NAME
fi

# 2. Setup Directory
echo "Setting up directories..."
sudo mkdir -p $INSTALL_DIR/ui
sudo mkdir -p $INSTALL_DIR/opt/envs
sudo mkdir -p $INSTALL_DIR/catalog
sudo mkdir -p $INSTALL_DIR/database

# 3. Copy Files
echo "Copying files..."
sudo cp testapp $INSTALL_DIR/
sudo cp config.yaml $INSTALL_DIR/
sudo cp -r ui/static $INSTALL_DIR/ui/
sudo cp -r ui/templates $INSTALL_DIR/ui/
sudo cp -r catalog/*.json $INSTALL_DIR/catalog/
sudo cp -r database/*.sql $INSTALL_DIR/database/
sudo cp -r opt/envs $INSTALL_DIR/opt/
sudo cp scripts/switch_env.sh $INSTALL_DIR/
sudo chmod +x $INSTALL_DIR/switch_env.sh
sudo chmod +x $INSTALL_DIR/testapp

# 4. Permissions
echo "Setting permissions..."
sudo chown -R $USER_NAME:$USER_NAME $INSTALL_DIR

# 5. Service
echo "Installing systemd service..."
sudo cp deploy/datagrid.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable $APP_NAME

echo "Installation complete!"
echo "1. Switch environment: cd $INSTALL_DIR && sudo ./switch_env.sh PROD"
echo "2. Edit config if needed: sudo nano $INSTALL_DIR/.env"
echo "3. Start service: sudo systemctl start $APP_NAME"
