package auth

import (
	"net/http"
	"strings"
)

const adminToken = "YOUR_ADMIN_TOKEN" // Set this securely in production

func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") || strings.TrimPrefix(token, "Bearer ") != adminToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
