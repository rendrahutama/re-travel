# RE-TRAVEL

RE-TRAVEL is a travel itinerary planner with a React frontend and a PHP Slim + MySQL backend. It helps organize trips, activities, dates, locations, and travel costs in a single timeline view.

## Stack

- Frontend: React + Vite
- Backend: PHP 8.2 + Slim 4
- Database: MySQL (via XAMPP)
- Maps: Leaflet + OpenStreetMap

## Project Structure

- [frontend](./frontend): React application
- [api](./api): PHP Slim 4 API

## Quick Start

### Prerequisites

- XAMPP (Apache + MySQL + PHP 8.2)
- Node.js >= 18 / npm >= 9
- Composer

### 1. Set up the API

```bash
cd api
composer install
cp .env.example .env
```

Create the database in MySQL:

```bash
/Applications/XAMPP/bin/mysql -u root -e "CREATE DATABASE IF NOT EXISTS retravel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

Run migrations and seed the default user:

```bash
/Applications/XAMPP/bin/php scripts/setup.php
```

Set `APP_BASE_PATH` in `api/.env` to match where Apache serves the API from, for example:

```env
APP_BASE_PATH=/re-travel/api/public
```

The API is served by Apache from the `public/` directory. Make sure XAMPP's `DocumentRoot` points to this project's parent directory and `AllowOverride All` is enabled.

### 2. Start the frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

In `frontend/.env`, set the API base URL:

```env
VITE_API_BASE_URL=http://localhost/re-travel/api/public
```

Frontend dev server URL:

```
http://localhost:5173
```

## Features

- User registration and login (session-based auth)
- Public and private itineraries — toggle per itinerary from the Edit page
- Shareable itinerary links — readable without login
- Create, edit, and delete itineraries with cover image upload
- Add, edit, delete, and reorder activities
- Group activities by day
- Auto-derived itinerary end date from activities
- Cost tracking per itinerary
- Interactive maps for activity locations
- AI-assisted activity import via JSON prompt
- SEO-friendly meta tags and OpenGraph tags on itinerary detail pages
- Auto-generated `sitemap.xml` at `/sitemap.xml` listing all public itineraries

## Authentication

Accounts are managed via `/api/auth/register` and `/api/auth/login`. Sessions are stored server-side in MySQL and validated via a Bearer token in the `Authorization` header. Tokens expire after 30 days.

A default demo account is created during setup using the `DEFAULT_USER_EMAIL` and `DEFAULT_USER_PASSWORD` environment variables (defaults: `demo@retravel.local` / `demo-password`).

## Image Uploads

Cover images are uploaded to `api/public/uploads/` and served directly by Apache. The upload endpoint is `POST /api/upload/image` (requires auth). Maximum file size is 5MB; supported formats are JPEG, PNG, WebP, and GIF.
