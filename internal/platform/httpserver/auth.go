package httpserver

import (
	"context"
	"errors"
	"net/http"

	"stock-etsy-service/internal/platform/authclient"
	"stock-etsy-service/internal/platform/response"
)

type currentUserContextKey struct{}

type currentUserProvider interface {
	CurrentUser(ctx context.Context, cookies []*http.Cookie) (*authclient.CurrentUser, error)
}

func requireAuth(provider currentUserProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := provider.CurrentUser(r.Context(), r.Cookies())
			if err != nil {
				if errors.Is(err, authclient.ErrUnauthorized) {
					response.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
					return
				}

				response.WriteError(w, http.StatusBadGateway, "auth_service_unavailable", "auth service unavailable")
				return
			}

			ctx := context.WithValue(r.Context(), currentUserContextKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentUserFromContext(ctx context.Context) (*authclient.CurrentUser, bool) {
	user, ok := ctx.Value(currentUserContextKey{}).(*authclient.CurrentUser)
	return user, ok
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	user, ok := CurrentUserFromContext(ctx)
	if !ok || user.ID == "" {
		return "", false
	}

	return user.ID, true
}
