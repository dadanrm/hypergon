package reswriter

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode     int
	bytesWritten   int
	headersWritten bool
}

func New(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{ResponseWriter: w}
}

func (w *ResponseWriterWrapper) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *ResponseWriterWrapper) BytesWritten() int {
	return w.bytesWritten
}

func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	if w.headersWritten {
		return
	}
	w.statusCode = statusCode
	w.headersWritten = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *ResponseWriterWrapper) Write(b []byte) (int, error) {
	if !w.headersWritten {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

func (w *ResponseWriterWrapper) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *ResponseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker is not supported")
}
