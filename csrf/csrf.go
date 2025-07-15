package csrf

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/dadanrm/hypergon"
)

type contextKey string

type CSRF struct {
	key contextKey
}

func New(key ...string) *CSRF {
	keyString := contextKey("csrf_token")
	if len(key) > 0 {
		keyString = contextKey(key[0])
	}

	return &CSRF{key: keyString}
}

func (cs *CSRF) Middleware() hypergon.Middleware {
	return func(hf hypergon.HandlerFunc) hypergon.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			var csrfToken string

			cookie, err := r.Cookie("csrf_token")
			if err != nil || cookie.Value == "" {
				// Generate new token if missing
				csrfToken = generateCSRFToken()

				// Set cache-control header to prevent Cloudflare or browsers from caching this respons
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")

				// Set secure HttpOnly CSRF token for backend verification
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token",
					Value:    csrfToken,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})

				// Set readable CSRF token for frontend (e.g., JavaScript)
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token_js",
					Value:    csrfToken,
					HttpOnly: false,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})
			} else {
				csrfToken = cookie.Value
			}

			// Store token in request context for access downstream
			ctx := context.WithValue(r.Context(), cs.key, csrfToken)
			r = r.WithContext(ctx)

			// CSRF validation for non-GET requests
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				formToken := r.FormValue(string(cs.key))
				headerToken := r.Header.Get("X-CSRF-Token")

				if formToken != csrfToken && headerToken != csrfToken {
					http.Error(w, "Invalid CSRF Token", http.StatusForbidden)
					return hypergon.HttpError(http.StatusForbidden, "Invalid CSRF Token")
				}
			}

			// Proceed to next handler
			return hf(w, r)
		}
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}
