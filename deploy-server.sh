#!/bin/bash
set -e

echo "🚀 Re-Itinerary Deployment Script"
echo "=================================="

# Config
PHP=/opt/alt/php84/usr/bin/php
REPO="https://github.com/rendrahutama/re-itinerary.git"
PUBLIC_HTML=~/public_html
API_DIR=$PUBLIC_HTML/re-itinerary/api
TRAVEL_DIR=$PUBLIC_HTML/travel

# Step 1 — Pull latest code
echo "📦 Pulling latest code..."
if [ -d "$PUBLIC_HTML/re-itinerary" ]; then
    cd $PUBLIC_HTML/re-itinerary
    git pull
else
    cd $PUBLIC_HTML
    git clone $REPO
fi

# Step 2 — Restore .env (never in git)
echo "⚙️  Checking .env..."
if [ ! -f "$API_DIR/.env" ]; then
    echo "❌ .env not found! Create it manually:"
    echo "   nano $API_DIR/.env"
    exit 1
fi

# Step 3 — Run migrations
echo "🗄️  Running migrations..."
cd $API_DIR
$PHP scripts/setup.php

# Step 4 — Fix permissions
echo "🔒 Fixing permissions..."
chmod 755 $API_DIR/public/uploads 2>/dev/null || mkdir -p $API_DIR/public/uploads && chmod 755 $API_DIR/public/uploads
chmod 644 $API_DIR/public/uploads/* 2>/dev/null || true

# Step 5 — Ensure uploads symlink exists
echo "🔗 Checking uploads symlink..."
if [ ! -L "$TRAVEL_DIR/uploads" ]; then
    ln -s $API_DIR/public/uploads $TRAVEL_DIR/uploads
    echo "   Symlink created!"
else
    echo "   Symlink OK!"
fi

# Step 6 — Ensure API proxy exists
echo "🔀 Checking API proxy..."
if [ ! -f "$TRAVEL_DIR/api/index.php" ]; then
    mkdir -p $TRAVEL_DIR/api
    cat > $TRAVEL_DIR/api/index.php << 'EOF'
<?php
$_SERVER['SCRIPT_NAME'] = '/index.php';
$_SERVER['PHP_SELF'] = '/index.php';
require '/home/renr4736/public_html/re-itinerary/api/public/index.php';
EOF
    echo "   Proxy created!"
else
    echo "   Proxy OK!"
fi

# Step 7 — Ensure .htaccess exists
echo "📝 Checking .htaccess..."
if [ ! -f "$TRAVEL_DIR/.htaccess" ]; then
    cat > $TRAVEL_DIR/.htaccess << 'EOF'
RewriteEngine On

RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule ^api/(.*)$ api/index.php [QSA,L]

RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule ^(.*)$ index.html [QSA,L]
EOF
    echo "   .htaccess created!"
else
    echo "   .htaccess OK!"
fi

echo ""
echo "✅ Deployment complete!"
echo "🌐 https://travel.rendrahutama.my.id"
