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

	var messageHandler *handlers.Handler

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if messageHandler != nil {
				messageHandler.MessageHandler(ctx, b, update)
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
	messageHandler = &h

	b.Start(ctx)
}
