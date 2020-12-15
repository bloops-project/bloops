package game

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"time"
)

func thumbUpButton(n int) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d %s", n, textThumbUp), textThumbUp)
}

func thumbDownButton(n int) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d %s", n, textThumbDown), textThumbDown)
}

func sleep(t time.Duration) {
	timer := time.NewTimer(t)
	defer timer.Stop()
	<-timer.C
}
