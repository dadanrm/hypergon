package logging

import (
	"net/http"
	"strings"
	"time"

	"github.com/dadanrm/hypergon/logger"
	"github.com/dadanrm/hypergon/reswriter"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		upgrade := isWebSocketUpgrade(r)

		wrapped := reswriter.New(w)

		next.ServeHTTP(wrapped, r)

		if !upgrade {
			if wrapped.StatusCode() >= 200 && wrapped.StatusCode() < 300 {
				logger.LOG(wrapped.StatusCode(), r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			} else if wrapped.StatusCode() >= 400 && wrapped.StatusCode() < 500 {
				logger.BAD(wrapped.StatusCode(), r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			} else if wrapped.StatusCode() >= 500 {
				logger.ERROR(wrapped.StatusCode(), r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			} else {
				logger.LOG(wrapped.StatusCode(), r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			}
		} else {
			logger.INFO("WS Connection", wrapped.StatusCode(), r.URL.Path, time.Since(start))
		}
	})
}

func isWebSocketUpgrade(r *http.Request) bool {
	connHdr := ""
	connHdrs := r.Header["Connection"]
	if len(connHdrs) > 0 {
		connHdr = connHdrs[0]
	}

	upgradeWebsocket := false
	upgradeHdrs := r.Header["Upgrade"]
	if len(upgradeHdrs) > 0 {
		upgradeWebsocket = (strings.ToLower(upgradeHdrs[0]) == "websocket")
	}

	return connHdr == "Upgrade" && upgradeWebsocket
}
