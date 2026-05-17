package middleware

import (
	"context"
	"net/http"

	"github.com/marcosalvi-01/gowatch/internal/account"
	"github.com/marcosalvi-01/gowatch/internal/common"
)

type authService interface {
	GetSession(ctx context.Context, sessionID string) (*account.Session, error)
	GetUserByID(ctx context.Context, userID int64) (*account.User, error)
}

func AuthMiddleware(authService authService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			session, err := authService.GetSession(r.Context(), cookie.Value)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			user, err := authService.GetUserByID(r.Context(), session.UserID)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), common.UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
