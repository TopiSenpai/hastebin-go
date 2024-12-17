package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/stampede"
	"github.com/samber/slog-chi"
	"github.com/topi314/otelchi"
	"github.com/topi314/tint"

	"github.com/topi314/gobin/v2/internal/ezhttp"
	"github.com/topi314/gobin/v2/internal/httperr"
	"github.com/topi314/gobin/v2/server/templates"
)

var (
	ErrDocumentNotFound       = errors.New("document not found")
	ErrDocumentFileNotFound   = errors.New("document file not found")
	ErrInvalidDocumentVersion = errors.New("document version is invalid")
	ErrPreviewsDisabled       = errors.New("document previews disabled")
	ErrRateLimit              = errors.New("rate limit exceeded")
)

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("gobin", otelchi.WithChiRoutes(r)))
	r.Use(middleware.CleanPath)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(slogchi.NewWithConfig(slog.Default(), slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelDebug,
		ServerErrorLevel: slog.LevelError,
		WithRequestID:    true,
		WithSpanID:       s.cfg.Otel != nil,
		WithTraceID:      s.cfg.Otel != nil,
		Filters: []slogchi.Filter{
			slogchi.IgnorePathPrefix("/assets"),
		},
	}))
	r.Use(cacheControl)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/ping"))
	if s.cfg.RateLimit != nil {
		r.Use(s.RateLimit)
	}
	r.Use(s.JWTMiddleware)
	r.Use(middleware.GetHead)

	if s.cfg.Debug {
		middleware.Profiler()
		r.Route("/debug", func(r chi.Router) {
			r.Get("/pprof", pprof.Index)
			r.Get("/cmdline", pprof.Cmdline)
			r.Get("/profile", pprof.Profile)
			r.Get("/symbol", pprof.Symbol)
			r.Get("/trace", pprof.Trace)

			r.Get("/goroutine", pprof.Handler("goroutine").ServeHTTP)
			r.Get("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
			r.Get("/mutex", pprof.Handler("mutex").ServeHTTP)
			r.Get("/heap", pprof.Handler("heap").ServeHTTP)
			r.Get("/block", pprof.Handler("block").ServeHTTP)
			r.Get("/allocs", pprof.Handler("allocs").ServeHTTP)
		})
	}

	var previewCache func(http.Handler) http.Handler
	previewHandler := func(r chi.Router) {
		r.Get("/preview", func(w http.ResponseWriter, r *http.Request) {
			s.error(w, r, httperr.NotFound(ErrPreviewsDisabled))
		})
	}
	if s.cfg.Preview != nil && s.cfg.Preview.CacheSize > 0 && s.cfg.Preview.CacheTTL > 0 {
		previewCache = stampede.HandlerWithKey(s.cfg.Preview.CacheSize, time.Duration(s.cfg.Preview.CacheTTL), s.cacheKeyFunc)
	}
	if s.cfg.Preview != nil {
		previewHandler = func(r chi.Router) {
			r.Route("/preview", func(r chi.Router) {
				if previewCache != nil {
					r.Use(previewCache)
				}
				r.Get("/", s.GetDocumentPreview)
			})
		}
	}

	r.Mount("/assets", http.FileServer(s.assets))
	r.HandleFunc("/assets/theme.css", s.ThemeCSS)
	r.Handle("/favicon.ico", s.file("/assets/favicon.png"))
	r.Handle("/favicon.png", s.file("/assets/favicon.png"))
	r.Handle("/favicon-light.png", s.file("/assets/favicon-light.png"))
	r.Handle("/robots.txt", s.file("/assets/robots.txt"))

	r.Get("/version", s.GetVersion)

	r.Route("/documents", func(r chi.Router) {
		r.Post("/", s.PostDocument)

		filesHandler := func(r chi.Router) {
			r.Route("/files/{fileName}", func(r chi.Router) {
				r.Get("/", s.GetDocumentFile)
			})
		}
		r.Route("/{documentID}", func(r chi.Router) {
			r.Get("/", s.GetDocument)
			r.Patch("/", s.PatchDocument)
			r.Delete("/", s.DeleteDocument)
			r.Post("/share", s.PostDocumentShare)

			r.Route("/versions", func(r chi.Router) {
				r.Get("/", s.DocumentVersions)
				r.Route("/{version}", func(r chi.Router) {
					r.Get("/", s.GetDocument)
					r.Delete("/", s.DeleteDocument)
				})
			})

			r.Route("/webhooks", func(r chi.Router) {
				r.Post("/", s.PostDocumentWebhook)
				r.Route("/{webhookID}", func(r chi.Router) {
					r.Get("/", s.GetDocumentWebhook)
					r.Patch("/", s.PatchDocumentWebhook)
					r.Delete("/", s.DeleteDocumentWebhook)
				})
			})

			filesHandler(r)
		})
	})

	rawFilesHandler := func(r chi.Router) {
		r.Route("/files/{fileName}", func(r chi.Router) {
			r.Get("/", s.GetRawDocumentFile)
		})
	}
	r.Route("/raw/{documentID}", func(r chi.Router) {
		r.Get("/", s.GetRawDocument)
		r.Route("/versions/{version}", func(r chi.Router) {
			r.Get("/", s.GetRawDocument)
			rawFilesHandler(r)
		})
		rawFilesHandler(r)
	})

	r.Route("/{documentID}", func(r chi.Router) {
		r.Get("/", s.GetPrettyDocument)
		previewHandler(r)
		r.Route("/{version}", func(r chi.Router) {
			r.Get("/", s.GetPrettyDocument)
			previewHandler(r)
		})
	})
	r.Get("/", s.GetPrettyDocument)

	r.NotFound(s.redirectRoot)

	if s.cfg.HTTPTimeout > 0 {
		return http.TimeoutHandler(r, time.Duration(s.cfg.HTTPTimeout), "Request timed out")
	}
	return r
}

func (s *Server) GetVersion(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(s.version))
}

func (s *Server) redirectRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) prettyError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	var httpErr *httperr.Error
	if errors.As(err, &httpErr) {
		status = httpErr.Status

		if httpErr.Location != "" {
			http.Redirect(w, r, httpErr.Location, status)
			return
		}
	}

	w.WriteHeader(status)
	if tmplErr := templates.Error(templates.ErrorVars{
		Error:     err.Error(),
		Status:    status,
		RequestID: middleware.GetReqID(r.Context()),
		Path:      r.URL.Path,
	}).Render(r.Context(), w); tmplErr != nil && !errors.Is(tmplErr, http.ErrHandlerTimeout) {
		slog.ErrorContext(r.Context(), "failed to execute error template", tint.Err(tmplErr))
	}
}

func (s *Server) error(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, http.ErrHandlerTimeout) {
		return
	}

	status := http.StatusInternalServerError
	var httpErr *httperr.Error
	if errors.As(err, &httpErr) {
		status = httpErr.Status
	}

	if httpErr.Location != "" {
		http.Redirect(w, r, httpErr.Location, status)
		return
	}

	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "internal server error", tint.Err(err))
	}
	s.json(w, r, ezhttp.ErrorResponse{
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
	w.Header().Set(ezhttp.HeaderContentType, ezhttp.ContentTypeJSON)
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.ErrorContext(r.Context(), "failed to encode json", tint.Err(err))
	}
}

func (s *Server) file(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := s.assets.Open(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = file.Close()
		}()
		if _, err = io.Copy(w, file); err != nil {
			slog.ErrorContext(r.Context(), "failed to copy file", tint.Err(err))
		}
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
