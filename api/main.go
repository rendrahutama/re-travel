package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultPort       = "8080"
	defaultUserName   = "Demo User"
	defaultUserEmail  = "demo@reitinerary.local"
	defaultUserPass   = "demo-password"
	defaultSQLitePath = "db/re_itinerary.db"
)

var supportedActivityTypes = []string{
	"Attraction",
	"Beach",
	"Bus",
	"Car",
	"Culinary",
	"Culture",
	"Cycling",
	"Event",
	"Explore",
	"Ferry",
	"Flight",
	"Hiking",
	"Motorscooter",
	"Nature",
	"Other",
	"Shopping",
	"Spa",
	"Sport",
	"Stay",
	"Taxi",
	"Train",
}

type contextKey string

const userIDKey contextKey = "userID"

func userIDFromCtx(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

type app struct {
	db *sql.DB
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type itinerary struct {
	ID            string     `json:"id"`
	OwnerID       int64      `json:"ownerId"`
	OwnerName     string     `json:"ownerName"`
	Slug          string     `json:"slug"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	StartDate     string     `json:"startDate"`
	EndDate       string     `json:"endDate"`
	Currency      string     `json:"currency"`
	EstimatedCost float64    `json:"estimatedCost"`
	Image         *string    `json:"image"`
	IsPublic      bool       `json:"isPublic"`
	CreatedAt     string     `json:"createdAt"`
	Activities    []activity `json:"activities"`
}

type activity struct {
	ID             string   `json:"id"`
	Datetime       string   `json:"datetime"`
	Type           string   `json:"type"`
	Identification string   `json:"identification"`
	Location       location `json:"location"`
	Cost           float64  `json:"cost"`
	TicketStatus   *string  `json:"ticketStatus"`
	Details        string   `json:"details"`
	SortOrder      int      `json:"sortOrder"`
}

type location struct {
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Lat     *float64 `json:"lat"`
	Lng     *float64 `json:"lng"`
}

type itineraryPayload struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	Currency      string   `json:"currency"`
	EstimatedCost *float64 `json:"estimatedCost"`
	Image         *string  `json:"image"`
	IsPublic      *bool    `json:"isPublic"`
}

type activityPayload struct {
	Datetime       string   `json:"datetime"`
	Type           string   `json:"type"`
	Identification string   `json:"identification"`
	Location       location `json:"location"`
	Cost           *float64 `json:"cost"`
	TicketStatus   *string  `json:"ticketStatus"`
	Details        string   `json:"details"`
	SortOrder      *int     `json:"sortOrder"`
}

type moveActivityPayload struct {
	Direction string `json:"direction"`
}

type importRequest struct {
	ReplaceExisting bool              `json:"replaceExisting"`
	Itineraries     []importItinerary `json:"itineraries"`
}

type importResult struct {
	ImportedCount int         `json:"importedCount"`
	Itineraries   []itinerary `json:"itineraries"`
}

type importItinerary struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	StartDate     string           `json:"startDate"`
	EndDate       string           `json:"endDate"`
	Currency      string           `json:"currency"`
	EstimatedCost float64          `json:"estimatedCost"`
	Image         *string          `json:"image"`
	CreatedAt     string           `json:"createdAt"`
	Activities    []importActivity `json:"activities"`
}

type importActivity struct {
	ID             string   `json:"id"`
	Datetime       string   `json:"datetime"`
	Type           string   `json:"type"`
	Identification string   `json:"identification"`
	Location       location `json:"location"`
	Cost           float64  `json:"cost"`
	TicketStatus   *string  `json:"ticketStatus"`
	Details        string   `json:"details"`
	SortOrder      int      `json:"sortOrder"`
}

// Auth types

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type sitemapURLEntry struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name          `xml:"urlset"`
	Xmlns   string            `xml:"xmlns,attr"`
	URLs    []sitemapURLEntry `xml:"url"`
}

func main() {
	dbPath := envOr("SQLITE_PATH", defaultSQLitePath)
	port := envOr("PORT", defaultPort)

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create db directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("open sqlite database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping sqlite database: %v", err)
	}

	a := &app{db: db}

	if err := a.bootstrap(); err != nil {
		log.Fatalf("bootstrap api: %v", err)
	}

	addr := ":" + port
	log.Printf("reitinerary api listening on %s using %s", addr, dbPath)
	if err := http.ListenAndServe(addr, a.routes()); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

func (a *app) bootstrap() error {
	if _, err := a.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	if err := a.migrateActivityTypesSchema(); err != nil {
		return fmt.Errorf("migrate activity schema: %w", err)
	}

	if err := a.migrateActivityTypesExpanded(); err != nil {
		return fmt.Errorf("migrate activity types expanded: %w", err)
	}

	if err := a.migrateItinerariesIsPublic(); err != nil {
		return fmt.Errorf("migrate itineraries is_public: %w", err)
	}

	if err := a.migrateItinerariesSlug(); err != nil {
		return fmt.Errorf("migrate itineraries slug: %w", err)
	}

	defaultPass := envOr("DEFAULT_USER_PASSWORD", defaultUserPass)
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPass), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash default password: %w", err)
	}
	if _, err := a.db.Exec(`
		INSERT INTO users (name, email, password)
		VALUES (?, ?, ?)
		ON CONFLICT(email) DO NOTHING
	`, envOr("DEFAULT_USER_NAME", defaultUserName), envOr("DEFAULT_USER_EMAIL", defaultUserEmail), string(hash)); err != nil {
		return fmt.Errorf("ensure default user: %w", err)
	}

	if err := a.migratePasswordHashes(); err != nil {
		return fmt.Errorf("migrate password hashes: %w", err)
	}

	return nil
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/sitemap.xml", a.handleSitemap)

	mux.HandleFunc("/api/auth/register", a.handleRegister)
	mux.HandleFunc("/api/auth/login", a.handleLogin)
	mux.HandleFunc("/api/auth/logout", a.handleLogout)
	mux.HandleFunc("/api/auth/me", a.requireAuth(a.handleMe))

	mux.HandleFunc("/api/import/local-storage", a.requireAuth(a.handleLocalStorageImport))

	mux.HandleFunc("/api/itineraries", a.optionalAuth(a.handleItineraries))
	mux.HandleFunc("/api/itineraries/", a.optionalAuth(a.handleItineraryRoutes))
	return withCORS(withJSON(mux))
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Auth middleware ───────────────────────────────────────────────────────────

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func (a *app) validateSession(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, errors.New("no token")
	}
	var userID int64
	var expiresAt string
	err := a.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at FROM sessions WHERE token = ?
	`, token).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(exp) {
		return 0, errors.New("session expired")
	}
	return userID, nil
}

func (a *app) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		userID, err := a.validateSession(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

func (a *app) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			userID, err := a.validateSession(r.Context(), token)
			if err == nil {
				ctx := context.WithValue(r.Context(), userIDKey, userID)
				r = r.WithContext(ctx)
			}
		}
		next(w, r)
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ── Auth handlers ─────────────────────────────────────────────────────────────

func (a *app) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, errValidation("name, email, and password are required"))
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, errValidation("password must be at least 8 characters"))
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	res, err := a.db.ExecContext(r.Context(), `
		INSERT INTO users (name, email, password) VALUES (?, ?, ?)
	`, req.Name, req.Email, string(hash))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, errValidation("email already registered"))
		} else {
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	userID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, userResponse{ID: userID, Name: req.Name, Email: req.Email})
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var userID int64
	var name, passwordHash string
	err := a.db.QueryRowContext(r.Context(), `
		SELECT id, name, password FROM users WHERE email = ?
	`, req.Email).Scan(&userID, &name, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, errValidation("invalid email or password"))
		} else {
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, errValidation("invalid email or password"))
		return
	}

	token, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	if _, err := a.db.ExecContext(r.Context(), `
		INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)
	`, userID, token, expiresAt.Format(time.RFC3339)); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userResponse{ID: userID, Name: name, Email: req.Email},
	})
}

