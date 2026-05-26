package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"zaglyt-tg/configs"
	"zaglyt-tg/handlers"
	"zaglyt-tg/repository"

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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handlers.EchoHandler),
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if nil != err {
		panic(err)
	}

	b.Start(ctx)
}
