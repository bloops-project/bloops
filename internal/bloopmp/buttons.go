package bloopmp

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	createButtonLabel = "Создать"
	joinButtonLabel   = "Присоединиться"
	playButtonLabel   = "Старт"
	stopButtonLabel   = "Завершить и сохранить"
	breakButtonLabel  = "Выйти"
	runButtonLabel    = "Начать"
)

var (
	createMenuButton  = tgbotapi.NewKeyboardButton(createButtonLabel)
	joinMenuButton    = tgbotapi.NewKeyboardButton(joinButtonLabel)
	playMenuButton    = tgbotapi.NewKeyboardButton(playButtonLabel)
	stopMenuButton    = tgbotapi.NewKeyboardButton(stopButtonLabel)
	breakMenuButton   = tgbotapi.NewKeyboardButton(breakButtonLabel)
	runGameButton     = tgbotapi.NewKeyboardButton(runButtonLabel)
	commonMenuButtons = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			createMenuButton,
			joinMenuButton,
		),
	)
)