func (a *app) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	token := extractToken(r)
	if token != "" {
		_, _ = a.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE token = ?`, token)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *app) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
		return
	}
	var name, email string
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT name, email FROM users WHERE id = ?
	`, userID).Scan(&name, &email); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse{ID: userID, Name: name, Email: email})
}

func (a *app) handleSitemap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	siteURL := strings.TrimRight(envOr("SITE_URL", "http://localhost:5173"), "/")
	items, err := a.listPublicItineraries(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	urls := []sitemapURLEntry{
		{Loc: siteURL + "/", ChangeFreq: "daily", Priority: "1.0"},
	}
	for _, it := range items {
		lastMod := ""
		if len(it.CreatedAt) >= 10 {
			lastMod = it.CreatedAt[:10]
		}
		urls = append(urls, sitemapURLEntry{
			Loc:        fmt.Sprintf("%s/itinerary/%s", siteURL, it.Slug),
			LastMod:    lastMod,
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})
	}
	data, err := xml.MarshalIndent(sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	w.Write(data)
}

// ── Route handlers ────────────────────────────────────────────────────────────

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"dbPath": envOr("SQLITE_PATH", defaultSQLitePath),
	})
}

func (a *app) handleItineraries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userID, authenticated := userIDFromCtx(r.Context())
		var items []itinerary
		var err error
		if authenticated {
			items, err = a.listItineraries(r.Context(), userID)
		} else {
			items, err = a.listPublicItineraries(r.Context())
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		userID, ok := userIDFromCtx(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
			return
		}
		var payload itineraryPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := a.createItinerary(r.Context(), userID, payload)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w)
	}
}

