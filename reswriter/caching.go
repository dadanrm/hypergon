package reswriter

import (
	"bytes"
	"maps"
	"net/http"
)

type CachingWriter struct {
	http.ResponseWriter
	Body        *bytes.Buffer
	Headers     http.Header
	Status      int
	IsStreaming bool
}

func NewCachingWriter(w http.ResponseWriter) *CachingWriter {
	return &CachingWriter{
		ResponseWriter: w,
		Body:           new(bytes.Buffer),
		Headers:        make(http.Header),
	}
}

// Header buffers the headers.
func (cw *CachingWriter) Header() http.Header {
	return cw.Headers
}

// WriteHeader buffers the status code.
func (cw *CachingWriter) WriteHeader(statusCode int) {
	// Only store the first status code written.
	if cw.Status == 0 {
		cw.Status = statusCode
	}
}

// Write buffers the body until streaming starts.
func (cw *CachingWriter) Write(p []byte) (int, error) {
	if cw.IsStreaming {
		// If we are already streaming, write directly to the client.
		return cw.ResponseWriter.Write(p)
	}
	// Otherwise, write to the internal buffer.
	return cw.Body.Write(p)
}

// Flush is the magic method that switches from buffering to streaming.
func (cw *CachingWriter) Flush() {
	flusher, ok := cw.ResponseWriter.(http.Flusher)
	if !ok {
		// If the underlying writer doesn't support flushing, we can't stream.
		return
	}

	if !cw.IsStreaming {
		// This is the first time Flush is called. Switch to streaming mode.
		cw.IsStreaming = true

		// Write the buffered headers and status code to the actual response.
		maps.Copy(cw.ResponseWriter.Header(), cw.Headers)
		if cw.Status != 0 {
			cw.ResponseWriter.WriteHeader(cw.Status)
		} else {
			cw.ResponseWriter.WriteHeader(http.StatusOK)
		}

		// Write the buffered body content to the actual response.
		if cw.Body.Len() > 0 {
			cw.ResponseWriter.Write(cw.Body.Bytes())
		}
	}

	// Now that we are in streaming mode, call the underlying flusher.
	flusher.Flush()
}
