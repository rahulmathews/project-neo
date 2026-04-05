package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "user_id"
	contextKeyUserRole contextKey = "user_role"
)

// Middleware validates Supabase JWTs and injects user_id + role into context.
// Unauthenticated requests pass through — resolvers enforce auth per operation.
func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				ctx = parseToken(ctx, tokenStr, jwtSecret)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseToken(ctx context.Context, tokenStr, secret string) context.Context {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return ctx
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctx
	}

	if sub, ok := claims["sub"].(string); ok {
		if id, err := uuid.Parse(sub); err == nil {
			ctx = context.WithValue(ctx, contextKeyUserID, id)
		}
	}

	if role, ok := claims["role"].(string); ok {
		ctx = context.WithValue(ctx, contextKeyUserRole, role)
	}

	return ctx
}

// UserIDFromCtx extracts the authenticated user's UUID from context.
// Returns an error if the user is not authenticated.
func UserIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(contextKeyUserID).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("unauthenticated")
	}
	return id, nil
}

// RoleFromCtx returns the user's role from context ("authenticated" | "anon").
func RoleFromCtx(ctx context.Context) string {
	role, _ := ctx.Value(contextKeyUserRole).(string)
	if role == "" {
		return "anon"
	}
	return role
}