func (a *app) handleLocalStorageImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	req, err := decodeImportRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := a.importLocalStorageData(r.Context(), req)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *app) handleItineraryRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/itineraries/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	parts := strings.Split(path, "/")
	itineraryID, err := a.resolveItineraryID(r.Context(), parts[0])
	if err != nil {
		writeRepoError(w, err)
		return
	}

	if len(parts) == 1 {
		a.handleSingleItinerary(w, r, itineraryID)
		return
	}

	if parts[1] != "activities" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	if len(parts) == 2 {
		a.handleActivities(w, r, itineraryID)
		return
	}

	activityID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid activity id"})
		return
	}

	if len(parts) == 4 && parts[3] == "move" {
		a.handleMoveActivity(w, r, itineraryID, activityID)
		return
	}

	if len(parts) == 3 {
		a.handleSingleActivity(w, r, itineraryID, activityID)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (a *app) handleSingleItinerary(w http.ResponseWriter, r *http.Request, itineraryID int64) {
	switch r.Method {
	case http.MethodGet:
		item, err := a.getItinerary(r.Context(), itineraryID)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		userID, authenticated := userIDFromCtx(r.Context())
		if !item.IsPublic && (!authenticated || item.OwnerID != userID) {
			writeRepoError(w, errForbidden("itinerary"))
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut, http.MethodPatch:
		if _, ok := userIDFromCtx(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
			return
		}
		var payload itineraryPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := a.updateItinerary(r.Context(), itineraryID, payload)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if _, ok := userIDFromCtx(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
			return
		}
		if err := a.deleteItinerary(r.Context(), itineraryID); err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		writeMethodNotAllowed(w)
	}
}

func (a *app) handleActivities(w http.ResponseWriter, r *http.Request, itineraryID int64) {
	switch r.Method {
	case http.MethodGet:
		item, err := a.getItinerary(r.Context(), itineraryID)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		userID, authenticated := userIDFromCtx(r.Context())
		if !item.IsPublic && (!authenticated || item.OwnerID != userID) {
			writeRepoError(w, errForbidden("itinerary"))
			return
		}
		items, err := a.listActivities(r.Context(), itineraryID)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		if _, ok := userIDFromCtx(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
			return
		}
		var payload activityPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := a.createActivity(r.Context(), itineraryID, payload)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w)
	}
}

func (a *app) handleSingleActivity(w http.ResponseWriter, r *http.Request, itineraryID, activityID int64) {
	switch r.Method {
	case http.MethodGet:
		item, err := a.getActivity(r.Context(), itineraryID, activityID)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut, http.MethodPatch:
		if _, ok := userIDFromCtx(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
			return
		}
		var payload activityPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := a.updateActivity(r.Context(), itineraryID, activityID, payload)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if _, ok := userIDFromCtx(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
			return
		}
		if err := a.deleteActivity(r.Context(), itineraryID, activityID); err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		writeMethodNotAllowed(w)
	}
}

func (a *app) handleMoveActivity(w http.ResponseWriter, r *http.Request, itineraryID, activityID int64) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if _, ok := userIDFromCtx(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, errValidation("unauthorized"))
		return
	}
	var payload moveActivityPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.moveActivity(r.Context(), itineraryID, activityID, payload.Direction)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// ── Database operations ───────────────────────────────────────────────────────

