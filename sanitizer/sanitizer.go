package sanitizer

import (
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/dadanrm/hypergon"
)

// SanitizeMiddleware sanitizes all form inputs in the request.
func SanitizeMiddleware() hypergon.Middleware {
	return func(hf hypergon.HandlerFunc) hypergon.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			// Sanitize query parameters
			sanitizedQuery := sanitizeValues(r.URL.Query())
			r.URL.RawQuery = sanitizedQuery.Encode()

			// Sanitize form data (POST, PUT, PATCH requests)
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				if err := r.ParseForm(); err == nil {
					r.Form = sanitizeValues(r.Form)
				}
			}
			return hf(w, r)
		}
	}
}

// sanitizeValues applies HTML escaping and trimming to all values in a map.
func sanitizeValues(values url.Values) url.Values {
	sanitized := url.Values{}
	for key, vals := range values {
		for _, v := range vals {
			// Escape HTML and trim whitespace
			sanitized.Add(key, strings.TrimSpace(html.EscapeString(v)))
		}
	}
	return sanitized
}
