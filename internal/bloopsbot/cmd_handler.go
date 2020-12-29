package bloopsbot

import (
	"bloop/internal/bloopsbot/resource"
	matchstateModel "bloop/internal/database/matchstate/model"
	userDb "bloop/internal/database/user/database"
	userModel "bloop/internal/database/user/model"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
)

func (m *manager) handleBanCommand(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextBanMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCommandCbHandler(u.Id, func(msg string) error {
		msg = strings.TrimPrefix(msg, "@")
		banned, err := m.userDb.FetchByUsername(msg)
		if err != nil {
			if errors.Is(err, userDb.NotFoundErr) {
				if _, err := m.tg.Send(tgbotapi.NewMessage(u.Id, fmt.Sprintf("Пользователь не найден: %s", msg))); err != nil {
					return fmt.Errorf("send msg: %v", err)
				}
			}
			return fmt.Errorf("fetch by username: %v", err)
		}

		if banned.Admin {
			if _, err := m.tg.Send(tgbotapi.NewMessage(chatId, "Нельзя забанить администратора")); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
			return nil
		}

		banned.Status = userModel.StatusBanned
		if err := m.userDb.Store(banned); err != nil {
			return fmt.Errorf("user db store: %v", err)
		}

		if _, err := m.tg.Send(tgbotapi.NewMessage(u.Id, fmt.Sprintf("Пользователь забанен: %s", msg))); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.commandCbHandlers, u.Id)

		return nil
	})

	return nil
}

func (m *manager) handleProfileCmd(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextSendProfileMsg)
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCommandCbHandler(u.Id, func(username string) error {
		username = strings.TrimPrefix(username, "@")

		u, err := m.userDb.FetchByUsername(username)
		if err != nil {
			if errors.Is(err, userDb.NotFoundErr) {
				msg := tgbotapi.NewMessage(chatId, resource.TextProfileCmdUserNotFound)
				msg.ParseMode = tgbotapi.ModeMarkdown
				if _, err := m.tg.Send(msg); err != nil {
					return fmt.Errorf("send msg: %v", err)
				}
				return nil
			}

			return fmt.Errorf("fetch by username: %v", err)
		}

		stat, err := m.statDb.FetchProfileStat(u.Id)
		if err != nil {
			return fmt.Errorf("fetch profile stat: %v", err)
		}

		msg := tgbotapi.NewMessage(chatId, renderProfile(u, stat))
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.commandCbHandlers, u.Id)

		return nil
	})

	return nil
}

func (m *manager) handleFeedbackCommand(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextFeedbackMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCommandCbHandler(u.Id, func(msg string) error {
		if m.config.Admin != "" {
			admin, err := m.userDb.FetchByUsername(m.config.Admin)
			if err != nil {
				return fmt.Errorf("fetch by username: %v", err)
			}

			if _, err := m.tg.Send(tgbotapi.NewMessage(
				admin.Id,
				fmt.Sprintf("Прилетел фидбек от пользователя: %s", msg),
			)); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.commandCbHandlers, u.Id)

		return nil
	})

	return nil
}

func (m *manager) handleRegisterOfflinePlayerCmd(u userModel.User, chatId int64) error {
	if session, ok := m.userMatchSession(u.Id); ok {
		msg := tgbotapi.NewMessage(chatId, resource.TextSendOfflinePlayerUsernameMsg)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		m.registerCommandCbHandler(u.Id, func(username string) error {
			u.FirstName = username

			if err := session.AddPlayer(matchstateModel.NewPlayer(chatId, u, true)); err != nil {
				return err
			}

			msg := tgbotapi.NewMessage(chatId, resource.TextOfflinePlayerAdded)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}

			m.mtx.Lock()
			defer m.mtx.Unlock()
			delete(m.commandCbHandlers, u.Id)

			return nil
		})
	} else {
		msg := tgbotapi.NewMessage(chatId, resource.TextGameRoomNotFound)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
	}

	return nil
}
