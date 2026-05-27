package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"zaglyt-tg/app"
	"zaglyt-tg/configs"
	"zaglyt-tg/handlers"
	"zaglyt-tg/repository"
	"zaglyt-tg/repository/channel"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	cfg, err := configs.LoadConfig()
	if err != nil {
		panic(err)
	}

	db, err := repository.InitDB(cfg)
	if err != nil {
		log.Fatalf("db initialization error: %v", err)
	}
	defer db.Close()

	channelRepo := channel.NewChannelRepository(db)

	app := app.NewApp(channelRepo)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var handler *handlers.Handler

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if handler != nil {
				handler.MessageHandler(ctx, b, update)
			}
		}),
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if err != nil {
		panic(err)
	}

	bot_info, err := b.GetMe(ctx)
	if err != nil {
		log.Fatalf("failed to get bot info: %v", err)
	}

	h := handlers.NewHandler(app, bot_info)
	handler = &h

	//commands
	b.RegisterHandler(bot.HandlerTypeMessageText, "/switcher", bot.MatchTypeExact, handler.SwitcherCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/clear", bot.MatchTypeExact, handler.ClearCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/download", bot.MatchTypeExact, handler.DownloadCommandHandler)

	//admin commands
	b.RegisterHandler(bot.HandlerTypeMessageText, "/whoami", bot.MatchTypeExact, handler.WhoAmICommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, handler.GetBotStatsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/broadcast", bot.MatchTypePrefix, h.BroadcastCommandHandler)

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

	b.Start(ctx)
}
