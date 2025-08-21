package etag

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dadanrm/hypergon"
)

type (
	MockHypergonError hypergon.HypergonError
	MockHandlerFunc   hypergon.HandlerFunc
	MockMiddleware    hypergon.Middleware
)

func TestEtagMiddleware(t *testing.T) {
	const expectedEtagForHelloWorld = `W/"4d551068"`

	testCases := []struct {
		name               string
		requestHeader      map[string]string
		handlerResponse    string
		handlerStatusCode  int
		expectedStatusCode int
		expectEtagHeader   bool
		expectedBody       string
	}{
		{
			name:               "First request should generate ETag and return 200 OK",
			requestHeader:      nil,
			handlerResponse:    "Hello, World!",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusOK,
			expectEtagHeader:   true,
			expectedBody:       "Hello, World!",
		},
		{
			name: "Request with matching ETag should return 304 Not Modified",
			requestHeader: map[string]string{
				"If-None-Match": expectedEtagForHelloWorld,
			},
			handlerResponse:    "Hello, World!",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusNotModified,
			expectEtagHeader:   true, // 304 responses MUST include the ETag header
			expectedBody:       "",   // Body must be empty for 304
		},
		{
			name: "Request with mismatched ETag should return 200 OK with new ETag",
			requestHeader: map[string]string{
				"If-None-Match": `W/"some-old-etag"`,
			},
			handlerResponse:    "Hello, World!",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusOK,
			expectEtagHeader:   true,
			expectedBody:       "Hello, World!",
		},
		{
			name:               "Request with non-200 status should not generate ETag",
			requestHeader:      nil,
			handlerResponse:    "Internal Server Error",
			handlerStatusCode:  http.StatusInternalServerError,
			expectedStatusCode: http.StatusInternalServerError,
			expectEtagHeader:   false,
			expectedBody:       "Internal Server Error",
		},
		{
			name:               "Request with empty body should not generate ETag",
			requestHeader:      nil,
			handlerResponse:    "",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusOK,
			expectEtagHeader:   false,
			expectedBody:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			etagMiddleware := MockMiddleware(New())

			mockHandler := func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
				w.WriteHeader(tc.handlerStatusCode)
				fmt.Fprint(w, tc.handlerResponse)
				return nil
			}

			handlerWithMiddleware := etagMiddleware(mockHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)

			for key, value := range tc.requestHeader {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()

			handlerWithMiddleware(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tc.expectedStatusCode, rr.Code)
			}

			if rr.Body.String() != tc.expectedBody {
				t.Errorf("expected body '%s', got '%s'", tc.expectedBody, rr.Body.String())
			}

			etagHeader := rr.Header().Get("Etag")

			if tc.expectEtagHeader {
				if etagHeader == "" {
					t.Error("expected Etag header but it was not set")
				}

				if tc.expectedStatusCode == http.StatusOK || tc.expectedStatusCode == http.StatusNotModified {
					if etagHeader != expectedEtagForHelloWorld {
						t.Errorf("expected Etag '%s', got '%s'", expectedEtagForHelloWorld, etagHeader)
					}
				}

			} else {
				if etagHeader != "" {
					t.Errorf("did not expect Etag header, but got '%s'", etagHeader)
				}
			}
		})
	}
}
