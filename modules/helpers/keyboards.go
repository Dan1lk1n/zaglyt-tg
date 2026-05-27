package helpers

import (
	"github.com/go-telegram/bot/models"
)

func GetSwitcherKeyboard(enabled bool) *models.InlineKeyboardMarkup {
	textEnable := "Включить"
	textDisable := "• Выключен •"

	if enabled {
		textEnable = "• Включен •"
		textDisable = "Выключить"
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         textEnable,
					CallbackData: "bot_enable",
				},
				{
					Text:         textDisable,
					CallbackData: "bot_disable",
				},
			},
		},
	}
}

func GetClearKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "Я уверен.",
					CallbackData: "bot_clear",
				},
			},
		},
	}
}
