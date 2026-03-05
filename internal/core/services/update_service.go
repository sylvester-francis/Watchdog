package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sylvester-francis/watchdog-proto/protocol"
)

// updateManifest is the JSON structure served at the manifest URL.
type updateManifest struct {
	Version  string                       `json:"version"`
	Binaries map[string]updateManifestBin `json:"binaries"`
}

// updateManifestBin describes a single platform binary.
type updateManifestBin struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
}

// UpdateService checks a remote manifest for newer agent versions and
// provides update payloads to push over WebSocket.
type UpdateService struct {
	manifestURL string
	logger      *slog.Logger
	client      *http.Client

	mu       sync.RWMutex
	manifest *updateManifest
}

// NewUpdateService creates a new UpdateService.
func NewUpdateService(manifestURL string, logger *slog.Logger) *UpdateService {
	return &UpdateService{
		manifestURL: manifestURL,
		logger:      logger,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Start begins the background manifest refresh loop (every 5 minutes).
// It fetches once immediately, then on a ticker.
func (s *UpdateService) Start(ctx context.Context) {
	// Initial fetch (non-blocking — errors are logged, not fatal).
	s.fetchManifest()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.fetchManifest()
			}
		}
	}()
}

// FetchManifest fetches the manifest from the configured URL and caches it.
func (s *UpdateService) fetchManifest() {
	resp, err := s.client.Get(s.manifestURL)
	if err != nil {
		s.logger.Error("failed to fetch update manifest",
			slog.String("url", s.manifestURL),
			slog.String("error", err.Error()),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("update manifest returned non-200",
			slog.String("url", s.manifestURL),
			slog.Int("status", resp.StatusCode),
		)
		return
	}

	var m updateManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		s.logger.Error("failed to decode update manifest",
			slog.String("error", err.Error()),
		)
		return
	}

	s.mu.Lock()
	s.manifest = &m
	s.mu.Unlock()

	s.logger.Info("update manifest refreshed",
		slog.String("version", m.Version),
		slog.Int("platforms", len(m.Binaries)),
	)
}

// GetUpdateForAgent returns an update message if the agent is behind the
// manifest version. Returns nil if up-to-date, dev build, or no binary
// for the agent's platform.
func (s *UpdateService) GetUpdateForAgent(currentVersion, agentOS, agentArch string) *protocol.Message {
	if currentVersion == "" || currentVersion == "dev" {
		return nil
	}

	s.mu.RLock()
	m := s.manifest
	s.mu.RUnlock()

	if m == nil {
		return nil
	}

	if !isNewerVersion(m.Version, currentVersion) {
		return nil
	}

	platform := agentOS + "/" + agentArch
	bin, ok := m.Binaries[platform]
	if !ok {
		return nil
	}

	return protocol.NewUpdateAvailableMessage(m.Version, bin.URL, bin.SHA256, bin.Signature)
}

// ManifestVersion returns the currently cached manifest version, or "" if none.
func (s *UpdateService) ManifestVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.manifest == nil {
		return ""
	}
	return s.manifest.Version
}

// isNewerVersion returns true if manifest version is strictly greater than current.
// Both must be dot-separated numeric strings (e.g. "1.2.3").
func isNewerVersion(manifest, current string) bool {
	mParts := parseSemver(manifest)
	cParts := parseSemver(current)
	if mParts == nil || cParts == nil {
		return false
	}

	for i := 0; i < 3; i++ {
		if mParts[i] > cParts[i] {
			return true
		}
		if mParts[i] < cParts[i] {
			return false
		}
	}
	return false // equal
}

// parseSemver splits a version string like "1.2.3" into [1, 2, 3].
// Returns nil if the format is invalid.
func parseSemver(v string) []int {
	// Strip leading "v" if present.
	v = strings.TrimPrefix(v, "v")

	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}

	result := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		if n < 0 {
			return nil
		}
		result[i] = n
	}
	return result
}

// GetUpdatePayloadForAgent is like GetUpdateForAgent but returns the raw
// payload struct (useful for the admin push-update API response).
func (s *UpdateService) GetUpdatePayloadForAgent(currentVersion, agentOS, agentArch string) *updatePayloadResponse {
	if currentVersion == "" || currentVersion == "dev" {
		return nil
	}

	s.mu.RLock()
	m := s.manifest
	s.mu.RUnlock()

	if m == nil {
		return nil
	}

	if !isNewerVersion(m.Version, currentVersion) {
		return nil
	}

	platform := agentOS + "/" + agentArch
	bin, ok := m.Binaries[platform]
	if !ok {
		return nil
	}

	return &updatePayloadResponse{
		Version:     m.Version,
		DownloadURL: bin.URL,
		SHA256:      bin.SHA256,
		Platform:    fmt.Sprintf("%s/%s", agentOS, agentArch),
	}
}

// updatePayloadResponse is the JSON response for the admin push-update endpoint.
type updatePayloadResponse struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	SHA256      string `json:"sha256"`
	Platform    string `json:"platform"`
}
