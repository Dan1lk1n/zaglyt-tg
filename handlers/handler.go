package handlers

import (
	"log/slog"
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
	// members remembers recent authors so the LLM can address people by name.
	members  *chatMembers
	personas *llm.Personas
}

func NewHandler(app app.App, bot *models.User, cfg *configs.Config) Handler {
	personas, err := llm.LoadPersonas(cfg.LLMPersonasFile)
	if err != nil {
		// Personas only flavour the replies; run without them rather than fail.
		slog.Error("load personas, continuing without them", "err", err)
		personas, _ = llm.LoadPersonas("")
	}
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
			Concurrency:     cfg.LLMConcurrency,
		}),
		members:  newChatMembers(personas),
		personas: personas,
	}
}
