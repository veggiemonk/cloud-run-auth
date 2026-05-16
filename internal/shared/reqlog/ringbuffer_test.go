package reqlog

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func TestBuffer_AddAndEntries(t *testing.T) {
	buf := NewBuffer()

	buf.Add(Entry{Method: "GET", Path: "/first"})
	buf.Add(Entry{Method: "POST", Path: "/second"})

	entries := buf.Entries()
	is.Equal(t, len(entries), 2, "")
	// Newest first.
	is.Equal(t, entries[0].Path, "/second", "newest first")
	is.Equal(t, entries[1].Path, "/first", "oldest last")
}

func TestBuffer_Empty(t *testing.T) {
	buf := NewBuffer()
	entries := buf.Entries()
	is.Equal(t, len(entries), 0, "")
}

func TestBuffer_Wraparound(t *testing.T) {
	buf := NewBuffer()

	// Fill beyond capacity to trigger wraparound.
	for i := range maxEntries + 50 {
		buf.Add(Entry{
			Method: "GET",
			Path:   fmt.Sprintf("/%d", i),
		})
	}

	entries := buf.Entries()
	is.Equal(t, len(entries), maxEntries, "after wraparound")

	// Most recent entry should be the last one added.
	is.Equal(t, entries[0].Path, fmt.Sprintf("/%d", maxEntries+49), "newest entry")

	// Oldest entry should be the first one that survived the wraparound.
	is.Equal(t, entries[maxEntries-1].Path, fmt.Sprintf("/%d", 50), "oldest entry")
}

func TestBuffer_ConcurrentAccess(t *testing.T) {
	buf := NewBuffer()
	var wg sync.WaitGroup

	// Concurrent writers.
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 100 {
				buf.Add(Entry{
					Timestamp: time.Now(),
					Method:    "GET",
					Path:      fmt.Sprintf("/worker-%d/%d", id, j),
				})
			}
		}(i)
	}

	// Concurrent readers.
	for range 5 {
		wg.Go(func() {
			for range 100 {
				_ = buf.Entries()
			}
		})
	}

	wg.Wait()

	// After all writes, buffer should be full (1000 writes > maxEntries).
	entries := buf.Entries()
	is.Equal(t, len(entries), maxEntries, "")
}

func TestBuffer_EntriesReturnsCopy(t *testing.T) {
	buf := NewBuffer()
	buf.Add(Entry{Method: "GET", Path: "/original"})

	entries := buf.Entries()
	entries[0].Path = "/modified"

	// Verify the buffer is not affected.
	fresh := buf.Entries()
	is.Equal(t, fresh[0].Path, "/original", "buffer should not be mutated by caller")
}
