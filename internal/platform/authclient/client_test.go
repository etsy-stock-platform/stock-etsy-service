package authclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCurrentUserForwardsCookiesAndDecodesUser(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		cookie, err := r.Cookie("access_token")
		if err != nil {
			t.Fatalf("missing forwarded access token cookie: %v", err)
		}

		if cookie.Value != "valid-token" {
			t.Fatalf("unexpected cookie value: %s", cookie.Value)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"user": {
				"id": "2b989df6-5c67-4c0b-810d-fdbf3557e125",
				"email": "user@example.com",
				"name": "Test User",
				"email_verified_at": null,
				"created_at": "2026-06-18T10:00:00Z",
				"updated_at": "2026-06-18T10:00:00Z"
			}
		}`))
	}))
	defer authServer.Close()

	client, err := New(authServer.URL, time.Second)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	user, err := client.CurrentUser(context.Background(), []*http.Cookie{
		{Name: "access_token", Value: "valid-token"},
	})
	if err != nil {
		t.Fatalf("current user: %v", err)
	}

	if user.ID != "2b989df6-5c67-4c0b-810d-fdbf3557e125" {
		t.Fatalf("unexpected user id: %s", user.ID)
	}
}

func TestCurrentUserReturnsUnauthorized(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer authServer.Close()

	client, err := New(authServer.URL, time.Second)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.CurrentUser(context.Background(), nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