func (a *app) listPublicItineraries(ctx context.Context) ([]itinerary, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id FROM itineraries WHERE is_public = 1 ORDER BY start_date ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []itinerary{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		item, err := a.getItinerary(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *app) listItineraries(ctx context.Context, userID int64) ([]itinerary, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id
		FROM itineraries
		WHERE user_id = ?
		ORDER BY start_date ASC, id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []itinerary{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		item, err := a.getItinerary(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *app) getItinerary(ctx context.Context, itineraryID int64) (itinerary, error) {
	var item itinerary
	var image sql.NullString
	var isPublic int

	err := a.db.QueryRowContext(ctx, `
		SELECT i.id, i.user_id, i.name, COALESCE(i.description, ''), i.start_date,
		       COALESCE(
		           (
		               SELECT CASE
		                   WHEN MAX(a.activity_date) IS NULL OR MAX(a.activity_date) < i.start_date
		                       THEN i.start_date
		                   ELSE MAX(a.activity_date)
		               END
		               FROM activities a
		               WHERE a.itinerary_id = i.id
		           ),
		           i.start_date
		       ) AS derived_end_date,
		       COALESCE(i.currency, 'IDR'),
		       COALESCE((SELECT SUM(a.cost) FROM activities a WHERE a.itinerary_id = i.id), 0),
		       cover_image_url, created_at, COALESCE(i.is_public, 0),
		       (SELECT COALESCE(name, 'Unknown') FROM users WHERE id = i.user_id),
		       COALESCE(i.slug, '')
		FROM itineraries i
		WHERE i.id = ?
	`, itineraryID).Scan(
		&item.ID,
		&item.OwnerID,
		&item.Name,
		&item.Description,
		&item.StartDate,
		&item.EndDate,
		&item.Currency,
		&item.EstimatedCost,
		&image,
		&item.CreatedAt,
		&isPublic,
		&item.OwnerName,
		&item.Slug,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return itinerary{}, errNotFound("itinerary")
		}
		return itinerary{}, err
	}

	item.IsPublic = isPublic == 1
	if image.Valid && image.String != "" {
		item.Image = &image.String
	}
	item.StartDate = normalizeDateValue(item.StartDate)
	item.EndDate = normalizeDateValue(item.EndDate)

	activities, err := a.listActivities(ctx, itineraryID)
	if err != nil {
		return itinerary{}, err
	}
	item.Activities = activities
	return item, nil
}

func (a *app) createItinerary(ctx context.Context, userID int64, payload itineraryPayload) (itinerary, error) {
	payload = normalizeItineraryPayload(payload)
	if err := validateItineraryPayload(payload); err != nil {
		return itinerary{}, err
	}

	isPublic := payload.IsPublic != nil && *payload.IsPublic

	slug, err := a.generateUniqueSlug(ctx, toSlug(payload.Name))
	if err != nil {
		return itinerary{}, err
	}

	res, err := a.db.ExecContext(ctx, `
		INSERT INTO itineraries (
			user_id, name, description, start_date, end_date, currency, cover_image_url, estimated_cost, is_public, slug
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, payload.Name, payload.Description, payload.StartDate, payload.StartDate, payload.Currency, nullableString(payload.Image), 0, boolToInt(isPublic), slug)
	if err != nil {
		return itinerary{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return itinerary{}, err
	}

	return a.getItinerary(ctx, id)
}

func (a *app) updateItinerary(ctx context.Context, itineraryID int64, payload itineraryPayload) (itinerary, error) {
	if err := a.checkItineraryOwnership(ctx, itineraryID); err != nil {
		return itinerary{}, err
	}

	current, err := a.getItinerary(ctx, itineraryID)
	if err != nil {
		return itinerary{}, err
	}

	payload = mergeItineraryPayload(current, payload)
	payload = normalizeItineraryPayload(payload)
	if err := validateItineraryPayload(payload); err != nil {
		return itinerary{}, err
	}

	isPublic := boolToInt(*payload.IsPublic)

	_, err = a.db.ExecContext(ctx, `
		UPDATE itineraries
		SET name = ?, description = ?, start_date = ?, end_date = ?, currency = ?,
		    cover_image_url = ?, estimated_cost = ?, is_public = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, payload.Name, payload.Description, payload.StartDate, payload.StartDate, payload.Currency, nullableString(payload.Image), 0, isPublic, itineraryID)
	if err != nil {
		return itinerary{}, err
	}

	if err := a.syncItineraryDerivedFields(ctx, a.db, itineraryID); err != nil {
		return itinerary{}, err
	}

	return a.getItinerary(ctx, itineraryID)
}

func (a *app) deleteItinerary(ctx context.Context, itineraryID int64) error {
	if err := a.checkItineraryOwnership(ctx, itineraryID); err != nil {
		return err
	}

	res, err := a.db.ExecContext(ctx, `
		DELETE FROM itineraries WHERE id = ?
	`, itineraryID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound("itinerary")
	}
	return nil
}

func (a *app) listActivities(ctx context.Context, itineraryID int64) ([]activity, error) {
	if !a.itineraryExists(ctx, itineraryID) {
		return nil, errNotFound("itinerary")
	}

	rows, err := a.db.QueryContext(ctx, `
		SELECT id, activity_date, start_time, activity_type, COALESCE(identifier, ''),
		       COALESCE(location_name, ''), COALESCE(location_address, ''),
		       latitude, longitude, COALESCE(cost, 0), ticket_status,
		       COALESCE(details, ''), COALESCE(sort_order, 0)
		FROM activities
		WHERE itinerary_id = ?
		ORDER BY sort_order ASC, activity_date ASC, start_time ASC, id ASC
	`, itineraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []activity{}
	for rows.Next() {
		item, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *app) getActivity(ctx context.Context, itineraryID, activityID int64) (activity, error) {
	if !a.itineraryExists(ctx, itineraryID) {
		return activity{}, errNotFound("itinerary")
	}

	row := a.db.QueryRowContext(ctx, `
		SELECT id, activity_date, start_time, activity_type, COALESCE(identifier, ''),
		       COALESCE(location_name, ''), COALESCE(location_address, ''),
		       latitude, longitude, COALESCE(cost, 0), ticket_status,
		       COALESCE(details, ''), COALESCE(sort_order, 0)
		FROM activities
		WHERE itinerary_id = ? AND id = ?
	`, itineraryID, activityID)

	item, err := scanActivity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return activity{}, errNotFound("activity")
		}
		return activity{}, err
	}
	return item, nil
}

func (a *app) createActivity(ctx context.Context, itineraryID int64, payload activityPayload) (activity, error) {
	if err := a.checkItineraryOwnership(ctx, itineraryID); err != nil {
		return activity{}, err
	}

	payload = normalizeActivityPayload(payload)
	if err := validateActivityPayload(payload); err != nil {
		return activity{}, err
	}

	datePart, timePart, err := splitDatetime(payload.Datetime)
	if err != nil {
		return activity{}, err
	}

	sortOrder := payload.SortOrder
	if sortOrder == nil {
		nextSort, err := a.nextSortOrder(ctx, itineraryID)
		if err != nil {
			return activity{}, err
		}
		sortOrder = &nextSort
	}

	res, err := a.db.ExecContext(ctx, `
		INSERT INTO activities (
			itinerary_id, activity_type, identifier, location_name, location_address,
			latitude, longitude, activity_date, start_time, cost, ticket_status, details, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, itineraryID, payload.Type, payload.Identification, payload.Location.Name, payload.Location.Address, payload.Location.Lat, payload.Location.Lng, datePart, timePart, valueOrZero(payload.Cost), nullableString(payload.TicketStatus), payload.Details, *sortOrder)
	if err != nil {
		return activity{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return activity{}, err
	}

	if err := a.syncItineraryDerivedFields(ctx, a.db, itineraryID); err != nil {
		return activity{}, err
	}

	return a.getActivity(ctx, itineraryID, id)
}

func (a *app) updateActivity(ctx context.Context, itineraryID, activityID int64, payload activityPayload) (activity, error) {
	if err := a.checkItineraryOwnership(ctx, itineraryID); err != nil {
		return activity{}, err
	}

	current, err := a.getActivity(ctx, itineraryID, activityID)
	if err != nil {
		return activity{}, err
	}

	payload = mergeActivityPayload(current, payload)
	payload = normalizeActivityPayload(payload)
	if err := validateActivityPayload(payload); err != nil {
		return activity{}, err
	}

	datePart, timePart, err := splitDatetime(payload.Datetime)
	if err != nil {
		return activity{}, err
	}

	sortOrder := current.SortOrder
	if payload.SortOrder != nil {
		sortOrder = *payload.SortOrder
	}

	_, err = a.db.ExecContext(ctx, `
		UPDATE activities
		SET activity_type = ?, identifier = ?, location_name = ?, location_address = ?,
		    latitude = ?, longitude = ?, activity_date = ?, start_time = ?, cost = ?,
		    ticket_status = ?, details = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE itinerary_id = ? AND id = ?
	`, payload.Type, payload.Identification, payload.Location.Name, payload.Location.Address, payload.Location.Lat, payload.Location.Lng, datePart, timePart, valueOrZero(payload.Cost), nullableString(payload.TicketStatus), payload.Details, sortOrder, itineraryID, activityID)
	if err != nil {
		return activity{}, err
	}

	if err := a.syncItineraryDerivedFields(ctx, a.db, itineraryID); err != nil {
		return activity{}, err
	}

	return a.getActivity(ctx, itineraryID, activityID)
}

func (a *app) deleteActivity(ctx context.Context, itineraryID, activityID int64) error {
	if err := a.checkItineraryOwnership(ctx, itineraryID); err != nil {
		return err
	}

	res, err := a.db.ExecContext(ctx, `
		DELETE FROM activities
		WHERE itinerary_id = ? AND id = ?
	`, itineraryID, activityID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound("activity")
	}

	if err := a.syncItineraryDerivedFields(ctx, a.db, itineraryID); err != nil {
		return err
	}
	return nil
}

func (a *app) moveActivity(ctx context.Context, itineraryID, activityID int64, direction string) (activity, error) {
	if err := a.checkItineraryOwnership(ctx, itineraryID); err != nil {
		return activity{}, err
	}

	items, err := a.listActivities(ctx, itineraryID)
	if err != nil {
		return activity{}, err
	}

	index := -1
	for i := range items {
		if items[i].ID == strconv.FormatInt(activityID, 10) {
			index = i
			break
		}
	}
	if index == -1 {
		return activity{}, errNotFound("activity")
	}

	targetIndex := index
	switch direction {
	case "up":
		targetIndex = index - 1
	case "down":
		targetIndex = index + 1
	default:
		return activity{}, errValidation("direction must be either 'up' or 'down'")
	}

	if targetIndex < 0 || targetIndex >= len(items) {
		return a.getActivity(ctx, itineraryID, activityID)
	}

	currentSort := items[index].SortOrder
	swapSort := items[targetIndex].SortOrder

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return activity{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE activities SET sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND itinerary_id = ?
	`, swapSort, activityID, itineraryID); err != nil {
		return activity{}, err
	}

	swapID, _ := strconv.ParseInt(items[targetIndex].ID, 10, 64)
	if _, err := tx.ExecContext(ctx, `
		UPDATE activities SET sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND itinerary_id = ?
	`, currentSort, swapID, itineraryID); err != nil {
		return activity{}, err
	}

	if err := tx.Commit(); err != nil {
		return activity{}, err
	}

	return a.getActivity(ctx, itineraryID, activityID)
}

func (a *app) importLocalStorageData(ctx context.Context, req importRequest) (importResult, error) {
	userID, ok := userIDFromCtx(ctx)
	if !ok {
		return importResult{}, errValidation("authentication required")
	}

	if len(req.Itineraries) == 0 {
		return importResult{}, errValidation("itineraries is required and must not be empty")
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return importResult{}, err
	}
	defer tx.Rollback()

	if req.ReplaceExisting {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM itineraries WHERE user_id = ?
		`, userID); err != nil {
			return importResult{}, err
		}
	}

	importedIDs := make([]int64, 0, len(req.Itineraries))
	batchSlugs := make(map[string]bool)
	for _, source := range req.Itineraries {
		payload := normalizeItineraryPayload(itineraryPayload{
			Name:          source.Name,
			Description:   source.Description,
			StartDate:     source.StartDate,
			EndDate:       source.EndDate,
			Currency:      source.Currency,
			EstimatedCost: floatPtr(source.EstimatedCost),
			Image:         source.Image,
		})
		if err := validateItineraryPayload(payload); err != nil {
			return importResult{}, fmt.Errorf("itinerary %q: %w", source.Name, err)
		}

		createdAt := normalizeCreatedAt(source.CreatedAt)

		base := toSlug(payload.Name)
		importSlug := base
		if importSlug == "" {
			importSlug = "itinerary"
		}
		for suffix := 2; ; suffix++ {
			if !batchSlugs[importSlug] {
				var count int
				if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM itineraries WHERE slug = ?`, importSlug).Scan(&count); err != nil {
					return importResult{}, err
				}
				if count == 0 {
					break
				}
			}
			importSlug = fmt.Sprintf("%s-%d", base, suffix)
		}
		batchSlugs[importSlug] = true

		res, err := tx.ExecContext(ctx, `
			INSERT INTO itineraries (
				user_id, name, description, start_date, end_date, currency,
				cover_image_url, estimated_cost, is_public, slug, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
		`, userID, payload.Name, payload.Description, payload.StartDate, payload.EndDate, payload.Currency, nullableString(payload.Image), valueOrZero(payload.EstimatedCost), importSlug, createdAt, createdAt)
		if err != nil {
			return importResult{}, err
		}

		itineraryID, err := res.LastInsertId()
		if err != nil {
			return importResult{}, err
		}
		importedIDs = append(importedIDs, itineraryID)

		for _, sourceActivity := range source.Activities {
			ap := normalizeActivityPayload(activityPayload{
				Datetime:       sourceActivity.Datetime,
				Type:           sourceActivity.Type,
				Identification: sourceActivity.Identification,
				Location:       sourceActivity.Location,
				Cost:           floatPtr(sourceActivity.Cost),
				TicketStatus:   sourceActivity.TicketStatus,
				Details:        sourceActivity.Details,
				SortOrder:      intPtr(sourceActivity.SortOrder),
			})
			if err := validateActivityPayload(ap); err != nil {
				return importResult{}, fmt.Errorf("activity %q in itinerary %q: %w", sourceActivity.Identification, source.Name, err)
			}

			datePart, timePart, err := splitDatetime(ap.Datetime)
			if err != nil {
				return importResult{}, err
			}

			if _, err := tx.ExecContext(ctx, `
				INSERT INTO activities (
					itinerary_id, activity_type, identifier, location_name, location_address,
					latitude, longitude, activity_date, start_time, cost, ticket_status,
					details, sort_order, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, itineraryID, ap.Type, ap.Identification, ap.Location.Name, ap.Location.Address, ap.Location.Lat, ap.Location.Lng, datePart, timePart, valueOrZero(ap.Cost), nullableString(ap.TicketStatus), ap.Details, *ap.SortOrder, createdAt, createdAt); err != nil {
				return importResult{}, err
			}
		}

		if err := a.syncItineraryDerivedFields(ctx, tx, itineraryID); err != nil {
			return importResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return importResult{}, err
	}

	items := make([]itinerary, 0, len(importedIDs))
	for _, id := range importedIDs {
		item, err := a.getItinerary(ctx, id)
		if err != nil {
			return importResult{}, err
		}
		items = append(items, item)
	}

	return importResult{
		ImportedCount: len(items),
		Itineraries:   items,
	}, nil
}

func (a *app) nextSortOrder(ctx context.Context, itineraryID int64) (int, error) {
	var next int
	if err := a.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(sort_order), -1) + 1
		FROM activities
		WHERE itinerary_id = ?
	`, itineraryID).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func (a *app) itineraryExists(ctx context.Context, itineraryID int64) bool {
	var exists int
	err := a.db.QueryRowContext(ctx, `
		SELECT 1 FROM itineraries WHERE id = ?
	`, itineraryID).Scan(&exists)
	return err == nil && exists == 1
}

func (a *app) checkItineraryOwnership(ctx context.Context, itineraryID int64) error {
	userID, ok := userIDFromCtx(ctx)
	if !ok {
		return errValidation("authentication required")
	}
	var ownerID int64
	err := a.db.QueryRowContext(ctx, `SELECT user_id FROM itineraries WHERE id = ?`, itineraryID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound("itinerary")
		}
		return err
	}
	if ownerID != userID {
		return errForbidden("itinerary")
	}
	return nil
}

// ── Scanning ──────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanActivity(s scanner) (activity, error) {
	var item activity
	var datePart string
	var timePart string
	var lat sql.NullFloat64
	var lng sql.NullFloat64
	var ticketStatus sql.NullString

	err := s.Scan(
		&item.ID,
		&datePart,
		&timePart,
		&item.Type,
		&item.Identification,
		&item.Location.Name,
		&item.Location.Address,
		&lat,
		&lng,
		&item.Cost,
		&ticketStatus,
		&item.Details,
		&item.SortOrder,
	)
	if err != nil {
		return activity{}, err
	}

	item.Datetime = strings.TrimSpace(normalizeDateValue(datePart) + "T" + normalizeTime(timePart))
	if lat.Valid {
		item.Location.Lat = &lat.Float64
	}
	if lng.Valid {
		item.Location.Lng = &lng.Float64
	}
	if ticketStatus.Valid && ticketStatus.String != "" {
		item.TicketStatus = &ticketStatus.String
	}

	return item, nil
}

// ── Payload helpers ───────────────────────────────────────────────────────────

func normalizeItineraryPayload(payload itineraryPayload) itineraryPayload {
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Description = strings.TrimSpace(payload.Description)
	payload.StartDate = strings.TrimSpace(payload.StartDate)
	payload.EndDate = strings.TrimSpace(payload.EndDate)
	payload.Currency = strings.ToUpper(strings.TrimSpace(payload.Currency))
	if payload.Currency == "" {
		payload.Currency = "IDR"
	}
	if payload.Image != nil {
		trimmed := strings.TrimSpace(*payload.Image)
		if trimmed == "" {
			payload.Image = nil
		} else {
			payload.Image = &trimmed
		}
	}
	return payload
}

func mergeItineraryPayload(current itinerary, payload itineraryPayload) itineraryPayload {
	if payload.Name == "" {
		payload.Name = current.Name
	}
	if payload.Description == "" {
		payload.Description = current.Description
	}
	if payload.StartDate == "" {
		payload.StartDate = current.StartDate
	}
	if payload.Currency == "" {
		payload.Currency = current.Currency
	}
	if payload.Image == nil && current.Image != nil {
		payload.Image = current.Image
	}
	if payload.EstimatedCost == nil {
		payload.EstimatedCost = &current.EstimatedCost
	}
	if payload.IsPublic == nil {
		payload.IsPublic = &current.IsPublic
	}
	return payload
}

func validateItineraryPayload(payload itineraryPayload) error {
	if payload.Name == "" {
		return errValidation("name is required")
	}
	if payload.StartDate == "" {
		return errValidation("startDate is required")
	}
	if _, err := time.Parse("2006-01-02", payload.StartDate); err != nil {
		return errValidation("startDate must use YYYY-MM-DD")
	}
	return nil
}

func normalizeActivityPayload(payload activityPayload) activityPayload {
	payload.Datetime = strings.TrimSpace(payload.Datetime)
	payload.Type = strings.TrimSpace(payload.Type)
	payload.Identification = strings.TrimSpace(payload.Identification)
	payload.Location.Name = strings.TrimSpace(payload.Location.Name)
	payload.Location.Address = strings.TrimSpace(payload.Location.Address)
	payload.Details = strings.TrimSpace(payload.Details)
	if payload.Type == "" {
		payload.Type = "Other"
	}
	if payload.TicketStatus != nil {
		trimmed := strings.TrimSpace(*payload.TicketStatus)
		if trimmed == "" {
			payload.TicketStatus = nil
		} else {
			payload.TicketStatus = &trimmed
		}
	}
	return payload
}

func mergeActivityPayload(current activity, payload activityPayload) activityPayload {
	if payload.Datetime == "" {
		payload.Datetime = current.Datetime
	}
	if payload.Type == "" {
		payload.Type = current.Type
	}
	if payload.Identification == "" {
		payload.Identification = current.Identification
	}
	if payload.Location == (location{}) {
		payload.Location = current.Location
	}
	if payload.Cost == nil {
		payload.Cost = &current.Cost
	}
	if payload.TicketStatus == nil && current.TicketStatus != nil {
		payload.TicketStatus = current.TicketStatus
	}
	if payload.Details == "" {
		payload.Details = current.Details
	}
	if payload.SortOrder == nil {
		payload.SortOrder = &current.SortOrder
	}
	return payload
}

func validateActivityPayload(payload activityPayload) error {
	if payload.Datetime == "" {
		return errValidation("datetime is required")
	}
	if _, _, err := splitDatetime(payload.Datetime); err != nil {
		return err
	}
	if payload.Type == "" {
		return errValidation("type is required")
	}
	if !isSupportedActivityType(payload.Type) {
		return errValidation("type is not supported")
	}
	return nil
}

func isSupportedActivityType(value string) bool {
	for _, item := range supportedActivityTypes {
		if item == value {
			return true
		}
	}
	return false
}

// ── String / date helpers ─────────────────────────────────────────────────────

func splitDatetime(value string) (string, string, error) {
	parsed, err := time.Parse("2006-01-02T15:04", value)
	if err != nil {
		return "", "", errValidation("datetime must use YYYY-MM-DDTHH:MM")
	}
	return parsed.Format("2006-01-02"), parsed.Format("15:04:05"), nil
}

func normalizeTime(value string) string {
	if len(value) >= 5 {
		return value[:5]
	}
	return value
}

func normalizeDateValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func normalizeCreatedAt(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func (a *app) syncItineraryDerivedFields(ctx context.Context, db execer, itineraryID int64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE itineraries
		SET end_date = COALESCE(
		        (
		            SELECT CASE
		                WHEN MAX(a.activity_date) IS NULL OR MAX(a.activity_date) < itineraries.start_date
		                    THEN itineraries.start_date
		                ELSE MAX(a.activity_date)
		            END
		            FROM activities a
		            WHERE a.itinerary_id = itineraries.id
		        ),
		        start_date
		    ),
		    estimated_cost = COALESCE(
		        (SELECT SUM(a.cost) FROM activities a WHERE a.itinerary_id = itineraries.id),
		        0
		    ),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, itineraryID)
	return err
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}
	return nil
}

