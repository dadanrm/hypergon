package idempotency

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"time"

	"github.com/dadanrm/hypergon"
	"github.com/dadanrm/hypergon/reswriter"
)

const cacheExpiry = 24 * time.Hour

// Cache is the caching service interface.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	SetNx(ctx context.Context, key string, value any, expiry time.Duration) error
}

// savedResponse is the struct we will cache.
type savedResponse struct {
	StatusCode int         `json:"statusCode"`
	Body       []byte      `json:"body"`
	Header     http.Header `json:"header"`
}

// New creates the idempotency middleware.
func New(cache Cache) hypergon.Middleware {
	return func(hf hypergon.HandlerFunc) hypergon.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
			// 1. Get the idempotency key.
			key := r.Header.Get("Idempotency-Key")

			// 2. Check for pass-through conditions.
			// We only apply idempotency if a key is provided
			// AND the method is one that is typically not idempotent.
			if key == "" || (r.Method != http.MethodPost && r.Method != http.MethodPatch) {
				// No key, or not a method we make idempotent. Act as a simple pass-through.
				return hf(w, r)
			}

			// 3. Try to get the cached response.
			cachedVal, err := cache.Get(r.Context(), key)

			// 4. --- CACHE HIT ---
			if err == nil {
				log.Printf("[Idempotency] HIT for key: %s", key)
				var cachedResp savedResponse

				if err := json.Unmarshal([]byte(cachedVal), &cachedResp); err != nil {
					log.Printf("[Idempotency] ERROR: Corrupt cache for key %s: %v", key, err)
					// Cache is corrupt. Fall through to re-execute.
				} else {
					// Success! Write the cached response back to the client.
					maps.Copy(w.Header(), cachedResp.Header)
					w.WriteHeader(cachedResp.StatusCode)
					w.Write(cachedResp.Body)
					return nil // We are done. DO NOT call the handler.
				}
			}

			// 5. --- CACHE MISS ---
			log.Printf("[Idempotency] MISS for key: %s", key)

			// Create your CachingWriter to wrap the real ResponseWriter.
			cw := reswriter.NewCachingWriter(w)

			// 6. Call the handler, passing our CachingWriter.
			handlerErr := hf(cw, r)
			if handlerErr != nil {
				// The handler returned an error. This check is now safe
				// because the new mockHandler returns a *true* nil.
				log.Printf("[Idempotency] Handler returned error: %v", handlerErr)
				if !cw.IsStreaming {
					cw.Flush() // Send the error response
				}
				return handlerErr
			}

			// 7. Handler was successful. Check if we can cache the response.
			if cw.IsStreaming {
				log.Printf("[Idempotency] WARNING: Handler for key %s started streaming. Cannot cache response.", key)
				return nil
			}

			// 8. The response was NOT streamed and is fully buffered.
			statusCode := cw.Status
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			// Policy: Don't cache 5xx server errors.
			if statusCode >= 500 {
				log.Printf("[Idempotency] WARNING: Handler for key %s returned %d. Not caching server error.", key, statusCode)
				cw.Flush() // Send the 5xx response
				return nil
			}

			// Create the response object to be cached.
			respToCache := savedResponse{
				StatusCode: statusCode,
				Body:       cw.Body.Bytes(),
				Header:     cw.Headers,
			}

			jsonResp, err := json.Marshal(respToCache)
			if err != nil {
				log.Printf("[Idempotency] ERROR: Failed to marshal response for key %s: %v", key, err)
				cw.Flush() // Send the original response
				return nil
			}

			// 9. Atomically save the JSON response to the cache.
			if err := cache.SetNx(r.Context(), key, string(jsonResp), cacheExpiry); err != nil {
				log.Printf("[Idempotency] WARNING: cache.SetNx failed for key %s: %v", key, err)
			} else {
				log.Printf("[Idempotency] CACHED response for key: %s", key)
			}

			// 10. Finally, send the buffered response to the client.
			cw.Flush()

			return nil
		}
	}
}
