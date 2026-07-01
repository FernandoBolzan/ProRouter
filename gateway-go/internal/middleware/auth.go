package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/prorouter/prorouter/internal/database"
)

type contextKey string

const APIKeyIDKey contextKey = "api_key_id"

func AuthMiddleware(db *database.DB, bypassPaths []string) func(http.Handler) http.Handler {
	bypass := make(map[string]bool)
	for _, p := range bypassPaths {
		bypass[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for bypass paths (support prefix matching)
			for _, bp := range bypassPaths {
				if bp == "/" {
					if r.URL.Path == "/" {
						ctx := context.WithValue(r.Context(), APIKeyIDKey, "anonymous")
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
					continue
				}
				if r.URL.Path == bp || strings.HasPrefix(r.URL.Path, strings.TrimRight(bp, "/")+"/") {
					ctx := context.WithValue(r.Context(), APIKeyIDKey, "anonymous")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				http.Error(w, `{"error":"invalid Authorization format"}`, http.StatusUnauthorized)
				return
			}

			// Strip "pr-" prefix before hashing for lookup
			rawKey := strings.TrimPrefix(token, "pr-")
			hash := sha256.Sum256([]byte(rawKey))
			keyHash := hex.EncodeToString(hash[:])

			apiKey, err := db.GetAPIKeyByHash(keyHash)
			if err != nil {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			if apiKey.IsRevoked {
				http.Error(w, `{"error":"API key has been revoked"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), APIKeyIDKey, apiKey.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
