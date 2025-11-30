package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Ensure CSRF cookie exists
		cookie, err := r.Cookie("csrf_token")
		var token string

		if err != nil || cookie.Value == "" {
			token, err = generateCSRFToken()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    token,
				Path:     "/",
				HttpOnly: false, // Must be readable by JS
				SameSite: http.SameSiteStrictMode,
				Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
			})
		} else {
			token = cookie.Value
		}

		// 2. Validate for unsafe methods
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			headerToken := r.Header.Get("X-CSRF-Token")
			if headerToken == "" || headerToken != token {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
