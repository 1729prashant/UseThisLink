package mw

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey string

const SessionKey ctxKey = "sessionID"

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const cookieName = "UTL_SESSION"
		c, err := r.Cookie(cookieName)
		var sid string
		if err == nil && c.Value != "" {
			sid = c.Value
		} else {
			sid = uuid.NewString()
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    sid,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   60 * 60 * 24 * 365, // 1 year
				SameSite: http.SameSiteLaxMode,
			})
		}
		ctx := context.WithValue(r.Context(), SessionKey, sid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