func decodeImportRequest(r *http.Request) (importRequest, error) {
	defer r.Body.Close()

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return importRequest{}, fmt.Errorf("read import payload: %w", err)
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return importRequest{}, errValidation("import payload is empty")
	}

	if strings.HasPrefix(trimmed, "[") {
		var itineraries []importItinerary
		if err := json.Unmarshal(raw, &itineraries); err != nil {
			return importRequest{}, fmt.Errorf("invalid import payload: %w", err)
		}
		return importRequest{
			ReplaceExisting: true,
			Itineraries:     itineraries,
		}, nil
	}

	var req importRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return importRequest{}, fmt.Errorf("invalid import payload: %w", err)
	}
	if len(req.Itineraries) == 0 {
		return importRequest{}, errValidation("itineraries is required and must not be empty")
	}
	return req, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNotFound("")):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, errValidation("")):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, errForbidden("")):
		writeError(w, http.StatusForbidden, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

// ── Error types ───────────────────────────────────────────────────────────────

type repoError struct {
	kind string
	msg  string
}

func (e repoError) Error() string {
	return e.msg
}

func (e repoError) Is(target error) bool {
	t, ok := target.(repoError)
	if !ok {
		return false
	}
	if t.kind == "" {
		return e.kind == t.kind || e.kind == "not_found" || e.kind == "validation"
	}
	return e.kind == t.kind
}

func errNotFound(resource string) error {
	return repoError{kind: "not_found", msg: resource + " not found"}
}

func errValidation(message string) error {
	return repoError{kind: "validation", msg: message}
}

func errForbidden(resource string) error {
	return repoError{kind: "forbidden", msg: "access denied"}
}

// ── Value helpers ─────────────────────────────────────────────────────────────

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// ── Slug helpers ─────────────────────────────────────────────────────────────

func toSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	prevHyphen := true
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func (a *app) generateUniqueSlug(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "itinerary"
	}
	slug := base
	for i := 2; ; i++ {
		var count int
		if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM itineraries WHERE slug = ?`, slug).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (a *app) resolveItineraryID(ctx context.Context, segment string) (int64, error) {
	if id, err := strconv.ParseInt(segment, 10, 64); err == nil {
		return id, nil
	}
	var id int64
	err := a.db.QueryRowContext(ctx, `SELECT id FROM itineraries WHERE slug = ?`, segment).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNotFound("itinerary")
	}
	return id, err
}

// ── Migrations ────────────────────────────────────────────────────────────────

func (a *app) migrateItinerariesIsPublic() error {
	var cols string
	if err := a.db.QueryRow(`
		SELECT COALESCE(sql, '') FROM sqlite_master WHERE type='table' AND name='itineraries'
	`).Scan(&cols); err != nil {
		return err
	}
	if strings.Contains(cols, "is_public") {
		return nil
	}
	_, err := a.db.Exec(`ALTER TABLE itineraries ADD COLUMN is_public INTEGER DEFAULT 0 NOT NULL`)
	return err
}

func (a *app) migratePasswordHashes() error {
	rows, err := a.db.Query(`SELECT id, password FROM users`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type userRow struct {
		id       int64
		password string
	}
	var users []userRow
	for rows.Next() {
		var u userRow
		if err := rows.Scan(&u.id, &u.password); err != nil {
			return err
		}
		users = append(users, u)
	}

	for _, u := range users {
		if strings.HasPrefix(u.password, "$2a$") || strings.HasPrefix(u.password, "$2b$") {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := a.db.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hash), u.id); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) migrateItinerariesSlug() error {
	var cols string
	if err := a.db.QueryRow(`
		SELECT COALESCE(sql, '') FROM sqlite_master WHERE type='table' AND name='itineraries'
	`).Scan(&cols); err != nil {
		return err
	}
	if !strings.Contains(cols, "slug") {
		if _, err := a.db.Exec(`ALTER TABLE itineraries ADD COLUMN slug TEXT`); err != nil {
			return err
		}
	}

	// Backfill before creating the unique index so NULLs don't conflict
	rows, err := a.db.Query(`SELECT id, name FROM itineraries WHERE slug IS NULL OR slug = ''`)
	if err != nil {
		return err
	}
	type rowData struct {
		id   int64
		name string
	}
	var pending []rowData
	for rows.Next() {
		var r rowData
		if err := rows.Scan(&r.id, &r.name); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range pending {
		base := toSlug(r.name)
		if base == "" {
			base = fmt.Sprintf("itinerary-%d", r.id)
		}
		slug, err := a.generateUniqueSlug(context.Background(), base)
		if err != nil {
			return err
		}
		if _, err := a.db.Exec(`UPDATE itineraries SET slug = ? WHERE id = ?`, slug, r.id); err != nil {
			return err
		}
	}

	if _, err := a.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_itineraries_slug ON itineraries(slug)`); err != nil {
		return err
	}
	return nil
}

