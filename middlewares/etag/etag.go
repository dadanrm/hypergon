package etag

import (
	"bytes"
	"hash/crc32"
	"maps"
	"net/http"
	"strconv"

	"github.com/dadanrm/hypergon"
)

func New() hypergon.Middleware {
	return func(hf hypergon.HandlerFunc) hypergon.HandlerFunc {
		var (
			headerEtag        = "Etag"
			headerIfNoneMatch = "If-None-Match"
			weakPrefix        = []byte("W/\"")
			quoteSuffix       = []byte("\"")
		)

		const crcPol = 0x82F63B78
		crc32q := crc32.MakeTable(crcPol)

		return func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
			ew := &etagwriter{
				ResponseWriter: w,
				body:           bytes.NewBuffer(nil),
				headers:        make(http.Header),
			}

			if err := hf(ew, r); err != nil {
				return err
			}

			body := ew.body.Bytes()

			if ew.status != http.StatusOK || len(body) == 0 {

				maps.Copy(w.Header(), ew.headers)

				w.WriteHeader(ew.status)
				w.Write(body)

				return nil
			}

			checksum := crc32.Checksum(body, crc32q)
			etag := bytes.NewBuffer(weakPrefix)
			etag.WriteString(strconv.FormatUint(uint64(checksum), 16))
			etag.Write(quoteSuffix)
			generatedEtag := etag.Bytes()

			if match := r.Header.Get(headerIfNoneMatch); match != "" && bytes.Equal([]byte(match), generatedEtag) {

				w.Header().Set(string(headerEtag), string(generatedEtag))
				w.WriteHeader(http.StatusNotModified)
				return nil
			} else {
				maps.Copy(w.Header(), ew.headers)

				w.Header().Set(headerEtag, string(generatedEtag))
				w.WriteHeader(ew.status)
				w.Write(body)

			}

			return nil
		}
	}
}

// etagWriter is a wrapper around http.ResponseWriter that captures the response body and status code.
type etagwriter struct {
	http.ResponseWriter
	body    *bytes.Buffer
	headers http.Header
	status  int
}

func (w *etagwriter) Header() http.Header {
	return w.headers
}

// WriteHeader captures the status code and calls the original WriteHeader.
func (w *etagwriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
}

// Write captures the response body and writes it to both the internal buffer and the original ResponseWriter.
func (w *etagwriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.body.Write(b)
}
