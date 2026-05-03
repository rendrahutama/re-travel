#!/bin/bash
set -e

SSH="ssh -p 2223 renr4736@rendrahutama.my.id"
SCP="scp -P 2223"

echo "🚀 Re-Itinerary Frontend Deploy"
echo "================================"
echo "1) Build for LOCAL  (http://localhost/re-itinerary/api/public)"
echo "2) Build for PROD   (https://travel.rendrahutama.my.id)"
echo ""
read -p "Choose [1/2]: " choice

if [ "$choice" = "1" ]; then
    API_URL="http://localhost/re-itinerary/api/public"
    SITE_URL="http://localhost:5173"
    echo "🏠 Building for LOCAL..."
elif [ "$choice" = "2" ]; then
    API_URL="https://travel.rendrahutama.my.id"
    SITE_URL="https://travel.rendrahutama.my.id"
    echo "🌐 Building for PROD..."
else
    echo "❌ Invalid choice!"
    exit 1
fi

# Backup current .env, inject prod values, build, restore
cp .env .env.backup
printf "VITE_API_BASE_URL=%s\nVITE_SITE_URL=%s\n" "$API_URL" "$SITE_URL" > .env
npm run build
cp .env.backup .env
echo "✅ Build complete!"

if [ "$choice" = "2" ]; then
    read -p "📤 Upload to server? [y/n]: " upload
    if [ "$upload" = "y" ]; then
        # Upload frontend dist (including dotfiles like .htaccess)
        echo "📁 Uploading frontend..."
        $SCP -r ./dist/. renr4736@rendrahutama.my.id:~/public_html/travel/

        # Upload .env for og.php
        echo "⚙️  Uploading og.php .env..."
        printf "VITE_API_BASE_URL=%s\nVITE_SITE_URL=%s\n" "$API_URL" "$SITE_URL" | \
            $SSH "cat > ~/public_html/travel/.env"

        echo ""
        echo "✅ Deployed to https://travel.rendrahutama.my.id"
    fi
fi