func (a *app) migrateActivityTypesExpanded() error {
	var createSQL string
	err := a.db.QueryRow(`
		SELECT COALESCE(sql, '')
		FROM sqlite_master
		WHERE type = 'table' AND name = 'activities'
	`).Scan(&createSQL)
	if err != nil {
		return err
	}
	if createSQL == "" || strings.Contains(createSQL, "'Hiking'") {
		return nil
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DROP INDEX IF EXISTS idx_activities_itinerary_id`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP INDEX IF EXISTS idx_activities_date`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE activities RENAME TO activities_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(activityTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO activities (
			id, itinerary_id, activity_type, identifier, name, location_name,
			location_address, latitude, longitude, activity_date, start_time,
			cost, ticket_status, details, sort_order, created_at, updated_at
		)
		SELECT
			id, itinerary_id, activity_type, identifier, name, location_name,
			location_address, latitude, longitude, activity_date, start_time,
			cost, ticket_status, details, sort_order, created_at, updated_at
		FROM activities_legacy
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE activities_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_activities_itinerary_id ON activities(itinerary_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_activities_date ON activities(itinerary_id, activity_date, start_time)`); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *app) migrateActivityTypesSchema() error {
	var createSQL string
	err := a.db.QueryRow(`
		SELECT COALESCE(sql, '')
		FROM sqlite_master
		WHERE type = 'table' AND name = 'activities'
	`).Scan(&createSQL)
	if err != nil {
		return err
	}

	if createSQL == "" || strings.Contains(createSQL, "'Nature'") {
		return nil
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DROP INDEX IF EXISTS idx_activities_itinerary_id`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP INDEX IF EXISTS idx_activities_date`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE activities RENAME TO activities_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(activityTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO activities (
			id, itinerary_id, activity_type, identifier, name, location_name,
			location_address, latitude, longitude, activity_date, start_time,
			cost, ticket_status, details, sort_order, created_at, updated_at
		)
		SELECT
			id, itinerary_id, activity_type, identifier, name, location_name,
			location_address, latitude, longitude, activity_date, start_time,
			cost, ticket_status, details, sort_order, created_at, updated_at
		FROM activities_legacy
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE activities_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_activities_itinerary_id ON activities(itinerary_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_activities_date ON activities(itinerary_id, activity_date, start_time)`); err != nil {
		return err
	}

	return tx.Commit()
}

// ── Schema ────────────────────────────────────────────────────────────────────

const schemaSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    email       TEXT UNIQUE NOT NULL,
    password    TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    token      TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS itineraries (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    slug            TEXT UNIQUE,
    name            TEXT NOT NULL,
    description     TEXT,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    currency        TEXT DEFAULT 'IDR',
    cover_image_url TEXT,
    estimated_cost  REAL DEFAULT 0,
    is_public       INTEGER DEFAULT 0 NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

` + activityTableSQL + `

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_itineraries_user_id ON itineraries(user_id);
CREATE INDEX IF NOT EXISTS idx_activities_itinerary_id ON activities(itinerary_id);
CREATE INDEX IF NOT EXISTS idx_activities_date ON activities(itinerary_id, activity_date, start_time);
`

const activityTableSQL = `
CREATE TABLE IF NOT EXISTS activities (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    itinerary_id     INTEGER NOT NULL,
    activity_type    TEXT NOT NULL CHECK(activity_type IN (
                         'Attraction','Beach','Bus','Car','Culinary',
                         'Culture','Cycling','Event','Explore','Ferry',
                         'Flight','Hiking','Motorscooter','Nature','Other',
                         'Shopping','Spa','Sport','Stay','Taxi','Train'
                     )),
    identifier       TEXT,
    name             TEXT,
    location_name    TEXT,
    location_address TEXT,
    latitude         REAL,
    longitude        REAL,
    activity_date    DATE NOT NULL,
    start_time       TIME NOT NULL,
    cost             REAL DEFAULT 0,
    ticket_status    TEXT DEFAULT 'Unbooked' CHECK(ticket_status IN (
                         'Secured','Unbooked','Go Show'
                     )),
    details          TEXT,
    sort_order       INTEGER DEFAULT 0,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (itinerary_id) REFERENCES itineraries(id) ON DELETE CASCADE
);
`
