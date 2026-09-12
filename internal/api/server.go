// Package api serves the chessdocs contribution endpoint over HTTP.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

// GitHub is the slice of the upstream repository this package needs. It is
// declared here, rather than imported, so the handlers can be exercised
// without a network.
type GitHub interface {
	FetchSource(ctx context.Context, sourcePath string) (string, error)
	OpenPullRequest(ctx context.Context, sub docs.Submission) (url string, err error)
}

// Options configure NewServer. GitHub is required; the rest have defaults.
type Options struct {
	GitHub GitHub
	// AllowedOrigin is the one browser origin allowed to call the API.
	AllowedOrigin string
	// Logger receives request and failure records. Defaults to slog's default.
	Logger *slog.Logger
}

// Request budgets. Reads are capped well under the 4000-character content limit
// so oversized payloads are dropped before any parsing happens, and writes are
// rationed because every accepted submission spends the server's GitHub token.
const (
	maxBodyBytes = 16 << 10

	readRequests = 60
	readWindow   = time.Minute

	writeRequests = 5
	writeWindow   = 10 * time.Minute
)

type server struct {
	github GitHub
	log    *slog.Logger
}

// NewServer builds the API handler. Both endpoints live at "/": GET returns a
// page's markdown source, POST turns an edited page into a pull request.
func NewServer(opts Options) http.Handler {
	s := &server{github: opts.GitHub, log: opts.Logger}
	if s.log == nil {
		s.log = slog.Default()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.getSource)
	mux.Handle("POST /{$}", newLimiter(writeRequests, writeWindow).limit(http.HandlerFunc(s.createSubmission)))
	mux.HandleFunc("/", s.notFound)

	return chain(mux,
		recoverPanics(s.log),
		logRequests(s.log),
		cors(opts.AllowedOrigin),
		newLimiter(readRequests, readWindow).limit,
	)
}

type middleware func(http.Handler) http.Handler

// chain wraps h so that the first middleware listed is the outermost one, i.e.
// the order they are written is the order a request travels through them.
func chain(h http.Handler, wrappers ...middleware) http.Handler {
	for i := len(wrappers) - 1; i >= 0; i-- {
		h = wrappers[i](h)
	}
	return h
}
