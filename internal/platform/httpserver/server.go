package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"stock-etsy-service/internal/platform/authclient"
	"stock-etsy-service/internal/platform/config"
	"stock-etsy-service/internal/platform/response"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(cfg *config.Config, dbPool *pgxpool.Pool) (*http.Server, error) {
	authClient, err := authclient.New(cfg.AuthService.BaseURL, cfg.AuthService.RequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("create auth client: %w", err)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(cors(cfg.FrontendOrigin))

	router.Get("/health", healthHandler(dbPool))
	router.Route("/etsy", func(r chi.Router) {
		r.Use(requireAuth(authClient))
		r.Get("/auth-check", authCheckHandler)
	})

	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

func healthHandler(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := dbPool.Ping(ctx); err != nil {
			response.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":   "unhealthy",
				"service":  "stock-etsy-service",
				"database": "down",
			})
			return
		}

		response.WriteJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"service":  "stock-etsy-service",
			"database": "up",
		})
	}
}

func authCheckHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusInternalServerError, "current_user_missing", "current user missing")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
	})
}

func cors(frontendOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && origin == frontendOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
