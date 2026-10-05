package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userUuidKey contextKey = "userUuid"

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		ctx := context.WithValue(r.Context(), userUuidKey, userUuid)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := getAndValidateJwt(w, r)
		if err != nil {
			log.Println(err)
			return
		}

		if claims["role"].(string) != "admin" {
			log.Println(err)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		userUuid, err := claims.GetSubject()
		if err != nil {
			log.Println(err)
			http.Error(w, "User is not logged in", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userUuidKey, userUuid)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
