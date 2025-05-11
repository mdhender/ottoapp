// Copyright (c) 2024. All rights reserved.

// Package main implements the ottobe API server.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/mdhender/ottoapp/ottobe/api"
	"github.com/mdhender/semver"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	// Version information
	version = semver.Version{Major: 0, Minor: 1, Patch: 2}  // Fixed isAdmin flag for admin user
	
	// Command line flags
	databasePath string
	dataPath     string
	host         string
	port         string
	jwtKey       string
	devMode      bool    // Development mode flag
	showVersion  bool    // Show version and exit
)

// SimpleUserStore is a simple implementation of the UserStore interface for demo purposes
type SimpleUserStore struct {
	users map[int64]*api.User
}

func NewSimpleUserStore() *SimpleUserStore {
	return &SimpleUserStore{
		users: make(map[int64]*api.User),
	}
}

func (s *SimpleUserStore) AuthenticateUser(email, password string) (*api.User, error) {
	// For the demo, we'll allow any login with demo/demo credentials
	if email == "demo@example.com" && password == "demo" {
		return &api.User{
			ID:        1,
			Email:     "demo@example.com",
			Clan:      "0001",
			IsActive:  true,
			IsAdmin:   false,
			Created:   time.Now().AddDate(0, 0, -30),
			LastLogin: time.Now(),
			Timezone:  "UTC",
		}, nil
	}
	
	// For admin login
	if email == "admin@example.com" && password == "admin" {
		return &api.User{
			ID:        2,
			Email:     "admin@example.com",
			Clan:      "0000",
			IsActive:  true,
			IsAdmin:   true,  // Set admin flag for admin user
			Created:   time.Now().AddDate(0, 0, -60),
			LastLogin: time.Now(),
			Timezone:  "UTC",
		}, nil
	}

	return nil, api.ErrUnauthorized
}

func (s *SimpleUserStore) GetUser(userID int64) (*api.User, error) {
	if user, exists := s.users[userID]; exists {
		return user, nil
	}

	// For demo purposes, hardcode some users
	if userID == 1 {
		return &api.User{
			ID:        1,
			Email:     "demo@example.com",
			Clan:      "0001",
			IsActive:  true,
			IsAdmin:   false,
			Created:   time.Now().AddDate(0, 0, -30),
			LastLogin: time.Now(),
			Timezone:  "UTC",
		}, nil
	} else if userID == 2 {
		return &api.User{
			ID:        2,
			Email:     "admin@example.com",
			Clan:      "0000",
			IsActive:  true,
			IsAdmin:   true,   // Ensure admin status is set for GetUser
			Created:   time.Now().AddDate(0, 0, -60),
			LastLogin: time.Now(),
			Timezone:  "UTC",
		}, nil
	}

	return nil, fmt.Errorf("user not found")
}

func (s *SimpleUserStore) CreateUser(email, password, clan, timezone string) (*api.User, error) {
	// Validate email
	if !strings.Contains(email, "@") {
		return nil, api.ErrInvalidEmail
	}

	// Validate clan ID
	if len(clan) != 4 {
		return nil, api.ErrInvalidClan
	}

	// Create a new user with next available ID
	nextID := int64(len(s.users) + 3) // Start from 3 to avoid conflicts with hardcoded demo users
	user := &api.User{
		ID:        nextID,
		Email:     email,
		Clan:      clan,
		IsActive:  true,
		Created:   time.Now(),
		LastLogin: time.Time{}, // Zero time
		Timezone:  timezone,
	}

	// Save the user
	s.users[nextID] = user
	return user, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting ottobe API server v%s", version.String())

	// Parse command line flags
	flag.StringVar(&databasePath, "database", "", "Path to the SQLite database")
	flag.StringVar(&dataPath, "data", "userdata", "Path to user data directory")
	flag.StringVar(&host, "host", "localhost", "Host to serve on")
	flag.StringVar(&port, "port", "29631", "Port to bind to")
	flag.StringVar(&jwtKey, "jwt-key", "", "Secret key for JWT signing")
	flag.BoolVar(&devMode, "dev", false, "Enable development mode (enables route logging and other debug features)")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.Parse()
	
	// Show version and exit if requested
	if showVersion {
		fmt.Printf("OttoApp Backend API Server v%s\n", version.String())
		os.Exit(0)
	}

	// Generate random JWT key if not provided
	if jwtKey == "" {
		randBytes := make([]byte, 32)
		if _, err := rand.Read(randBytes); err != nil {
			log.Fatal("Error generating random JWT key: ", err)
		}
		jwtKey = hex.EncodeToString(randBytes)
		log.Println("Generated random JWT key")
	}

	// Create user store
	// In a real implementation, we would connect to the database here
	// But for now, use a simple in-memory store for demonstration
	userStore := NewSimpleUserStore()

	// Create API handlers
	authHandler := &api.AuthHandler{
		Store:  userStore,
		JWTKey: []byte(jwtKey),
	}

	dataHandler := &api.DataHandler{
		Store:    userStore,
		BasePath: dataPath,
	}

	// Create router
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/auth/user", authHandler.GetUser)
	
	// Data routes
	mux.HandleFunc("GET /api/data", dataHandler.GetUserData)
	mux.HandleFunc("GET /api/data/turn", dataHandler.GetTurnData)

	// Health check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().Format(time.RFC3339))
	})
	
	// Version endpoint
	versionHandler := api.NewVersionHandler(version)
	mux.HandleFunc("GET /api/version", versionHandler.GetVersion)

	// Create debug handler
	if devMode {
		log.Println("Starting in development mode with route logging enabled")
	}
	loggingConfig := api.NewLoggingConfig(devMode)
	debugHandler := &api.DebugHandler{
		LoggingConfig: loggingConfig,
	}

	// Add debug routes - we'll add this manually after applying middleware

	// Apply middlewares
	jwtKeyBytes := []byte(jwtKey)
	handler := api.CORSMiddleware("http://localhost:3000")(mux)
	handler = api.LoggingMiddleware(loggingConfig)(handler)
	handler = api.AuthMiddleware(jwtKeyBytes)(handler)
	
	// Add authorization check for admin routes
	adminOnlyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is admin
		isAdmin, ok := r.Context().Value("isAdmin").(bool)
		log.Printf("Admin route access: isAdmin=%v, ok=%v", isAdmin, ok)
		if !ok || !isAdmin {
			api.RespondWithError(w, http.StatusForbidden, "Admin access required")
			return
		}
		// Call the debug handler if admin
		debugHandler.ToggleRouteLogging(w, r)
	})
	
	// Replace the debug route with the admin-protected version
	mux.Handle("POST /api/admin/debug/log-all-routes", adminOnlyHandler)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("API server listening on %s:%s", host, port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal
	sig := <-sigChan
	log.Printf("Received signal %s, shutting down", sig)

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}