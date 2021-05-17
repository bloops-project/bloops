package bloopsbot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	matchstateModel "github.com/bloops-games/bloops/internal/database/matchstate/model"
	userDb "github.com/bloops-games/bloops/internal/database/user/database"
	userModel "github.com/bloops-games/bloops/internal/database/user/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (m *manager) handleStartCommand(u userModel.User, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(resource.TextGreetingMsg, u.FirstName))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = resource.CommonButtons

	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}
	return nil
}

func (m *manager) handleBanCommand(u userModel.User, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, resource.TextBanMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	m.registerCommandCbHandler(u.ID, func(msg string) error {
		msg = strings.TrimPrefix(msg, "@")
		banned, err := m.userDB.FetchByUsername(msg)
		if err != nil {
			if errors.Is(err, userDb.ErrNotFound) {
				if _, err := m.tg.Send(tgbotapi.NewMessage(u.ID, fmt.Sprintf("Пользователь не найден: %s", msg))); err != nil {
					return fmt.Errorf("send msg: %w", err)
				}
			}
			return fmt.Errorf("fetch by username: %w", err)
		}

		if banned.Admin {
			if _, err := m.tg.Send(tgbotapi.NewMessage(chatID, "Нельзя забанить администратора")); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}
			return nil
		}

		banned.Status = userModel.StatusBanned
		if err := m.userDB.Store(banned); err != nil {
			return fmt.Errorf("user db store: %w", err)
		}

		if _, err := m.tg.Send(tgbotapi.NewMessage(u.ID, fmt.Sprintf("Пользователь забанен: %s", msg))); err != nil {
			return fmt.Errorf("send msg: %w", err)
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.commandCbHandlers, u.ID)

		return nil
	})

	return nil
}

func (m *manager) handleProfileCmd(u userModel.User, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, resource.TextSendProfileMsg)
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	m.registerCommandCbHandler(u.ID, func(username string) error {
		username = strings.TrimPrefix(username, "@")

		u, err := m.userDB.FetchByUsername(username)
		if err != nil {
			if errors.Is(err, userDb.ErrNotFound) {
				msg := tgbotapi.NewMessage(chatID, resource.TextProfileCmdUserNotFound)
				msg.ParseMode = tgbotapi.ModeMarkdown
				if _, err := m.tg.Send(msg); err != nil {
					return fmt.Errorf("send msg: %w", err)
				}
				return nil
			}

			return fmt.Errorf("fetch by username: %w", err)
		}

		stat, err := m.statDB.FetchProfileStat(u.ID)
		if err != nil {
			return fmt.Errorf("fetch profile stat: %w", err)
		}

		msg := tgbotapi.NewMessage(chatID, renderProfile(u, stat))
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %w", err)
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.commandCbHandlers, u.ID)

		return nil
	})

	return nil
}

func (m *manager) handleFeedbackCommand(u userModel.User, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, resource.TextFeedbackMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	m.registerCommandCbHandler(u.ID, func(msg string) error {
		if m.config.Admin != "" {
			admin, err := m.userDB.FetchByUsername(m.config.Admin)
			if err != nil {
				return fmt.Errorf("fetch by username: %w", err)
			}

			if _, err := m.tg.Send(tgbotapi.NewMessage(
				admin.ID,
				fmt.Sprintf("Прилетел фидбек от пользователя: %s", msg),
			)); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.commandCbHandlers, u.ID)

		return nil
	})

	return nil
}

func (m *manager) handleRegisterOfflinePlayerCmd(u userModel.User, chatID int64) error {
	if session, ok := m.userMatchSession(u.ID); ok {
		msg := tgbotapi.NewMessage(chatID, resource.TextSendOfflinePlayerUsernameMsg)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %w", err)
		}

		m.registerCommandCbHandler(u.ID, func(username string) error {
			u.FirstName = username

			if err := session.AddPlayer(matchstateModel.NewPlayer(chatID, u, true)); err != nil {
				return err
			}

			msg := tgbotapi.NewMessage(chatID, resource.TextOfflinePlayerAdded)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}

			m.mtx.Lock()
			defer m.mtx.Unlock()
			delete(m.commandCbHandlers, u.ID)

			return nil
		})
	} else {
		msg := tgbotapi.NewMessage(chatID, resource.TextGameRoomNotFound)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %w", err)
		}
	}

	return nil
}
