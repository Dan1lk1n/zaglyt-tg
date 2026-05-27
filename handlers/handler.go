package handlers

import (
	"zaglyt-tg/app"

	"github.com/go-telegram/bot/models"
)

type Handler struct {
	app app.App
	bot *models.User
}

func NewHandler(app app.App, bot *models.User) Handler {
	return Handler{
		app: app,
		bot: bot,
	}
}
