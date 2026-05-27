package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"
	"zaglyt-tg/app"
	"zaglyt-tg/configs"
	"zaglyt-tg/handlers"
	"zaglyt-tg/repository"
	"zaglyt-tg/repository/channel"

	"github.com/go-telegram/bot"
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

	channel_repo := channel.NewChannelRepository(db)

	app := app.NewApp(channel_repo)

	handler := handlers.NewHandler(app)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	rand.Seed(time.Now().UnixNano())

	opts := []bot.Option{
		bot.WithDefaultHandler(handler.MessageHandler),
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if nil != err {
		panic(err)
	}

	b.Start(ctx)
}
