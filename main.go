package main

import (
	"context"
	"fmt"
	"log"
	"zaglyt-tg/configs"
	"zaglyt-tg/repository"
	"zaglyt-tg/repository/channel"
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

	ctx := context.Background()

	channel_repo := channel.NewChannelRepository(db)

	err = channel_repo.Insert(ctx, 234, false, "default")
	if err != nil {
		panic(err)
	}

	channel, err := channel_repo.GetByChannelID(ctx, 234)
	if err != nil {
		panic(err)
	}

	fmt.Println(channel)

	updated_channel, err := channel_repo.Update(ctx, 234, true, "sigma")
	if err != nil {
		panic(err)
	}

	fmt.Println(updated_channel)
}
