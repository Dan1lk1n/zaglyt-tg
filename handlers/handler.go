package handlers

import (
	"zaglyt-tg/app"
)

type Handler struct {
	app app.App
}

func NewHandler(app app.App) Handler {
	return Handler{
		app: app,
	}
}
