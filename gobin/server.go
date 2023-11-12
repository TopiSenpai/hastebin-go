package gobin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-jose/go-jose/v3"
	"github.com/topi314/tint"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/topi314/gobin/templates"
)

type ExecuteTemplateFunc func(wr io.Writer, name string, data any) error

func NewServer(version string, debug bool, cfg Config, db *DB, signer jose.Signer, tracer trace.Tracer, meter metric.Meter, assets http.FileSystem, tmpl ExecuteTemplateFunc) *Server {
	var allStyles []templates.Style
	for _, name := range styles.Names() {
		allStyles = append(allStyles, templates.Style{
			Name:  name,
			Theme: styles.Get(name).Theme,
		})
	}

	s := &Server{
		version: version,
		debug:   debug,
		cfg:     cfg,
		db:      db,
		client: &http.Client{
			Transport: otelhttp.NewTransport(
				http.DefaultTransport,
				otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
					return otelhttptrace.NewClientTrace(ctx)
				}),
			),
			Timeout: cfg.Webhook.Timeout,
		},
		signer: signer,
		tracer: tracer,
		meter:  meter,
		assets: assets,
		styles: allStyles,
		tmpl:   tmpl,
	}

	s.server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: s.Routes(),
	}

	if cfg.RateLimit != nil && cfg.RateLimit.Requests > 0 && cfg.RateLimit.Duration > 0 {
		s.rateLimitHandler = httprate.NewRateLimiter(
			cfg.RateLimit.Requests,
			cfg.RateLimit.Duration,
			httprate.WithLimitHandler(s.rateLimit),
			httprate.WithKeyFuncs(
				httprate.KeyByIP,
				httprate.KeyByEndpoint,
			),
		).Handler
	}

	return s
}

type Server struct {
	version          string
	debug            bool
	cfg              Config
	db               *DB
	server           *http.Server
	client           *http.Client
	signer           jose.Signer
	tracer           trace.Tracer
	meter            metric.Meter
	assets           http.FileSystem
	styles           []templates.Style
	tmpl             ExecuteTemplateFunc
	rateLimitHandler func(http.Handler) http.Handler
	webhookWaitGroup sync.WaitGroup
}

func (s *Server) Start() {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Error while listening", tint.Err(err))
		os.Exit(1)
	}
}

func (s *Server) Close() {
	if err := s.server.Close(); err != nil {
		slog.Error("Error while closing server", tint.Err(err))
	}

	s.webhookWaitGroup.Wait()

	if err := s.db.Close(); err != nil {
		slog.Error("Error while closing database", tint.Err(err))
	}
}

func (s *Server) prettyError(w http.ResponseWriter, r *http.Request, err error, status int) {
	w.WriteHeader(status)

	vars := templates.ErrorVariables{
		Error:     err.Error(),
		Status:    status,
		RequestID: middleware.GetReqID(r.Context()),
		Path:      r.URL.Path,
	}
	if tmplErr := s.tmpl(w, "error.gohtml", vars); tmplErr != nil && !errors.Is(tmplErr, http.ErrHandlerTimeout) {
		slog.ErrorContext(r.Context(), "failed to execute error template", tint.Err(tmplErr))
	}
}

func (s *Server) error(w http.ResponseWriter, r *http.Request, err error, status int) {
	if errors.Is(err, http.ErrHandlerTimeout) {
		return
	}
	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "internal server error", tint.Err(err))
	}
	s.json(w, r, ErrorResponse{
		Message:   err.Error(),
		Status:    status,
		Path:      r.URL.Path,
		RequestID: middleware.GetReqID(r.Context()),
	}, status)
}

func (s *Server) ok(w http.ResponseWriter, r *http.Request, v any) {
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.json(w, r, v, http.StatusOK)
}

func (s *Server) json(w http.ResponseWriter, r *http.Request, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.ErrorContext(r.Context(), "failed to encode json", tint.Err(err))
	}
}

func (s *Server) exceedsMaxDocumentSize(w http.ResponseWriter, r *http.Request, content string) bool {
	if s.cfg.MaxDocumentSize > 0 && len([]rune(content)) > s.cfg.MaxDocumentSize {
		s.error(w, r, ErrContentTooLarge(s.cfg.MaxDocumentSize), http.StatusBadRequest)
		return true
	}
	return false
}

func (s *Server) file(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := s.assets.Open(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		_, _ = io.Copy(w, file)
	}
}

func (s *Server) shortContent(content string) string {
	if s.cfg.Preview != nil && s.cfg.Preview.MaxLines > 0 {
		var newLines int
		maxNewLineIndex := strings.IndexFunc(content, func(r rune) bool {
			if r == '\n' {
				newLines++
			}
			return newLines == s.cfg.Preview.MaxLines
		})

		if maxNewLineIndex > 0 {
			content = content[:maxNewLineIndex]
		}
	}
	return content
}

func FormatBuildVersion(version string, commit string, buildTime time.Time) string {
	if len(commit) > 7 {
		commit = commit[:7]
	}

	buildTimeStr := "unknown"
	if !buildTime.IsZero() {
		buildTimeStr = buildTime.Format(time.ANSIC)
	}
	return fmt.Sprintf("Go Version: %s\nVersion: %s\nCommit: %s\nBuild Time: %s\nOS/Arch: %s/%s\n", runtime.Version(), version, commit, buildTimeStr, runtime.GOOS, runtime.GOARCH)
}
