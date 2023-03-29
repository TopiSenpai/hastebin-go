package main

import (
	"context"
	"embed"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/go-jose/go-jose/v3"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/topisenpai/gobin/gobin"
)

// These variables are set via the -ldflags option in go build
var (
	version   = "unknown"
	commit    = "unknown"
	buildTime = "unknown"
)

var (
	//go:embed templates
	Templates embed.FS

	//go:embed assets
	Assets embed.FS

	//go:embed sql/schema.sql
	Schema string
)

func main() {
	log.Printf("Starting Gobin with version: %s (commit: %s, build time: %s)...", version, commit, buildTime)
	cfgPath := flag.String("config", "", "path to gobin.json")
	flag.Parse()

	viper.SetDefault("listen_addr", ":80")
	viper.SetDefault("dev_mode", false)
	viper.SetDefault("debug", false)
	viper.SetDefault("database_type", "sqlite")
	viper.SetDefault("database_debug", false)
	viper.SetDefault("database_expire_after", "0")
	viper.SetDefault("database_cleanup_interval", "1m")
	viper.SetDefault("database_path", "gobin.db")
	viper.SetDefault("database_host", "localhost")
	viper.SetDefault("database_port", 5432)
	viper.SetDefault("database_username", "gobin")
	viper.SetDefault("database_database", "gobin")
	viper.SetDefault("database_ssl_mode", "disable")
	viper.SetDefault("max_document_size", 0)

	if *cfgPath != "" {
		viper.SetConfigFile(*cfgPath)
	} else {
		viper.SetConfigName("gobin")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/gobin/")
	}
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("Error while reading config:", err)
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("gobin")
	viper.AutomaticEnv()

	var cfg gobin.Config
	if err := viper.Unmarshal(&cfg, func(config *mapstructure.DecoderConfig) {
		config.TagName = "cfg"
	}); err != nil {
		log.Fatalln("Error while unmarshalling config:", err)
	}
	log.Println("Config:", cfg)

	if cfg.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	db, err := gobin.NewDB(context.Background(), cfg.Database, Schema)
	if err != nil {
		log.Fatalln("Error while connecting to database:", err)
	}
	defer db.Close()

	key := jose.SigningKey{
		Algorithm: jose.HS512,
		Key:       []byte(cfg.JWTSecret),
	}
	signer, err := jose.NewSigner(key, nil)
	if err != nil {
		log.Fatalln("Error while creating signer:", err)
	}

	ctx := context.Background()
	exporter, err := newExporter(ctx)
	if err != nil {
		log.Fatalln("Error while creating exporter:", err)
	}

	defer func() {
		if err = exporter.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()
	traceProvider := newTraceProvider(exporter)
	tracer := traceProvider.Tracer("gobin")

	var (
		tmplFunc gobin.ExecuteTemplateFunc
		assets   http.FileSystem
	)
	if cfg.DevMode {
		log.Println("Development mode enabled")
		tmplFunc = func(wr io.Writer, name string, data any) error {
			tmpl, err := template.New("").ParseGlob("templates/*")
			if err != nil {
				return err
			}
			return tmpl.ExecuteTemplate(wr, name, data)
		}
		assets = http.Dir(".")
	} else {
		tmpl, err := template.New("").ParseFS(Templates, "templates/*")
		if err != nil {
			log.Fatalln("Error while parsing templates:", err)
		}
		tmplFunc = tmpl.ExecuteTemplate
		assets = http.FS(Assets)
	}

	styles.Fallback = styles.Get("onedark")
	lexers.Fallback = lexers.Get("plaintext")
	formatters.Register("html", html.New(
		html.WithClasses(true),
		html.ClassPrefix("ch-"),
		html.Standalone(false),
		html.InlineCode(false),
		html.WithNopPreWrapper(),
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, "L"),
		html.TabWidth(4),
	))
	formatters.Register("html-standalone", html.New(
		html.Standalone(true),
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, "L"),
		html.TabWidth(4),
	))

	s := gobin.NewServer(gobin.FormatBuildVersion(version, commit, buildTime), cfg, db, signer, tracer, assets, tmplFunc)
	log.Println("Gobin listening on:", cfg.ListenAddr)
	go s.Start()
	defer s.Close()

	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-si
}
