package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"zaglyt-tg/app"
	"zaglyt-tg/configs"
	"zaglyt-tg/handlers"
	"zaglyt-tg/middlewares"
	"zaglyt-tg/modules/logger"
	"zaglyt-tg/modules/z3abp"
	"zaglyt-tg/repository"
	"zaglyt-tg/repository/channel"
	"zaglyt-tg/repository/message"
	"zaglyt-tg/server"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	cfg, err := configs.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.LogFormat)

	db, err := repository.InitDB(cfg)
	if err != nil {
		slog.Error("db initialization error", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		slog.Error("db migration error", "err", err)
		os.Exit(1)
	}

	if err := z3abp.Init(cfg.MystemCacheSize); err != nil {
		slog.Error("mystem initialization error", "err", err)
		os.Exit(1)
	}
	defer z3abp.Mystem.Close()

	channelRepo := channel.NewChannelRepository(db)
	messageRepo := message.NewMessageRepository(db)
	application := app.NewApp(channelRepo, messageRepo)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var handler *handlers.Handler

	// Update types the bot needs. my_chat_member is NOT in Telegram's default
	// set, so it must be requested explicitly (for both polling and webhook) to
	// enforce who may add the bot to chats.
	allowedUpdates := []string{"message", "callback_query", "my_chat_member"}

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if handler == nil {
				return
			}
			if update.MyChatMember != nil {
				handler.MyChatMemberHandler(ctx, b, update)
				return
			}
			handler.MessageHandler(ctx, b, update)
		}),
		bot.WithMiddlewares(middlewares.RecoveryMiddleware),
		bot.WithAllowedUpdates(bot.AllowedUpdates(allowedUpdates)),
	}

	// In webhook mode the library validates the secret token sent by Telegram
	// in the X-Telegram-Bot-Api-Secret-Token header on every incoming request.
	if cfg.Mode == configs.ModeWebhook {
		opts = append(opts, bot.WithWebhookSecretToken(cfg.WebhookSecret))
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if err != nil {
		slog.Error("failed to create bot", "err", err)
		os.Exit(1)
	}

	botInfo, err := b.GetMe(ctx)
	if err != nil {
		slog.Error("failed to get bot info", "err", err)
		os.Exit(1)
	}

	h := handlers.NewHandler(application, botInfo, cfg)
	handler = &h

	//commands
	b.RegisterHandler(bot.HandlerTypeMessageText, "/switcher", bot.MatchTypePrefix, handler.SwitcherCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/clear", bot.MatchTypePrefix, handler.ClearCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/download", bot.MatchTypePrefix, handler.DownloadCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/anecdote", bot.MatchTypePrefix, handler.GenerateAnecdoteCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix, handler.HelpCommandHandler)

	// per-chat settings. Prefix matching tolerates "/cmd@botname" and trailing
	// arguments; the three prefixes are distinct (none prefixes another).
	b.RegisterHandler(bot.HandlerTypeMessageText, "/settings", bot.MatchTypePrefix, handler.SettingsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/setwords", bot.MatchTypePrefix, handler.SetWordsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/setprob", bot.MatchTypePrefix, handler.SetProbabilityCommandHandler)

	//admin commands
	b.RegisterHandler(bot.HandlerTypeMessageText, "/whoami", bot.MatchTypeExact, handler.WhoAmICommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, handler.GetBotStatsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/broadcast", bot.MatchTypePrefix, handler.BroadcastCommandHandler)

	// callbacks
	b.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		"bot_clear",
		bot.MatchTypePrefix,
		handler.CallbackClear,
	)
	b.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		"bot_",
		bot.MatchTypePrefix,
		handler.CallbackBotSwitcher,
	)

	slog.Info("bot started", "username", botInfo.Username, "mode", cfg.Mode)

	switch cfg.Mode {
	case configs.ModeWebhook:
		ok, err := b.SetWebhook(ctx, &bot.SetWebhookParams{
			URL:            cfg.WebhookFullURL(),
			SecretToken:    cfg.WebhookSecret,
			AllowedUpdates: allowedUpdates,
		})
		if err != nil || !ok {
			slog.Error("failed to set webhook", "err", err, "ok", ok)
			os.Exit(1)
		}
		// Remove the webhook on shutdown so Telegram stops delivering to a dead
		// endpoint (uses a fresh context because ctx is already cancelled here).
		defer func() {
			if _, err := b.DeleteWebhook(context.Background(), &bot.DeleteWebhookParams{}); err != nil {
				slog.Error("failed to delete webhook", "err", err)
			}
		}()

		if err := server.RunWebhook(ctx, b, cfg); err != nil {
			slog.Error("webhook server error", "err", err)
			os.Exit(1)
		}
	default:
		b.Start(ctx)
	}
}
