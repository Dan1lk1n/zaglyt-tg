// /help command
package handlers

import (
	"context"
	"strings"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HelpCommandHandler lists the available commands. Developer-only commands are
// shown only to configured developers.
func (h *Handler) HelpCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	var sb strings.Builder
	sb.WriteString("Команды бота:\n\n")

	sb.WriteString("Для всех (в группах):\n")
	sb.WriteString("/anecdote — сгенерировать анекдот из истории чата\n")
	sb.WriteString("/help — показать это сообщение\n\n")

	sb.WriteString("Для администраторов чата:\n")
	sb.WriteString("/switcher — включить/выключить бота в чате\n")
	sb.WriteString("/settings — показать настройки генерации\n")
	sb.WriteString("/setwords <мин> <макс> — длина ответа в словах\n")
	sb.WriteString("/setprob <процент> — шанс случайного ответа (0–100)\n")
	sb.WriteString("/clear — стереть выученную историю чата\n")
	sb.WriteString("/download — выгрузить историю чата файлом\n")

	if helpers.IsUserDeveloper(update.Message.From.ID, h.cfg.Developers) {
		sb.WriteString("\nДля разработчика:\n")
		sb.WriteString("/whoami — проверить статус разработчика\n")
		sb.WriteString("/stats — статистика по чатам\n")
		sb.WriteString("/broadcast <текст> — рассылка по всем активным чатам\n")
	}

	reply(ctx, b, update, sb.String())
}
