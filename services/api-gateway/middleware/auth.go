package middleware

import (
	"net/http"
	"strings"

	"github.com/bench/api-gateway/config"
)

func TeamAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := bearerToken(r.Header.Get("Authorization")); token == "" {
			WriteAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func AdminAuth(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bearerToken(r.Header.Get("Authorization")) != cfg.AdminToken {
			WriteAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid admin token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
