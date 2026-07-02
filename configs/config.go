package configs

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Supported values for BOT_MODE.
const (
	ModePolling = "polling"
	ModeWebhook = "webhook"
)

// Config holds every environment-derived setting for the application.
// It is loaded once at startup and injected wherever it is needed — nothing
// should read os.Getenv outside this package.
type Config struct {
	BotName  string
	BotToken string

	Developers []int64

	// Mode selects how the bot receives updates: "polling" (default) or "webhook".
	Mode string

	// Webhook settings are only used (and validated) when Mode == "webhook".
	// WebhookURL is the base public URL (scheme + host, e.g. https://example.com);
	// WebhookPath is appended to it automatically — see WebhookFullURL.
	WebhookURL                string
	WebhookSecret             string
	WebhookListenAddr         string
	WebhookPath               string
	WebhookIPAllowlistEnabled bool

	DatabaseURL string

	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string

	LogLevel  string
	LogFormat string

	// MystemCacheSize bounds the in-memory morphological-analysis cache by
	// number of entries. <= 0 disables caching.
	MystemCacheSize int
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, relying on process environment")
	}

	developers, err := parseDevelopers(os.Getenv("DEVELOPERS"))
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		BotName:    os.Getenv("BOT_NAME"),
		BotToken:   os.Getenv("BOT_TOKEN"),
		Developers: developers,

		Mode:                      normalizeMode(os.Getenv("BOT_MODE")),
		WebhookURL:                os.Getenv("WEBHOOK_URL"),
		WebhookSecret:             os.Getenv("WEBHOOK_SECRET"),
		WebhookListenAddr:         os.Getenv("WEBHOOK_LISTEN_ADDR"),
		WebhookPath:               os.Getenv("WEBHOOK_PATH"),
		WebhookIPAllowlistEnabled: parseBool(os.Getenv("WEBHOOK_IP_ALLOWLIST_ENABLED"), true),

		DatabaseURL: os.Getenv("DATABASE_URL"),
		Driver:      os.Getenv("DB_DRIVER"),
		Host:        os.Getenv("DB_HOST"),
		Port:        os.Getenv("DB_PORT"),
		User:        os.Getenv("DB_USER"),
		Password:    os.Getenv("DB_PASSWORD"),
		DBName:      os.Getenv("DB_NAME"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		LogFormat:   os.Getenv("LOG_FORMAT"),

		MystemCacheSize: parseInt(os.Getenv("MYSTEM_CACHE_SIZE"), 20000),
	}

	if strings.TrimSpace(cfg.BotToken) == "" {
		return nil, errors.New("BOT_TOKEN is required but empty")
	}

	if cfg.Mode != ModePolling && cfg.Mode != ModeWebhook {
		return nil, fmt.Errorf("BOT_MODE must be %q or %q, got %q", ModePolling, ModeWebhook, cfg.Mode)
	}

	// Webhook-mode defaults and validation.
	if cfg.WebhookListenAddr == "" {
		cfg.WebhookListenAddr = ":8080"
	}
	if cfg.WebhookPath == "" {
		cfg.WebhookPath = "/webhook"
	}
	if !strings.HasPrefix(cfg.WebhookPath, "/") {
		cfg.WebhookPath = "/" + cfg.WebhookPath
	}

	if cfg.Mode == ModeWebhook {
		if strings.TrimSpace(cfg.WebhookURL) == "" {
			return nil, errors.New("WEBHOOK_URL is required when BOT_MODE=webhook")
		}
		// The secret guards the endpoint against forged requests; refusing to
		// start without it prevents accidentally exposing an open webhook.
		if strings.TrimSpace(cfg.WebhookSecret) == "" {
			return nil, errors.New("WEBHOOK_SECRET is required when BOT_MODE=webhook")
		}
	}

	// DATABASE_URL takes precedence: when set, the individual DB_* values
	// are not required and the URL is used directly as the connection DSN.
	if cfg.DatabaseURL == "" {
		if cfg.Host == "" || cfg.Port == "" || cfg.User == "" || cfg.DBName == "" {
			return nil, errors.New("neither DATABASE_URL nor the full set of DB_* values is provided")
		}
	}

	if cfg.Driver == "" {
		cfg.Driver = "postgres"
	}

	return cfg, nil
}

// WebhookFullURL assembles the public webhook URL that Telegram posts to, from
// the base WebhookURL and WebhookPath. This keeps the path defined in one place
// (WEBHOOK_PATH) instead of having to repeat it inside WEBHOOK_URL. A trailing
// slash on the base is trimmed; WebhookPath is already normalized to start with "/".
func (c *Config) WebhookFullURL() string {
	base := strings.TrimRight(strings.TrimSpace(c.WebhookURL), "/")
	return base + c.WebhookPath
}

// normalizeMode lowercases and trims BOT_MODE, defaulting to polling when empty.
func normalizeMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return ModePolling
	}
	return mode
}

// parseBool interprets a boolean env var, returning def when unset/unparsable.
func parseBool(raw string, def bool) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return v
}

// parseInt interprets an integer env var, returning def when unset/unparsable.
func parseInt(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func parseDevelopers(raw string) ([]int64, error) {
	if raw == "" {
		return nil, nil
	}

	fields := strings.Fields(raw)
	developers := make([]int64, 0, len(fields))
	for _, field := range fields {
		id, err := strconv.ParseInt(field, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse developer ID %q: %w", field, err)
		}
		developers = append(developers, id)
	}

	return developers, nil
}
