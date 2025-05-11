// Copyright (c) 2024. All rights reserved.

package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

// Common domain errors
var (
	ErrInvalidClan     = errors.New("invalid clan")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidTimezone = errors.New("invalid timezone")
	ErrUnauthorized    = errors.New("unauthorized")
)

// User represents a user in the system
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Clan      string    `json:"clan"`
	IsActive  bool      `json:"isActive"`
	IsAdmin   bool      `json:"isAdmin"`
	Created   time.Time `json:"created"`
	LastLogin time.Time `json:"lastLogin"`
	Timezone  string    `json:"timezone"`
}

// UserStore defines the interface for user storage operations
type UserStore interface {
	AuthenticateUser(email, password string) (*User, error)
	GetUser(userID int64) (*User, error)
	CreateUser(email, password, clan, timezone string) (*User, error)
}

// LoginRequest represents the JSON payload for login requests
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the JSON response for login requests
type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
	Clan    string `json:"clan,omitempty"`
	UserID  int64  `json:"userId,omitempty"`
}

// UserResponse represents the JSON response for user information
type UserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Clan      string    `json:"clan"`
	IsActive  bool      `json:"isActive"`
	IsAdmin   bool      `json:"isAdmin"`
	Created   time.Time `json:"created"`
	LastLogin time.Time `json:"lastLogin"`
	Timezone  string    `json:"timezone"`
}

// AuthHandler handles authentication routes
type AuthHandler struct {
	Store  UserStore
	JWTKey []byte // Key for signing JWT tokens
}

// Register handles user registration requests
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Clan     string `json:"clan"`
		Timezone string `json:"timezone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Create user
	user, err := h.Store.CreateUser(req.Email, req.Password, req.Clan, req.Timezone)
	if err != nil {
		code := http.StatusInternalServerError
		msg := "Error creating user"

		switch err {
		case ErrInvalidEmail:
			code = http.StatusBadRequest
			msg = "Invalid email format"
		case ErrInvalidClan:
			code = http.StatusBadRequest
			msg = "Invalid clan ID"
		}

		RespondWithError(w, code, msg)
		return
	}

	RespondWithJSON(w, http.StatusCreated, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Clan:      user.Clan,
		IsActive:  user.IsActive,
		IsAdmin:   user.IsAdmin,
		Created:   user.Created,
		LastLogin: user.LastLogin,
		Timezone:  user.Timezone,
	})
}

// Login handles user login requests
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	log.Printf("Login attempt for user: %q", req.Email)

	// Authenticate user
	user, err := h.Store.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		code := http.StatusInternalServerError
		msg := "Error during authentication"

		if err == ErrUnauthorized {
			code = http.StatusUnauthorized
			msg = "Invalid credentials"
		}

		log.Printf("Login attempt for user: %q: failed", req.Email)
		RespondWithJSON(w, code, LoginResponse{
			Success: false,
			Message: msg,
		})
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(h.JWTKey, user.ID, user.Clan, user.IsActive, user.IsAdmin)
	if err != nil {
		log.Printf("Login attempt for user: %q: token creation failed", req.Email)
		RespondWithJSON(w, http.StatusInternalServerError, LoginResponse{
			Success: false,
			Message: "Error generating authentication token",
		})
		return
	}

	log.Printf("Login attempt for user: %q: succeeded", req.Email)

	RespondWithJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		Token:   token,
		Clan:    user.Clan,
		UserID:  user.ID,
	})
}

// GetUser returns the current user's information
func (h *AuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by authorization middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get user details
	user, err := h.Store.GetUser(userID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Error retrieving user information")
		return
	}

	RespondWithJSON(w, http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Clan:      user.Clan,
		IsActive:  user.IsActive,
		IsAdmin:   user.IsAdmin,
		Created:   user.Created,
		LastLogin: user.LastLogin,
		Timezone:  user.Timezone,
	})
}
