package bloopsbot

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bloops-games/bloops/internal/bloopsbot/builder"
	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	matchstateModel "github.com/bloops-games/bloops/internal/database/matchstate/model"
	statDb "github.com/bloops-games/bloops/internal/database/stat/database"
	userModel "github.com/bloops-games/bloops/internal/database/user/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (m *manager) handleRulesButton(_ userModel.User, chatID int64) error {
	msgText := resource.TextRulesMsg
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func (m *manager) handleCreateButton(u userModel.User, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, resource.TextSettingsMsg)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(resource.LeaveButton))
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	session, err := builder.NewSession(
		m.tg,
		chatID,
		u.ID,
		u.Username,
		m.builderDoneFn,
		m.builderWarnFn,
		m.config.BuildingTimeout,
	)
	if err != nil {
		return fmt.Errorf("new builder session: %w", err)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.commandCbHandlers, u.ID)
	m.userBuildingSessions[u.ID] = session
	session.Run(m.ctxSess)

	return nil
}

func (m *manager) handleButtonExit(u userModel.User, chatID int64) error {
	if session, ok := m.userMatchSession(u.ID); ok {
		session.RemovePlayer(u.ID)
	}

	m.resetUserSessions(u.ID)

	msg := tgbotapi.NewMessage(chatID, resource.TextLeavingSessionsMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func (m *manager) handleProfileButton(u userModel.User, chatID int64) error {
	stat, err := m.statDB.FetchProfileStat(u.ID)
	if err != nil && !errors.Is(err, statDb.ErrNotFound) {
		return fmt.Errorf("fetch profile stat: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, renderProfile(u, stat))
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func (m *manager) handleJoinButton(u userModel.User, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, resource.TextSendJoinedCodeMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	m.registerCommandCbHandler(u.ID, func(msg string) error {
		n, err := strconv.Atoi(msg)
		if err != nil {
			return fmt.Errorf("strconv: %w", err)
		}

		if session, ok := m.matchSession(int64(n)); ok {
			if err := session.AddPlayer(matchstateModel.NewPlayer(chatID, u, false)); err != nil {
				return fmt.Errorf("add player: %w", err)
			}

			greetingText := resource.TextJoinedGameMsg

			row := tgbotapi.NewKeyboardButtonRow()
			if session.Config.AuthorID == u.ID {
				greetingText += resource.TextAuthorGreetingMsg
				row = append(row, resource.StartButton)
			}

			row = append(row, resource.LeaveButton, resource.GameSettingButton)
			msg := tgbotapi.NewMessage(chatID, greetingText)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				row,
				tgbotapi.NewKeyboardButtonRow(resource.RatingButton, resource.RulesButton),
			)

			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}

			m.mtx.Lock()
			m.userMatchSessions[u.ID] = session
			delete(m.commandCbHandlers, u.ID)
			m.mtx.Unlock()
		} else {
			msg := tgbotapi.NewMessage(chatID, resource.TextGameRoomNotFoundMsg)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}
		}

		return nil
	})

	return nil
}
