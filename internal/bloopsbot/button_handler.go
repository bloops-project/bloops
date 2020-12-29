package bloopsbot

import (
	"bloop/internal/bloopsbot/builder"
	"bloop/internal/bloopsbot/resource"
	matchstateModel "bloop/internal/database/matchstate/model"
	statDb "bloop/internal/database/stat/database"
	userModel "bloop/internal/database/user/model"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
)

func (m *manager) handleStartButton(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf(resource.TextGreetingMsg, u.FirstName))
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, mSessExist := m.userMatchSession(u.Id)
	_, bSessExist := m.userBuildingSession(u.Id)
	if !mSessExist && !bSessExist {
		msg.ReplyMarkup = resource.CommonButtons
	}

	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}
	return nil
}

func (m *manager) handleRulesButton(_ userModel.User, chatId int64) error {
	msgText := resource.TextRulesMsg
	msg := tgbotapi.NewMessage(chatId, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *manager) handleCreateButton(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextSettingsMsg)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(resource.LeaveButton))
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	session, err := builder.NewSession(
		m.tg,
		chatId,
		u.Id,
		u.Username,
		m.builderDoneFn,
		m.builderWarnFn,
		m.config.BuildingTimeout,
	)

	if err != nil {
		return fmt.Errorf("new builder session: %v", err)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.commandCbHandlers, u.Id)
	m.userBuildingSessions[u.Id] = session
	session.Run(m.ctxSess)

	return nil
}

func (m *manager) handleButtonExit(u userModel.User, chatId int64) error {
	if session, ok := m.userMatchSession(u.Id); ok {
		session.RemovePlayer(u.Id)
	}

	m.resetUserSessions(u.Id)

	msg := tgbotapi.NewMessage(chatId, resource.TextLeavingSessionsMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *manager) handleProfileButton(u userModel.User, chatId int64) error {
	stat, err := m.statDb.FetchProfileStat(u.Id)
	if err != nil && !errors.Is(err, statDb.NotFoundErr) {
		return fmt.Errorf("fetch profile stat: %v", err)
	}

	msg := tgbotapi.NewMessage(chatId, renderProfile(u, stat))
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *manager) handleJoinButton(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextSendJoinedCodeMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCommandCbHandler(u.Id, func(msg string) error {
		n, err := strconv.Atoi(msg)
		if err != nil {
			return fmt.Errorf("strconv: %v", err)
		}

		if session, ok := m.matchSession(int64(n)); ok {
			if err := session.AddPlayer(matchstateModel.NewPlayer(chatId, u, false)); err != nil {
				return fmt.Errorf("add player: %v", err)
			}

			greetingText := resource.TextJoinedGameMsg

			row := tgbotapi.NewKeyboardButtonRow()
			if session.Config.AuthorId == u.Id {
				greetingText += resource.TextAuthorGreetingMsg
				row = append(row, resource.StartButton)
			}

			row = append(row, resource.LeaveButton, resource.GameSettingButton)
			msg := tgbotapi.NewMessage(chatId, greetingText)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				row,
				tgbotapi.NewKeyboardButtonRow(resource.RatingButton, resource.RulesButton),
			)

			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}

			m.mtx.Lock()
			m.userMatchSessions[u.Id] = session
			delete(m.commandCbHandlers, u.Id)
			m.mtx.Unlock()
		} else {
			msg := tgbotapi.NewMessage(chatId, resource.TextGameRoomNotFoundMsg)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}

		return nil
	})

	return nil
}
