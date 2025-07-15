package hypergon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/dadanrm/hypergon/logging"
)

// Custom http error type
type HypergonError interface {
	Error() string
	StatusCode() int
}

type httperror struct {
	status  int
	message string
}

func (he httperror) StatusCode() int {
	return he.status
}

func (he httperror) Error() string {
	return fmt.Sprintf("Status: %d, Message: %s", he.status, he.message)
}

func HttpError(status int, message string) HypergonError {
	return &httperror{status: status, message: message}
}

// HandlerFunc is a custom http handler that can return an error
type HandlerFunc func(http.ResponseWriter, *http.Request) HypergonError

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		http.Error(w, err.Error(), err.StatusCode())
	}
}

// Middleware func type for easy middleware typing.
type Middleware func(HandlerFunc) HandlerFunc

// Hyper act as a custom web router multiplexer that utilizes the standard library.
type Hyper struct {
	mux         *http.ServeMux
	server      *http.Server
	middlewares []Middleware
	mu          sync.Mutex
}

func New() *Hyper {
	return &Hyper{
		mux:         http.NewServeMux(),
		middlewares: []Middleware{},
	}
}

// Use method will accept a middleware.
func (hy *Hyper) Use(m Middleware) {
	hy.middlewares = append([]Middleware{m}, hy.middlewares...) // ensures first added runs first.
}

func (hy *Hyper) Chain(middlewares ...Middleware) {
	for _, mw := range middlewares {
		hy.Use(mw)
	}
}

// The implemetation of custom http handler
func (hy *Hyper) Action(pattern string, handler HandlerFunc) {
	finalHandler := handler

	for _, m := range hy.middlewares {
		finalHandler = m(finalHandler)
	}

	hy.mux.Handle(pattern, finalHandler)
}

func (hy *Hyper) Group(prefix string) *Hyper {
	group := &Hyper{
		mux:         http.NewServeMux(),
		middlewares: make([]Middleware, len(hy.middlewares)),
	}

	copy(group.middlewares, hy.middlewares)

	hy.mux.Handle(prefix+"/", http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		group.mux.ServeHTTP(w, r)
	})))

	return group
}

// Start the server
//
// Accepts adn address eg. `:8000`
func (hy *Hyper) Start(address string) error {
	hy.mu.Lock()
	hy.server = &http.Server{
		Addr:    address,
		Handler: logging.Logging(hy.mux),
	}
	hy.mu.Unlock()

	return hy.server.ListenAndServe()
}

func (hy *Hyper) Shutdown(ctx context.Context) error {
	hy.mu.Lock()
	server := hy.server
	hy.mu.Unlock()

	if server == nil {
		return errors.New("server not running")
	}

	return hy.server.Shutdown(ctx)
}

// Adapter for handling native implementation of standard handlers for Hyper.
func AdapterHandler(h http.Handler) HandlerFunc {
	// return func(w http.ResponseWriter, r *http.Request) error {
	// 	h.ServeHTTP(w, r)
	// 	return nil
	// }
	return func(w http.ResponseWriter, r *http.Request) HypergonError {
		h.ServeHTTP(w, r)
		return nil
	}
}

// This will convert the Hyper HandlerFunc to native handler.
func Adapter(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
