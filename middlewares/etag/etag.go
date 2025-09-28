package etag

import (
	"bytes"
	"hash/crc32"
	"maps"
	"net/http"
	"strconv"

	"github.com/dadanrm/hypergon"
	"github.com/dadanrm/hypergon/reswriter"
)

func New() hypergon.Middleware {
	return func(next hypergon.HandlerFunc) hypergon.HandlerFunc {
		var (
			headerEtag        = "Etag"
			headerIfNoneMatch = "If-None-Match"
			weakPrefix        = []byte("W/\"")
			quoteSuffix       = []byte("\"")
		)
		crc32q := crc32.MakeTable(crc32.IEEE)

		return func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
			// 1. Create the caching writer.
			cw := reswriter.NewCachingWriter(w)

			// 2. Execute the next handler EXACTLY ONCE.
			if err := next(cw, r); err != nil {
				return err
			}

			// 3. Check if the handler switched to streaming mode.
			if cw.IsStreaming {
				// The response has already been sent. Do nothing.
				return nil
			}

			// --- If we get here, we know the full response is buffered. ---

			body := cw.Body.Bytes()
			status := cw.Status
			if status == 0 {
				status = http.StatusOK
			}

			if status != http.StatusOK || len(body) == 0 {
				maps.Copy(w.Header(), cw.Headers)
				w.WriteHeader(status)
				w.Write(body)
				return nil
			}

			// Generate and check the ETag just like before.
			checksum := crc32.Checksum(body, crc32q)
			etag := bytes.NewBuffer(weakPrefix)
			etag.WriteString(strconv.FormatUint(uint64(checksum), 16))
			etag.Write(quoteSuffix)
			generatedEtag := etag.Bytes()

			if match := r.Header.Get(headerIfNoneMatch); match != "" && bytes.Equal([]byte(match), generatedEtag) {
				w.Header().Set(headerEtag, string(generatedEtag))
				w.WriteHeader(http.StatusNotModified)
				return nil
			}

			maps.Copy(w.Header(), cw.Headers)
			w.Header().Set(headerEtag, string(generatedEtag))
			w.WriteHeader(status)
			w.Write(body)
			return nil
		}
	}
}
