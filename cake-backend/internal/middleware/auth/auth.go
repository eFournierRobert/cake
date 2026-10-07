// Package auth provides HTTP middleware for JWT-based authentication and authorization.
package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for context keys to avoid collisions between
// different packages that might use context values.
type contextKey string

// UserUuidKey is the context key for the authenticated user's UUID.
const UserUuidKey contextKey = "userUuid"

// RequireAuth is middleware that validates a JWT token from the "jwt-token" cookie.
// If validation succeeds, it adds the user's UUID to the request context and
// calls the next handler. If validation fails, it logs the error and returns
// a 401 Unauthorized response.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := getAndValidateJwt(w, r)
		if err != nil {
			log.Println(err)
			return
		}

		userUuid, err := claims.GetSubject()
		if err != nil {
			log.Println(err)
			http.Error(w, "User is not logged in", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserUuidKey, userUuid)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAdminAuth is middleware that validates a JWT token and ensures the
// user has the "admin" role. If validation succeeds and the user is an admin,
// it adds the user's UUID to the request context and calls the next handler.
// If validation fails or the user is not an admin, it logs the error and returns
// an appropriate error response (401 Unauthorized or 403 Forbidden).
func RequireAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := getAndValidateJwt(w, r)
		if err != nil {
			log.Println(err)
			return
		}

		if claims["role"].(string) != "admin" {
			log.Println("User is not an admin, access denied")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		userUuid, err := claims.GetSubject()
		if err != nil {
			log.Println(err)
			http.Error(w, "User is not logged in", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserUuidKey, userUuid)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getAndValidateJwt extracts and validates a JWT token from the "jwt-token" cookie.
// It returns the parsed claims if valid, or an error if the token is missing,
// invalid, or has invalid claims.
func getAndValidateJwt(w http.ResponseWriter, r *http.Request) (jwt.MapClaims, error) {
	cookie, err := r.Cookie("jwt-token")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			http.Error(w, "User is not logged in", http.StatusUnauthorized)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return nil, err
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		http.Error(w, "User is not logged in", http.StatusUnauthorized)
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "User is not logged in", http.StatusUnauthorized)
		return nil, errors.New("invalid JWT claims")
	}

	return claims, nil
}