package middleware

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// maxConcurrentSessions is the maximum number of active sessions allowed per
// user. When a new login exceeds this limit the oldest session is evicted
// (H-017).
const maxConcurrentSessions = 5

// sessionMaxAge defines how long a tracked session entry is kept before the
// cleanup goroutine removes it. Aligned with maxSessionAge (24h) from auth.go.
const sessionMaxAge = 24 * time.Hour

// cleanupInterval controls how often expired session entries are purged.
const cleanupInterval = 1 * time.Hour

// SessionTracker tracks concurrent sessions per user using an in-memory map.
// On login, sessions are added via Add(). On logout, sessions are removed via
// Remove(). A background goroutine periodically removes entries older than
// sessionMaxAge.
type SessionTracker struct {
	mu       sync.Mutex
	sessions map[uuid.UUID][]int64 // user ID -> list of issued_at unix timestamps
	stopCh   chan struct{}
}

// NewSessionTracker creates a new SessionTracker and starts its cleanup goroutine.
func NewSessionTracker() *SessionTracker {
	st := &SessionTracker{
		sessions: make(map[uuid.UUID][]int64),
		stopCh:   make(chan struct{}),
	}
	go st.cleanup()
	return st
}

// Add registers a new session for the given user. If the user already has
// maxConcurrentSessions sessions, the oldest one is evicted. Returns the
// issued_at timestamp of the new session.
func (st *SessionTracker) Add(userID uuid.UUID) int64 {
	st.mu.Lock()
	defer st.mu.Unlock()

	issuedAt := time.Now().Unix()
	sessions := st.sessions[userID]
	sessions = append(sessions, issuedAt)

	// Evict oldest if over the limit.
	if len(sessions) > maxConcurrentSessions {
		sessions = sessions[len(sessions)-maxConcurrentSessions:]
	}

	st.sessions[userID] = sessions
	return issuedAt
}

// Remove removes a specific session (by issuedAt timestamp) for the given user.
// Called on logout.
func (st *SessionTracker) Remove(userID uuid.UUID, issuedAt int64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	sessions := st.sessions[userID]
	for i, ts := range sessions {
		if ts == issuedAt {
			st.sessions[userID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}
	if len(st.sessions[userID]) == 0 {
		delete(st.sessions, userID)
	}
}

// IsValid checks whether a session identified by (userID, issuedAt) is still
// tracked (i.e. has not been evicted by a newer login).
func (st *SessionTracker) IsValid(userID uuid.UUID, issuedAt int64) bool {
	st.mu.Lock()
	defer st.mu.Unlock()

	for _, ts := range st.sessions[userID] {
		if ts == issuedAt {
			return true
		}
	}
	return false
}

// Count returns the number of active sessions for a user.
func (st *SessionTracker) Count(userID uuid.UUID) int {
	st.mu.Lock()
	defer st.mu.Unlock()

	return len(st.sessions[userID])
}

// Stop stops the cleanup goroutine.
func (st *SessionTracker) Stop() {
	close(st.stopCh)
}

// cleanup periodically removes session entries older than sessionMaxAge.
func (st *SessionTracker) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			st.mu.Lock()
			cutoff := time.Now().Add(-sessionMaxAge).Unix()
			for userID, sessions := range st.sessions {
				var kept []int64
				for _, ts := range sessions {
					if ts > cutoff {
						kept = append(kept, ts)
					}
				}
				if len(kept) == 0 {
					delete(st.sessions, userID)
				} else {
					st.sessions[userID] = kept
				}
			}
			st.mu.Unlock()
		case <-st.stopCh:
			return
		}
	}
}
