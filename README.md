# RE-ITINERARY

RE-ITINERARY is a travel itinerary planner with a React frontend and a Go + SQLite backend. It helps organize trips, activities, dates, locations, and travel costs in a single timeline view.

## Stack

- Frontend: React + Vite
- Backend: Go
- Database: SQLite
- Maps: Leaflet + OpenStreetMap

## Project Structure

- [frontend](./frontend): React application
- [api](./api): Go API and SQLite integration
- [api/db/re_itinerary.db](./api/db/re_itinerary.db): SQLite database

## Quick Start

### 1. Start the API

```bash
cd api
go mod tidy
go run .
```

API default URL:

```bash
http://localhost:8080
```

### 2. Start the frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Frontend default URL:

```bash
http://localhost:5173
```

### 3. Configure the frontend API URL

In `frontend/.env`:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

### 4. (Optional) Configure the site URL for sitemap

Set `SITE_URL` in the API environment to your production domain so the generated sitemap uses the correct URLs:

```bash
SITE_URL=https://yourdomain.com go run .
```

## Features

- User registration and login (session-based auth)
- Public and private itineraries — toggle per itinerary from the Edit page
- Shareable itinerary links — readable without login
- Create, edit, and delete itineraries
- Add, edit, delete, and reorder activities
- Group activities by day
- Auto-derived itinerary end date from activities
- Cost tracking per itinerary
- Interactive maps for activity locations
- LocalStorage import into SQLite
- SEO-friendly meta tags and OpenGraph tags on itinerary detail pages
- Auto-generated `sitemap.xml` at `/sitemap.xml` listing all public itineraries

## Authentication

Accounts are managed via `/api/auth/register` and `/api/auth/login`. Sessions are stored server-side in SQLite and validated via a Bearer token in the `Authorization` header. Tokens expire after 30 days.

A default demo account is created on first run using the `DEFAULT_USER_EMAIL` and `DEFAULT_USER_PASSWORD` environment variables (defaults: `demo@reitinerary.local` / `demo-password`).

## Existing Docs

- API documentation: [api/README.md](./api/README.md)
- Frontend notes: [frontend/README.md](./frontend/README.md)
- Environment requirements: [requirements.txt](./requirements.txt)
