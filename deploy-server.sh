#!/bin/bash
set -e

SSH="ssh rumahweb"
SCP="scp -P 2223"
PHP=/opt/alt/php84/usr/bin/php
COMPOSER=/home/renr4736/bin/composer
API_DIR=/home/renr4736/public_html/re-travel/api
TRAVEL_DIR=/home/renr4736/public_html/travel

echo "Re-Travel API Deploy"
echo "=================================="

# Step 1 — Pack and upload API (excluding vendor)
echo "Packing API..."
cd "$(dirname "$0")"
tar --exclude='api/vendor' --exclude='api/.env' --exclude='api/public/uploads' --exclude='.git' -czf /tmp/re-travel-api.tar.gz api/
echo "Uploading..."
$SCP /tmp/re-travel-api.tar.gz renr4736@rendrahutama.my.id:~/

# Step 2 — Extract, install deps, run migrations on server
echo "Deploying on server..."
$SSH "
set -e
mkdir -p $API_DIR
cd /home/renr4736/public_html/re-travel
tar --warning=no-unknown-keyword -xzf ~/re-travel-api.tar.gz
cd $API_DIR
$PHP $COMPOSER install --no-dev --optimize-autoloader
$PHP scripts/setup.php

# Fix uploads dir permissions
mkdir -p $API_DIR/public/uploads
chmod 755 $API_DIR/public/uploads

# Ensure uploads symlink exists
if [ ! -L '$TRAVEL_DIR/uploads' ]; then
    ln -s $API_DIR/public/uploads $TRAVEL_DIR/uploads
    echo 'uploads symlink created'
fi

# Ensure API proxy is pointing to re-travel
cat > $TRAVEL_DIR/api/index.php << 'EOF'
<?php
\$_SERVER['SCRIPT_NAME'] = '/index.php';
\$_SERVER['PHP_SELF'] = '/index.php';
require '/home/renr4736/public_html/re-travel/api/public/index.php';
EOF

echo 'Server setup complete!'
"

echo ""
echo "Done! https://travel.rendrahutama.my.id"
