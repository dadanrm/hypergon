package secureheaders

import (
	"net/http"
	"strings"

	"github.com/dadanrm/hypergon"
)

func New(cfg ...configfunc) hypergon.Middleware {
	var config *config

	if len(cfg) > 0 {
		config = cfg[0]()
	}

	return func(hf hypergon.HandlerFunc) hypergon.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
			// --- Add Security Headers ---

			// 1. Content-Security-Policy (CSP) - customize this for your app

			if len(config.CSPNetworks) > 0 {
				w.Header().Set("Content-Security-Policy", strings.Join(config.CSPNetworks, "; "))
			} else {
				w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; object-src 'none';")
			}

			// 2. HTTP Strict-Transport-Security (HSTS)
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

			// 3. X-Frame-Options
			w.Header().Set("X-Frame-Options", "DENY")

			// 4. X-Content-Type-Options
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// 5. Referrer-Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// 6. Permissions-Policy
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			return hf(w, r)
		}
	}
}

func WithNetworkHash(networks []string) configfunc {
	return func() *config {
		return &config{
			CSPNetworks: networks,
		}
	}
}

type config struct {
	// key: network, value: hash
	CSPNetworks []string
}

type configfunc func() *config
