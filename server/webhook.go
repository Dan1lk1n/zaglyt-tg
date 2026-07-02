package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"zaglyt-tg/configs"
	"zaglyt-tg/middlewares"

	"github.com/go-telegram/bot"
)

// RunWebhook serves the Telegram webhook endpoint and a health endpoint,
// blocking until ctx is cancelled. It runs the bot's internal webhook worker
// (b.StartWebhook) alongside the HTTP server and shuts the server down
// gracefully on ctx.Done.
//
// The webhook endpoint is protected by:
//   - the secret token check performed inside b.WebhookHandler() (configured via
//     bot.WithWebhookSecretToken),
//   - POST-only + body-size limits,
//   - an optional Telegram IP allowlist (cfg.WebhookIPAllowlistEnabled).
func RunWebhook(ctx context.Context, b *bot.Bot, cfg *configs.Config) error {
	// b.WebhookHandler() validates the X-Telegram-Bot-Api-Secret-Token header.
	var webhook http.Handler = b.WebhookHandler()

	// Order: allowlist (outermost) → method/size → secret-token handler.
	webhook = middlewares.PostOnly(webhook)
	if cfg.WebhookIPAllowlistEnabled {
		// trustForwardedFor is false: match against the real RemoteAddr. Behind a
		// reverse proxy, disable the allowlist (WEBHOOK_IP_ALLOWLIST_ENABLED=false)
		// or terminate it at the proxy, since RemoteAddr would be the proxy's IP.
		webhook = middlewares.TelegramIPAllowlist(false)(webhook)
		slog.Info("webhook IP allowlist enabled (Telegram ranges only)")
	} else {
		slog.Warn("webhook IP allowlist disabled; relying on secret token only")
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.WebhookPath, webhook)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              cfg.WebhookListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Bot's internal worker that consumes updates delivered to WebhookHandler.
	go b.StartWebhook(ctx)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("webhook HTTP server listening",
			"addr", cfg.WebhookListenAddr, "path", cfg.WebhookPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
