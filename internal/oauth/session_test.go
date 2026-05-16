package oauth

import (
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func TestSessionStore_CreateAndGet(t *testing.T) {
	store := NewSessionStore(nil)
	token := &oauth2.Token{AccessToken: "test-token"}

	session := store.Create("user@example.com", "Test User", "pic.jpg", token)

	is.True(t, session.ID != "")
	is.Equal(t, session.Email, "user@example.com", "")

	got := store.Get(session.ID)
	is.True(t, got != nil)
	is.Equal(t, got.Email, "user@example.com", "")
}

func TestSessionStore_Delete(t *testing.T) {
	store := NewSessionStore(nil)
	session := store.Create("user@example.com", "Test User", "", nil)

	store.Delete(session.ID)

	is.True(t, store.Get(session.ID) == nil)
}

func TestSessionStore_GetNonExistent(t *testing.T) {
	store := NewSessionStore(nil)
	is.True(t, store.Get("nonexistent") == nil)
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	store := NewSessionStore(nil)
	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() {
			s := store.Create("user@example.com", "User", "", nil)
			store.Get(s.ID)
			store.Delete(s.ID)
		})
	}

	wg.Wait()
}

func TestSessionStore_Cleanup(t *testing.T) {
	store := NewSessionStore(nil)

	// Create a session and backdate it beyond the TTL.
	session := store.Create("old@example.com", "Old User", "", nil)
	store.mu.Lock()
	store.sessions[session.ID].CreatedAt = time.Now().Add(-(sessionTTL + time.Hour))
	store.mu.Unlock()

	// Create a fresh session.
	fresh := store.Create("new@example.com", "New User", "", nil)

	// Run cleanup inline (same logic as StartCleanup ticker).
	store.mu.Lock()
	for id, sess := range store.sessions {
		if time.Since(sess.CreatedAt) > sessionTTL {
			delete(store.sessions, id)
		}
	}
	store.mu.Unlock()

	is.True(t, store.Get(session.ID) == nil)
	is.True(t, store.Get(fresh.ID) != nil)
}

func TestSessionStore_MaxSessions(t *testing.T) {
	store := NewSessionStore(nil)

	// Fill to capacity.
	for range maxSessions {
		store.Create("user@example.com", "User", "", nil)
	}

	is.Equal(t, store.Len(), maxSessions, "filled to cap")

	// Adding one more should evict the oldest and stay at cap.
	store.Create("new@example.com", "New User", "", nil)

	is.Equal(t, store.Len(), maxSessions, "after overflow")
}
