// Copyright (c) 2024. All rights reserved.

package api

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// AuthMiddleware validates JWT tokens and sets user information in request context
func AuthMiddleware(jwtKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for specific paths
			if r.URL.Path == "/api/auth/login" ||
				r.URL.Path == "/api/health" ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				RespondWithError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Check Bearer scheme
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				RespondWithError(w, http.StatusUnauthorized, "Authorization header must be Bearer {token}")
				return
			}

			token := parts[1]
			claims, err := ParseJWT(token, jwtKey)
			if err != nil {
				RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Set claims in request context
			ctx := context.WithValue(r.Context(), "userID", claims.UserID)
			ctx = context.WithValue(ctx, "clan", claims.Clan)
			ctx = context.WithValue(ctx, "isActive", claims.IsActive)
			ctx = context.WithValue(ctx, "isAdmin", claims.IsAdmin)

			log.Printf("Auth middleware: userId=%v, clan=%s, isAdmin=%v",
				claims.UserID, claims.Clan, claims.IsAdmin)

			// Call next handler with enhanced context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ActiveUserMiddleware ensures the user is active
func ActiveUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get active status from context (set by auth middleware)
		isActive, ok := r.Context().Value("isActive").(bool)
		if !ok || !isActive {
			RespondWithError(w, http.StatusForbidden, "Account is inactive")
			return
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(devOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", devOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
