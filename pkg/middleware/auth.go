package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
	RolesContextKey  contextKey = "roles"
	EmailContextKey  contextKey = "email"
)

// AuthMiddleware validates JWT access tokens.
func AuthMiddleware(cfg *config.Config, rdb *redis.Client) func(http.Handler) http.Handler {
	signingKey, err := getSigningKey(cfg.JWT.Secret)
	if err != nil {
		panic(fmt.Sprintf("failed to parse JWT signing key: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header is required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate JWT token
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return signingKey, nil
			})

			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					httpresponse.Error(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
					return
				}
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or malformed token")
				return
			}

			if !token.Valid {
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
				return
			}

			// Extract standard claims
			sub, _ := claims["sub"].(string)
			email, _ := claims["email"].(string)
			rawRoles, _ := claims["roles"].([]any)

			userID, err := uuid.Parse(sub)
			if err != nil {
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user ID claim")
				return
			}

			// Verify Redis blocklist for user suspension
			if rdb != nil {
				blocklistKey := fmt.Sprintf("blocklist:user:%s", userID.String())
				exists, err := rdb.Exists(r.Context(), blocklistKey).Result()
				if err == nil && exists > 0 {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Your account has been suspended")
					return
				}
			}

			roles := make([]string, len(rawRoles))
			for i, r := range rawRoles {
				roles[i], _ = r.(string)
			}

			// Inject claims into request context
			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			ctx = context.WithValue(ctx, RolesContextKey, roles)
			ctx = context.WithValue(ctx, EmailContextKey, email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware attempts to parse JWT access tokens, but proceeds even if missing/expired.
func OptionalAuthMiddleware(cfg *config.Config, rdb *redis.Client) func(http.Handler) http.Handler {
	signingKey, err := getSigningKey(cfg.JWT.Secret)
	if err != nil {
		panic(fmt.Sprintf("failed to parse JWT signing key: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]

			// Parse and validate JWT token
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return signingKey, nil
			})

			if err != nil || !token.Valid {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			// Extract standard claims
			sub, _ := claims["sub"].(string)
			email, _ := claims["email"].(string)
			rawRoles, _ := claims["roles"].([]any)

			userID, err := uuid.Parse(sub)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Verify Redis blocklist for user suspension
			if rdb != nil {
				blocklistKey := fmt.Sprintf("blocklist:user:%s", userID.String())
				exists, err := rdb.Exists(r.Context(), blocklistKey).Result()
				if err == nil && exists > 0 {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Your account has been suspended")
					return
				}
			}

			roles := make([]string, len(rawRoles))
			for i, r := range rawRoles {
				roles[i], _ = r.(string)
			}

			// Inject claims into request context
			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			ctx = context.WithValue(ctx, RolesContextKey, roles)
			ctx = context.WithValue(ctx, EmailContextKey, email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if the authenticated user has at least one of the required roles.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, err := GetUserRoles(r.Context())
			if err != nil {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied: unauthenticated")
				return
			}

			for _, allowed := range allowedRoles {
				for _, userRole := range roles {
					if userRole == allowed {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied: insufficient permissions")
		})
	}
}

// GetUserID retrieves the authenticated user's UUID from context.
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(UserIDContextKey)
	if val == nil {
		return uuid.Nil, errors.New("user ID not found in context")
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID type in context")
	}
	return id, nil
}

// GetUserRoles retrieves the roles list from context.
func GetUserRoles(ctx context.Context) ([]string, error) {
	val := ctx.Value(RolesContextKey)
	if val == nil {
		return nil, errors.New("roles not found in context")
	}
	roles, ok := val.([]string)
	if !ok {
		return nil, errors.New("invalid roles type in context")
	}
	return roles, nil
}

// GetUserEmail retrieves the user email from context.
func GetUserEmail(ctx context.Context) (string, error) {
	val := ctx.Value(EmailContextKey)
	if val == nil {
		return "", errors.New("email not found in context")
	}
	email, ok := val.(string)
	if !ok {
		return "", errors.New("invalid email type in context")
	}
	return email, nil
}

func getSigningKey(secretB64 string) ([]byte, error) {
	if secretB64 == "" {
		return nil, errors.New("JWT secret key is missing")
	}
	key, err := base64.StdEncoding.DecodeString(secretB64)
	if err != nil {
		// Fallback to raw string for dev keys
		return []byte(secretB64), nil
	}
	return key, nil
}
