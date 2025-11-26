package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type UserMiddleware struct {
	UserStore store.UserStore
}

type contextKey string

const UserContextKey = contextKey("user")

func SetUser(r *http.Request, user *store.User) *http.Request {
	ctx := context.WithValue(r.Context(), UserContextKey, user)
	return r.WithContext(ctx)
}

func GetUser(r *http.Request) *store.User {
	user, ok := r.Context().Value(UserContextKey).(*store.User)
	if !ok {
		panic("missing user in request") // bad actor call
	}
	return user
}

func (um *UserMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// within this anonymous function
		// we can interject any incoming requests to our server

		w.Header().Add("Vary", "Authorization")
		authHeader := r.Header.Get("Authorization")

		var token string

		if authHeader == "" {
			// Try to get token from cookie
			cookie, err := r.Cookie("auth_token")
			if err == nil {
				token = cookie.Value
			}
		} else {
			headerParts := strings.Split(authHeader, " ") // Bearer <TOKEN>
			if len(headerParts) == 2 && headerParts[0] == "Bearer" {
				token = headerParts[1]
			}
		}

		if token == "" {
			r = SetUser(r, store.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		user, err := um.UserStore.GetUserToken(tokens.ScopeAuth, token)
		if err != nil {
			utils.Error(w, http.StatusUnauthorized, "invalid token")
			return
		}

		if user == nil {
			// Token invalid or expired
			// For API clients, this might be 401.
			// For Web clients (cookie), we might want to redirect or just treat as Anonymous.
			// Current logic: returns 401.
			utils.Error(w, http.StatusUnauthorized, "token expired or invalid")
			return
		}

		if !user.IsActive {
			utils.Error(w, http.StatusForbidden, "account disabled")
			return
		}

		r = SetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

func (um *UserMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user.IsAnonymous() {
			// Check if the request prefers HTML (Browser navigation)
			accept := r.Header.Get("Accept")
			if strings.Contains(accept, "text/html") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			utils.Error(w, http.StatusUnauthorized, "you must be logged in to access this route")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (um *UserMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user.IsAnonymous() {
			utils.Error(w, http.StatusUnauthorized, "authentication required")
			return
		}

		if user.Role != "administrator" {
			utils.Error(w, http.StatusForbidden, "insufficient privileges")
			return
		}

		next.ServeHTTP(w, r)
	})
}

