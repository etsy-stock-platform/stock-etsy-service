package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stock-etsy-service/internal/platform/config"
)

func TestProtectedEtsyRouteRequiresAuth(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer authServer.Close()

	server := newTestServer(t, authServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/etsy/auth-check", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestProtectedEtsyRouteStoresUserIDInContext(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("access_token"); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"user": {
				"id": "5a24b84f-5d82-4b9f-8bfa-0400400e5e0a",
				"email": "user@example.com",
				"name": "Test User",
				"email_verified_at": null,
				"created_at": "2026-06-18T10:00:00Z",
				"updated_at": "2026-06-18T10:00:00Z"
			}
		}`))
	}))
	defer authServer.Close()

	server := newTestServer(t, authServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/etsy/auth-check", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-token"})
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.UserID != "5a24b84f-5d82-4b9f-8bfa-0400400e5e0a" {
		t.Fatalf("unexpected user id: %s", body.UserID)
	}
}

func newTestServer(t *testing.T, authServiceURL string) *http.Server {
	t.Helper()

	server, err := New(&config.Config{
		HTTPAddr:       ":0",
		FrontendOrigin: "http://localhost:5173",
		AuthService: config.AuthServiceConfig{
			BaseURL:        authServiceURL,
			RequestTimeout: time.Second,
		},
	}, nil)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	return server
}
