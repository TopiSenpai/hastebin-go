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

	"github.com/go-jose/go-jose/v3"
	"github.com/mattn/go-colorable"
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

	//go:embed styles
	Styles embed.FS
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

	loadEmbeddedStyles()
	loadLocalStyles(cfg.CustomStyles)

	htmlRenderer := html.NewRenderer(nil)

	s := server.NewServer(ver.FormatBuildVersion(Version, Commit, buildTime), cfg.DevMode, cfg, db, signer, tracer, meter, assets, htmlRenderer)
	slog.Info("Gobin started...", slog.String("address", cfg.ListenAddr))
	go s.Start()
	defer s.Close()

	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-si
}

const (
	ansiFaint         = "\033[2m"
	ansiWhiteBold     = "\033[37;1m"
	ansiYellowBold    = "\033[33;1m"
	ansiCyanBold      = "\033[36;1m"
	ansiCyanBoldFaint = "\033[36;1;2m"
	ansiRedFaint      = "\033[31;2m"
	ansiRedBold       = "\033[31;1m"

	ansiRed     = "\033[31m"
	ansiYellow  = "\033[33m"
	ansiGreen   = "\033[32m"
	ansiMagenta = "\033[35m"
)

func setupLogger(cfg server.LogConfig) {
	var handler slog.Handler
	switch cfg.Format {
	case server.LogFormatJSON:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: cfg.AddSource,
			Level:     cfg.Level,
		})

	case server.LogFormatText:
		handler = tint.NewHandler(colorable.NewColorable(os.Stdout), &tint.Options{
			AddSource: cfg.AddSource,
			Level:     cfg.Level,
			NoColor:   cfg.NoColor,
			LevelColors: map[slog.Level]string{
				slog.LevelDebug: ansiMagenta,
				slog.LevelInfo:  ansiGreen,
				slog.LevelWarn:  ansiYellow,
				slog.LevelError: ansiRed,
			},
			Colors: map[tint.Kind]string{
				tint.KindTime:            ansiYellowBold,
				tint.KindSourceFile:      ansiCyanBold,
				tint.KindSourceSeparator: ansiCyanBoldFaint,
				tint.KindSourceLine:      ansiCyanBold,
				tint.KindMessage:         ansiWhiteBold,
				tint.KindKey:             ansiFaint,
				tint.KindSeparator:       ansiFaint,
				tint.KindValue:           ansiWhiteBold,
				tint.KindErrorKey:        ansiRedFaint,
				tint.KindErrorSeparator:  ansiFaint,
				tint.KindErrorValue:      ansiRedBold,
			},
		})
	default:
		slog.Error("Unknown log format", slog.String("format", string(cfg.Format)))
		os.Exit(-1)
	}
	slog.SetDefault(slog.New(handler))
}

func loadEmbeddedStyles() {
	slog.Info("Loading embedded styles")
}

func loadLocalStyles(stylesDir string) {
	if stylesDir == "" {
		return
	}

	slog.Info("Loading local styles", slog.String("dir", stylesDir))
}
