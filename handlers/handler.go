package handlers

import (
	"zaglyt-tg/app"
	"zaglyt-tg/configs"

	"github.com/go-telegram/bot/models"
)

type Handler struct {
	app app.App
	bot *models.User
	cfg *configs.Config
}

func NewHandler(app app.App, bot *models.User, cfg *configs.Config) Handler {
	return Handler{
		app: app,
		bot: bot,
		cfg: cfg,
	}
}
