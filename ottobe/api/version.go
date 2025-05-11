// Copyright (c) 2024. All rights reserved.

package api

import (
	"encoding/json"
	"fmt"
	"github.com/mdhender/semver"
	"net/http"
	"runtime"
	"time"
)

// VersionHandler handles version-related routes
type VersionHandler struct {
	Version   semver.Version
	StartTime time.Time
}

// NewVersionHandler creates a new version handler
func NewVersionHandler(version semver.Version) *VersionHandler {
	return &VersionHandler{
		Version:   version,
		StartTime: time.Now(),
	}
}

// GetVersion returns the server version information
func (h *VersionHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version": h.Version.String(),
		"build": map[string]interface{}{
			"go":      runtime.Version(),
			"platform": fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		},
		"server": map[string]interface{}{
			"uptime": time.Since(h.StartTime).String(),
			"started": h.StartTime.Format(time.RFC3339),
		},
	}

	// Return version information as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Error encoding version information", http.StatusInternalServerError)
	}
}