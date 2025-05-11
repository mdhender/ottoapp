// Copyright (c) 2024. All rights reserved.

package api

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// LoggingConfig holds the logging middleware configuration
type LoggingConfig struct {
	// enabled is accessed atomically to avoid locks
	// Note: This approach doesn't guarantee atomic operations for all
	// readers/writers when toggling, but for our debug purposes, occasional
	// missed log entries are acceptable and won't impact functionality.
	enabled int32
}

// NewLoggingConfig creates a new logging configuration
func NewLoggingConfig(enabledInDevelopment bool) *LoggingConfig {
	config := &LoggingConfig{}
	if enabledInDevelopment {
		atomic.StoreInt32(&config.enabled, 1)
		log.Printf("Route logging enabled in development mode")
	}
	return config
}

// IsEnabled returns true if logging is enabled
func (c *LoggingConfig) IsEnabled() bool {
	return atomic.LoadInt32(&c.enabled) == 1
}

// Toggle switches the logging state and returns the new state
func (c *LoggingConfig) Toggle() bool {
	for {
		current := atomic.LoadInt32(&c.enabled)
		newValue := int32(0)
		if current == 0 {
			newValue = 1
		}
		if atomic.CompareAndSwapInt32(&c.enabled, current, newValue) {
			return newValue == 1
		}
	}
}

// LoggingMiddleware creates middleware that logs all routes when enabled
func LoggingMiddleware(config *LoggingConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.IsEnabled() {
				start := time.Now()
				method := r.Method
				path := r.URL.Path
				query := r.URL.RawQuery
				userAgent := r.UserAgent()
				remoteAddr := r.RemoteAddr

				logInfo := struct {
					Method     string
					Path       string
					Query      string
					UserAgent  string
					RemoteAddr string
					StartTime  time.Time
				}{
					Method:     method,
					Path:       path,
					Query:      query,
					UserAgent:  userAgent,
					RemoteAddr: remoteAddr,
					StartTime:  start,
				}

				// Create a wrapped response writer to capture status code
				respWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

				// Process the request
				next.ServeHTTP(respWriter, r)

				// Calculate duration
				duration := time.Since(start)

				// Log the request details
				log.Printf(
					"[REQUEST] %s %s%s - Status: %d - Duration: %s - Remote: %s - Agent: %s",
					logInfo.Method,
					logInfo.Path,
					query,
					respWriter.statusCode,
					duration,
					logInfo.RemoteAddr,
					logInfo.UserAgent,
				)
			} else {
				// If logging is disabled, just pass the request through
				next.ServeHTTP(w, r)
			}
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// DebugHandler manages debug routes
type DebugHandler struct {
	LoggingConfig *LoggingConfig
}

// ToggleRouteLogging toggles the route logging and returns the new state
func (h *DebugHandler) ToggleRouteLogging(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin (this should be enforced by middleware as well)
	isAdmin, ok := r.Context().Value("isAdmin").(bool)
	log.Printf("Toggle route logging request: isAdmin=%v, ok=%v", isAdmin, ok)
	if !ok || !isAdmin {
		RespondWithError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Toggle the logging state
	enabled := h.LoggingConfig.Toggle()
	status := "disabled"
	if enabled {
		status = "enabled"
	}

	// Respond with the new state
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"logging": status,
	})
}