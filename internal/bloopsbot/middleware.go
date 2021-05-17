package bloopsbot

import (
	"fmt"

	userModel "github.com/bloops-games/bloops/internal/database/user/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (m *manager) isAdmin(u userModel.User, chatID int64) (bool, error) {
	if !u.Admin {
		if _, err := m.tg.Send(tgbotapi.NewMessage(chatID, "Для этой команды нужны права администратора")); err != nil {
			return false, fmt.Errorf("send msg: %w", err)
		}

		return false, nil
	}

	return true, nil
}

func (m *manager) isActive(u userModel.User, chatID int64) (bool, error) {
	if !u.Admin && u.Status == userModel.StatusBanned {
		if _, err := m.tg.Send(tgbotapi.NewMessage(chatID, "Бан")); err != nil {
			return false, fmt.Errorf("send msg: %w", err)
		}

		return false, nil
	}

	return true, nil
}
