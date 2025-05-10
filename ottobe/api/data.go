// Copyright (c) 2024. All rights reserved.

package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// DataHandler handles data-related API endpoints
type DataHandler struct {
	Store    UserStore
	BasePath string // Base path for user data
}

// GetUserData returns user data information
func (h *DataHandler) GetUserData(w http.ResponseWriter, r *http.Request) {
	// We only need the clan from context for this endpoint

	clan, ok := r.Context().Value("clan").(string)
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Create user data path
	userDataPath := filepath.Join(h.BasePath, clan, "data")

	// Check if data directory exists
	if _, err := os.Stat(userDataPath); os.IsNotExist(err) {
		// Create directory structure if it doesn't exist
		if err := os.MkdirAll(userDataPath, 0755); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Error creating data directory")
			return
		}
		
		// Create input, logs, and output directories
		for _, dir := range []string{"input", "logs", "output"} {
			if err := os.MkdirAll(filepath.Join(userDataPath, dir), 0755); err != nil {
				RespondWithError(w, http.StatusInternalServerError, "Error creating subdirectories")
				return
			}
		}
	}

	// Get directory listing
	response := map[string]interface{}{
		"clan": clan,
		"path": userDataPath,
	}

	RespondWithJSON(w, http.StatusOK, response)
}

// GetTurnData returns data for a specific turn
func (h *DataHandler) GetTurnData(w http.ResponseWriter, r *http.Request) {
	// We don't need the userID for this endpoint, just the clan

	clan, ok := r.Context().Value("clan").(string)
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get query parameters
	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")

	// Validate parameters
	yearNum, err := strconv.Atoi(year)
	if err != nil || yearNum < 1 {
		RespondWithError(w, http.StatusBadRequest, "Invalid year parameter")
		return
	}

	monthNum, err := strconv.Atoi(month)
	if err != nil || monthNum < 1 || monthNum > 12 {
		RespondWithError(w, http.StatusBadRequest, "Invalid month parameter")
		return
	}

	// Create user data path
	userDataPath := filepath.Join(h.BasePath, clan, "data")
	// Construct path to turn data
	turnDataPath := filepath.Join(userDataPath, "output", year, month)

	// Check if directory exists
	if _, err := os.Stat(turnDataPath); os.IsNotExist(err) {
		RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"turn": map[string]int{
				"year":  yearNum,
				"month": monthNum,
			},
			"exists": false,
		})
		return
	}

	// Directory exists, return success
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"turn": map[string]int{
			"year":  yearNum,
			"month": monthNum,
		},
		"exists": true,
		"path":   turnDataPath,
	})
}