package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-jose/go-jose/v3"
	"github.com/muesli/termenv"
	"github.com/topi314/gomigrate"
	"github.com/topi314/gomigrate/drivers/postgres"
	"github.com/topi314/gomigrate/drivers/sqlite"
	"github.com/topi314/tint"
	"go.gopad.dev/go-tree-sitter-highlight/html"
	meternoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/topi314/gobin/v2/internal/ver"
	"github.com/topi314/gobin/v2/server"
	"github.com/topi314/gobin/v2/server/database"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

// These variables are set via the -ldflags option in go build
var (
	Name      = "gobin"
	Namespace = "github.com/topi314/gobin/v2"

	Version   = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var (
	//go:embed server/assets
	Assets embed.FS

	//go:embed server/migrations
	Migrations embed.FS

	//go:embed languages.toml
	Languages []byte

	//go:embed queries/*
	Queries embed.FS

	//go:embed themes/*
	Themes embed.FS
)

func main() {
	cfgPath := flag.String("config", "gobin.toml", "path to gobin.toml")
	flag.Parse()

	cfg, err := server.LoadConfig(*cfgPath)
	if err != nil {
		slog.Error("Error while loading config", tint.Err(err))
		return
	}

	setupLogger(cfg.Log)
	buildTime, _ := time.Parse(time.RFC3339, BuildTime)
	slog.Info("Starting Gobin...", slog.String("version", Version), slog.String("commit", Commit), slog.Time("build-time", buildTime))
	slog.Info("Config", slog.String("config", cfg.String()))

	var (
		tracer = tracenoop.NewTracerProvider().Tracer(Name)
		meter  = meternoop.NewMeterProvider().Meter(Name)
	)
	if cfg.Otel != nil {
		tracer, err = newTracer(*cfg.Otel)
		if err != nil {
			slog.Error("Error while creating tracer", tint.Err(err))
			return
		}
		meter, err = newMeter(*cfg.Otel)
		if err != nil {
			slog.Error("Error while creating meter", tint.Err(err))
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		slog.Error("Error while connecting to database", tint.Err(err))
		return
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Error while closing database", tint.Err(closeErr))
		}
	}()

	var driver gomigrate.NewDriver
	switch cfg.Database.Type {
	case database.TypePostgres:
		driver = postgres.New
	case database.TypeSQLite:
		driver = sqlite.New
	}

	if err = gomigrate.Migrate(ctx, db, driver, Migrations, gomigrate.WithDirectory("server/migrations")); err != nil {
		slog.Error("Error while migrating database", tint.Err(err))
		return
	}

	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS512,
		Key:       []byte(cfg.JWTSecret),
	}, nil)
	if err != nil {
		slog.Error("Error while creating signer", tint.Err(err))
		return
	}

	var assets http.FileSystem
	if cfg.DevMode {
		slog.Info("Development mode enabled")
		assets = http.Dir("server")
	} else {
		sub, err := fs.Sub(Assets, "server")
		if err != nil {
			slog.Error("Failed to get sub fs for embedded assets", tint.Err(err))
			return
		}
		assets = http.FS(sub)
	}

	if err = server.LoadLanguages(Queries, Languages); err != nil {
		slog.Error("Error while loading languages", tint.Err(err))
		return
	}

	if err = server.LoadThemes(Themes); err != nil {
		slog.Error("Error while loading themes", tint.Err(err))
		return
	}

	htmlRenderer := html.NewRenderer(nil)

	s := server.NewServer(ver.FormatBuildVersion(Version, Commit, buildTime), cfg.DevMode, cfg, db, signer, tracer, meter, assets, htmlRenderer)
	slog.Info("Gobin started...", slog.String("address", cfg.ListenAddr))
	go s.Start()
	defer s.Close()

	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-si
}

func setupLogger(cfg server.LogConfig) {
	var formatter log.Formatter
	switch cfg.Format {
	case server.LogFormatJSON:
		formatter = log.JSONFormatter
	case server.LogFormatText:
		formatter = log.TextFormatter
	case server.LogFormatLogFMT:
		formatter = log.LogfmtFormatter
	default:
		slog.Error("Unknown log format", slog.String("format", string(cfg.Format)))
		os.Exit(-1)
	}

	handler := log.NewWithOptions(os.Stdout, log.Options{
		Level:           log.Level(cfg.Level),
		ReportTimestamp: true,
		ReportCaller:    cfg.AddSource,
		Formatter:       formatter,
	})
	if cfg.Format == server.LogFormatText && !cfg.NoColor {
		handler.SetColorProfile(termenv.TrueColor)
	}

	slog.SetDefault(slog.New(handler))
}
