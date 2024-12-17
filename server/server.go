package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/topi314/tint"
	"go.gopad.dev/go-tree-sitter-highlight/html"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/topi314/gobin/v2/internal/httperr"
	"github.com/topi314/gobin/v2/internal/httprate"
	"github.com/topi314/gobin/v2/server/database"
	"github.com/topi314/gobin/v2/server/templates"
)

func NewServer(version string, debug bool, cfg Config, db *database.DB, signer jose.Signer, tracer trace.Tracer, meter metric.Meter, assets http.FileSystem, htmlRenderer *html.Renderer) *Server {
	var allThemes []templates.Theme
	for name, theme := range themes {
		allThemes = append(allThemes, templates.Theme{
			Name:        name,
			ColorScheme: theme.ColorScheme,
		})
	}

	var client *http.Client
	if cfg.Webhook != nil {
		client = &http.Client{
			Transport: otelhttp.NewTransport(
				http.DefaultTransport,
				otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
					return otelhttptrace.NewClientTrace(ctx)
				}),
			),
			Timeout: time.Duration(cfg.Webhook.Timeout),
		}
	}

	s := &Server{
		version:      version,
		debug:        debug,
		cfg:          cfg,
		db:           db,
		client:       client,
		signer:       signer,
		tracer:       tracer,
		meter:        meter,
		assets:       assets,
		themes:       allThemes,
		htmlRenderer: htmlRenderer,
	}

	s.server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: s.Routes(),
	}

	if cfg.RateLimit != nil && cfg.RateLimit.Requests > 0 && cfg.RateLimit.Duration > 0 {
		s.rateLimitHandler = httprate.NewRateLimiter(
			cfg.RateLimit.Requests,
			time.Duration(cfg.RateLimit.Duration),
			func(w http.ResponseWriter, r *http.Request) {
				s.error(w, r, httperr.TooManyRequests(ErrRateLimit))
			},
		).Handler
	}

	return s
}

type Server struct {
	version          string
	debug            bool
	cfg              Config
	db               *database.DB
	server           *http.Server
	client           *http.Client
	signer           jose.Signer
	tracer           trace.Tracer
	meter            metric.Meter
	assets           http.FileSystem
	htmlRenderer     *html.Renderer
	themes           []templates.Theme
	rateLimitHandler func(http.Handler) http.Handler
	webhookWaitGroup sync.WaitGroup
	cleanupCancel    context.CancelFunc
}

func (s *Server) Start() {
	cleanupContext, cancel := context.WithCancel(context.Background())
	s.cleanupCancel = cancel

	go s.cleanup(cleanupContext, time.Duration(s.cfg.Database.CleanupInterval), time.Duration(s.cfg.Database.ExpireAfter))
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Error while listening", tint.Err(err))
	}
}

func (s *Server) Close() {
	s.cleanupCancel()

	if err := s.server.Close(); err != nil {
		slog.Error("Error while closing server", tint.Err(err))
	}

	s.webhookWaitGroup.Wait()

	if err := s.db.Close(); err != nil {
		slog.Error("Error while closing database", tint.Err(err))
	}
}

func (s *Server) cleanup(ctx context.Context, cleanUpInterval time.Duration, expireAfter time.Duration) {
	if cleanUpInterval <= 0 {
		cleanUpInterval = 10 * time.Minute
	}

	ctx, span := s.tracer.Start(ctx, "cleanup", trace.WithAttributes(
		attribute.String("cleanUpInterval", cleanUpInterval.String()),
		attribute.String("expireAfter", expireAfter.String()),
	))
	defer span.End()

	slog.Debug("Starting document cleanup...")
	ticker := time.NewTicker(cleanUpInterval)
	defer func() {
		ticker.Stop()
		slog.Debug("document cleanup stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doCleanup(ctx, expireAfter)
		}
	}
}

func (s *Server) doCleanup(ctx context.Context, expireAfter time.Duration) {
	ctx, span := s.tracer.Start(ctx, "doCleanup")
	defer span.End()

	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()
	documents, err := s.db.DeleteExpiredDocuments(dbCtx, expireAfter)
	if err != nil && !errors.Is(err, context.Canceled) {
		span.SetStatus(codes.Error, "failed to delete expired documents")
		span.RecordError(err)
		slog.ErrorContext(ctx, "failed to delete expired documents", tint.Err(err))
	}

	var wg sync.WaitGroup
	for i := range documents {
		wg.Add(1)
		go func(ctx context.Context, document database.Document) {
			webhooksFiles := make([]WebhookDocumentFile, len(document.Files))
			for i, file := range document.Files {
				webhooksFiles[i] = WebhookDocumentFile{
					Name:      file.Name,
					Content:   file.Content,
					Language:  file.Language,
					ExpiresAt: file.ExpiresAt,
				}
			}
			s.ExecuteWebhooks(ctx, WebhookEventUpdate, WebhookDocument{
				Key:     document.ID,
				Version: document.Version,
				Files:   webhooksFiles,
			})
		}(ctx, documents[i])
	}
	wg.Wait()
}
