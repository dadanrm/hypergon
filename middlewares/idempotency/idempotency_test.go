package idempotency

import (
	"context"
	"errors"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dadanrm/hypergon"
)

// --- Mocks & Stubs ---

// mockCache is a thread-safe, in-memory cache for testing.
type mockCache struct {
	mu   sync.RWMutex
	data map[string]string
	t    *testing.T // Logger
}

func newMockCache(t *testing.T) *mockCache {
	return &mockCache{
		data: make(map[string]string),
		t:    t,
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	if !ok {
		log.Printf("[MockCache] GET %s: NOT FOUND", key)
		return "", errors.New("not found")
	}
	log.Printf("[MockCache] GET %s: FOUND", key)
	return val, nil
}

func (m *mockCache) SetNx(ctx context.Context, key string, value any, expiry time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		log.Printf("[MockCache] SETNX %s: FAILED (key exists)", key)
		return errors.New("key already exists")
	}
	m.data[key] = value.(string)
	log.Printf("[MockCache] SETNX %s: SUCCESS", key)
	return nil
}

// --- Test Suite ---

func TestIdempotencyMiddleware(t *testing.T) {
	// A simple handler that increments a counter so we know if it was called.
	var handlerCalls int64

	// This mockHandler is simplified to *guarantee* it returns a true nil.
	mockHandler := func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
		atomic.AddInt64(&handlerCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Handler-Called", "true")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message":"created"}`)) // Use simple w.Write
		return nil                               // This is a true nil
	}

	// Reset counter for each subtest
	reset := func() {
		atomic.StoreInt64(&handlerCalls, 0)
	}

	t.Run("pass-through with no key", func(t *testing.T) {
		reset()
		cache := newMockCache(t)
		middleware := New(cache)
		wrappedHandler := middleware(mockHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rr := httptest.NewRecorder()

		if err := wrappedHandler(rr, req); err != nil {
			t.Fatalf("handler returned an error: %v", err)
		}

		if atomic.LoadInt64(&handlerCalls) != 1 {
			t.Fatalf("handler should be called once, got %d", atomic.LoadInt64(&handlerCalls))
		}
		if len(cache.data) > 0 {
			t.Fatal("cache should be empty")
		}
	})

	t.Run("pass-through with GET method", func(t *testing.T) {
		reset()
		cache := newMockCache(t)
		middleware := New(cache)
		wrappedHandler := middleware(mockHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Idempotency-Key", "get-key-123")
		rr := httptest.NewRecorder()

		if err := wrappedHandler(rr, req); err != nil {
			t.Fatalf("handler returned an error: %v", err)
		}

		if atomic.LoadInt64(&handlerCalls) != 1 {
			t.Fatalf("handler should be called once, got %d", atomic.LoadInt64(&handlerCalls))
		}
		if len(cache.data) > 0 {
			t.Fatal("cache should be empty")
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		reset()
		cache := newMockCache(t)
		middleware := New(cache)
		wrappedHandler := middleware(mockHandler)

		key := "miss-key-123"
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Idempotency-Key", key)
		rr := httptest.NewRecorder()

		if err := wrappedHandler(rr, req); err != nil {
			t.Fatalf("handler returned an error: %v", err)
		}

		// Check handler
		if atomic.LoadInt64(&handlerCalls) != 1 {
			t.Fatalf("handler should be called once, got %d", atomic.LoadInt64(&handlerCalls))
		}

		// Check response
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "created") {
			t.Fatal("response body is incorrect")
		}
		if rr.Header().Get("X-Handler-Called") != "true" {
			t.Fatal("response header is missing")
		}

		// Check cache
		val, err := cache.Get(context.Background(), key)
		if err != nil {
			t.Logf("Current cache keys: %v", maps.Keys(cache.data))
			t.Fatal("response was not cached")
		}

		expectedCacheBody := "eyJtZXNzYWdlIjoiY3JlYXRlZCJ9"
		if !strings.Contains(val, expectedCacheBody) {
			t.Fatal("cached value is incorrect")
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		reset()
		cache := newMockCache(t)
		middleware := New(cache)
		wrappedHandler := middleware(mockHandler)

		key := "hit-key-123"
		// Pre-populate the cache
		preCachedResp := `{"statusCode":202,"body":"YWxyZWFkeSBiZWVuIGRvbmU=","header":{"Content-Type":["application/json"],"X-Cached":["true"]}}`
		cache.SetNx(context.Background(), key, preCachedResp, time.Hour)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Idempotency-Key", key)
		rr := httptest.NewRecorder()

		if err := wrappedHandler(rr, req); err != nil {
			t.Fatalf("handler returned an error: %v", err)
		}

		// Check handler
		if atomic.LoadInt64(&handlerCalls) != 0 {
			t.Fatal("handler should not have been called")
		}

		// Check response
		if rr.Code != http.StatusAccepted { // 202
			t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "already been done") {
			t.Fatal("response body is incorrect")
		}
		if rr.Header().Get("X-Cached") != "true" {
			t.Fatal("response header is incorrect")
		}
	})
}
