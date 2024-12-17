package server

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/topi314/gobin/v2/internal/timex"
	"github.com/topi314/gobin/v2/server/database"
)

func LoadConfig(cfgPath string) (Config, error) {
	file, err := os.Open(cfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	cfg := defaultConfig()
	if err = toml.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:     slog.LevelInfo,
			Format:    LogFormatText,
			AddSource: false,
			NoColor:   false,
		},
		Debug:       false,
		DevMode:     false,
		ListenAddr:  ":80",
		HTTPTimeout: timex.Duration(30 * time.Second),
		Database: database.Config{
			Type:            database.TypeSQLite,
			Debug:           false,
			ExpireAfter:     0,
			CleanupInterval: timex.Duration(time.Minute),
			Path:            "gobin.db",
			Host:            "localhost",
			Port:            5432,
			Username:        "gobin",
			Password:        "",
			Database:        "gobin",
			SSLMode:         "disable",
		},
		MaxDocumentSize:  0,
		MaxHighlightSize: 0,
		RateLimit: &RateLimitConfig{
			Requests:  10,
			Duration:  timex.Duration(time.Minute),
			Whitelist: []string{"127.0.0.1"},
			Blacklist: nil,
		},
		JWTSecret: "",
		Preview: &PreviewConfig{
			InkscapePath: "inkscape",
			MaxLines:     0,
			DPI:          120,
			CacheSize:    1024,
			CacheTTL:     timex.Duration(time.Hour),
		},
		Otel: nil,
		Webhook: &WebhookConfig{
			Timeout:       timex.Duration(10 * time.Second),
			MaxTries:      3,
			Backoff:       timex.Duration(time.Second),
			BackoffFactor: 2,
			MaxBackoff:    timex.Duration(5 * time.Minute),
		},
		CustomStyles: "",
		DefaultStyle: "onedark",
	}
}

type Config struct {
	Log              LogConfig        `toml:"log"`
	Debug            bool             `toml:"debug"`
	DevMode          bool             `toml:"dev_mode"`
	ListenAddr       string           `toml:"listen_addr"`
	HTTPTimeout      timex.Duration   `toml:"http_timeout"`
	Database         database.Config  `toml:"database"`
	MaxDocumentSize  int64            `toml:"max_document_size"`
	MaxHighlightSize int              `toml:"max_highlight_size"`
	RateLimit        *RateLimitConfig `toml:"rate_limit"`
	JWTSecret        string           `toml:"jwt_secret"`
	Preview          *PreviewConfig   `toml:"preview"`
	Otel             *OtelConfig      `toml:"otel"`
	Webhook          *WebhookConfig   `toml:"webhook"`
	CustomStyles     string           `toml:"custom_styles"`
	DefaultStyle     string           `toml:"default_style"`
}

func (c Config) String() string {
	return fmt.Sprintf("\n Log: %s\n Debug: %t\n DevMode: %t\n ListenAddr: %s\n HTTPTimeout: %s\n Database: %s\n MaxDocumentSize: %d\n MaxHighlightSize: %d\n RateLimit: %s\n JWTSecret: %s\n Preview: %s\n Otel: %s\n Webhook: %s\n CustomStyles: %s\n DefaultStyle: %s\n",
		c.Log,
		c.Debug,
		c.DevMode,
		c.ListenAddr,
		time.Duration(c.HTTPTimeout),
		c.Database,
		c.MaxDocumentSize,
		c.MaxHighlightSize,
		c.RateLimit,
		strings.Repeat("*", len(c.JWTSecret)),
		c.Preview,
		c.Otel,
		c.Webhook,
		c.CustomStyles,
		c.DefaultStyle,
	)
}

type LogFormat string

const (
	LogFormatJSON   LogFormat = "json"
	LogFormatText   LogFormat = "text"
	LogFormatLogFMT LogFormat = "log-fmt"
)

type LogConfig struct {
	Level     slog.Level `toml:"level"`
	Format    LogFormat  `toml:"format"`
	AddSource bool       `toml:"add_source"`
	NoColor   bool       `toml:"no_color"`
}

func (c LogConfig) String() string {
	return fmt.Sprintf("\n  Level: %s\n  Format: %s\n  AddSource: %t\n  NoColor: %t\n",
		c.Level,
		c.Format,
		c.AddSource,
		c.NoColor,
	)
}

type RateLimitConfig struct {
	Requests  int            `toml:"requests"`
	Duration  timex.Duration `toml:"duration"`
	Whitelist []string       `toml:"whitelist"`
	Blacklist []string       `toml:"blacklist"`
}

func (c RateLimitConfig) String() string {
	return fmt.Sprintf("\n  Requests: %d\n  Duration: %s\n  Whitelist: %v\n  Blacklist: %v",
		c.Requests,
		time.Duration(c.Duration),
		c.Whitelist,
		c.Blacklist,
	)
}

type PreviewConfig struct {
	InkscapePath string         `toml:"inkscape_path"`
	MaxLines     int            `toml:"max_lines"`
	DPI          int            `toml:"dpi"`
	CacheSize    int            `toml:"cache_size"`
	CacheTTL     timex.Duration `toml:"cache_ttl"`
}

func (c PreviewConfig) String() string {
	return fmt.Sprintf("\n  InkscapePath: %s\n  MaxLines: %d\n  DPI: %d\n  CacheSize: %d\n  CacheTTL: %s",
		c.InkscapePath,
		c.MaxLines,
		c.DPI,
		c.CacheSize,
		time.Duration(c.CacheTTL),
	)
}

type OtelConfig struct {
	InstanceID string         `toml:"instance_id"`
	Trace      *TraceConfig   `toml:"trace"`
	Metrics    *MetricsConfig `toml:"metrics"`
}

func (c OtelConfig) String() string {
	return fmt.Sprintf("\n  InstanceID: %s\n  Trace: %s\n  Metrics: %s",
		c.InstanceID,
		c.Trace,
		c.Metrics,
	)
}

type TraceConfig struct {
	Endpoint string `toml:"endpoint"`
	Insecure bool   `toml:"insecure"`
}

func (c TraceConfig) String() string {
	return fmt.Sprintf("\n   Endpoint: %s\n   Insecure: %t",
		c.Endpoint,
		c.Insecure,
	)
}

type MetricsConfig struct {
	ListenAddr string `toml:"listen_addr"`
}

func (c MetricsConfig) String() string {
	return fmt.Sprintf("\n   ListenAddr: %s",
		c.ListenAddr,
	)
}

type WebhookConfig struct {
	Timeout       timex.Duration `toml:"timeout"`
	MaxTries      int            `toml:"max_tries"`
	Backoff       timex.Duration `toml:"backoff"`
	BackoffFactor float64        `toml:"backoff_factor"`
	MaxBackoff    timex.Duration `toml:"max_backoff"`
}

func (c WebhookConfig) String() string {
	return fmt.Sprintf("\n  Timeout: %s\n  MaxTries: %d\n  Backoff: %s\n  BackoffFactor: %f\n  MaxBackoff: %s",
		time.Duration(c.Timeout),
		c.MaxTries,
		time.Duration(c.Backoff),
		c.BackoffFactor,
		time.Duration(c.MaxBackoff),
	)
}
