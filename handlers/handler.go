package handlers

import (
	"time"
	"zaglyt-tg/app"
	"zaglyt-tg/configs"
	"zaglyt-tg/modules/llm"

	"github.com/go-telegram/bot/models"
)

type Handler struct {
	app app.App
	bot *models.User
	cfg *configs.Config
	// llm is nil when LLM_URL is not set.
	llm *llm.Client
}

func NewHandler(app app.App, bot *models.User, cfg *configs.Config) Handler {
	return Handler{
		app: app,
		bot: bot,
		cfg: cfg,
		llm: llm.New(llm.Config{
			URL:             cfg.LLMURL,
			Model:           cfg.LLMModel,
			SystemPrompt:    cfg.LLMSystemPrompt,
			Timeout:         time.Duration(cfg.LLMTimeoutSeconds) * time.Second,
			MaxTokens:       cfg.LLMMaxTokens,
			MaxPromptChars:  cfg.LLMMaxPromptChars,
			ContextMessages: cfg.LLMContextMessages,
		}),
	}
}
