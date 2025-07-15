package logging

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dadanrm/hypergon/logger"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
	hijacked   bool
}

func (w *wrappedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// Ensure the underlying ResponseWriter also implements http.Hijacker
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		// The underlying ResponseWriter does not support hijacking, which is a problem.
		return nil, nil, fmt.Errorf("the ResponseWriter does not implement http.Hijacker")
	}

	// Call the Hijack method on the underlying ResponseWriter
	w.hijacked = true
	return hj.Hijack()
}

func (w *wrappedWriter) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	if w.hijacked {
		return
	}

	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
	if w.hijacked {
		return 0, fmt.Errorf("connection has been hijacked")
	}

	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
		w.ResponseWriter.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		upgrade := isWebSocketUpgrade(r)

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			hijacked:       upgrade,
		}

		next.ServeHTTP(wrapped, r)

		statusCode := wrapped.statusCode

		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		if !upgrade {
			if wrapped.statusCode >= 200 && wrapped.statusCode < 300 {
				logger.LOG(wrapped.statusCode, r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			} else if wrapped.statusCode >= 400 && wrapped.statusCode < 500 {
				logger.BAD(wrapped.statusCode, r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			} else if wrapped.statusCode >= 500 {
				logger.ERROR(wrapped.statusCode, r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			} else {
				logger.LOG(wrapped.statusCode, r.URL.Path, r.URL.Query().Encode(), time.Since(start))
			}
		} else {
			logger.INFO("WS Connection", wrapped.statusCode, r.URL.Path, time.Since(start))
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
