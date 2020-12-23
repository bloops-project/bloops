package bloops

import (
	"bloop/internal/bloops/builder"
	"bloop/internal/bloops/match"
	"bloop/internal/bloops/resource"
	"bloop/internal/logging"
	statDb "bloop/internal/stat/database"
	userDb "bloop/internal/user/database"
	"bloop/internal/user/model"
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"hash/fnv"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var CommandNotFoundErr = fmt.Errorf("command not found")

func NewManager(tg *tgbotapi.BotAPI, config *Config, userDb *userDb.DB, statDb *statDb.DB) *manager {
	return &manager{
		tg:                   tg,
		config:               config,
		userBuildingSessions: map[int64]*builder.Session{},
		userPlayingSessions:  map[int64]*match.Session{},
		playingSessions:      map[int64]*match.Session{},
		cmdCb:                map[int64]func(string) error{},
		userDb:               userDb,
		statDb:               statDb,
	}
}

type manager struct {
	mtx sync.RWMutex

	ctx    context.Context
	tg     *tgbotapi.BotAPI
	config *Config
	// key: userId
	userBuildingSessions map[int64]*builder.Session
	// key: userId
	userPlayingSessions map[int64]*match.Session
	// key: generated int64 code
	playingSessions map[int64]*match.Session
	// command callbacks
	cmdCb  map[int64]func(string) error
	userDb *userDb.DB
	statDb *statDb.DB
	cancel func()
}

func (m *manager) Stop() {
	m.cancel()
}

func (m *manager) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancel = cancel

	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = int(m.config.TgBotPollTimeout.Seconds())
	updates, err := m.tg.GetUpdatesChan(upd)
	if err != nil {
		return fmt.Errorf("tg get updates chan: %v", err)
	}

	wg := &sync.WaitGroup{}
	poolWorkerNum := runtime.NumCPU()
	wg.Add(poolWorkerNum + 1)

	for i := 0; i < poolWorkerNum; i++ {
		go m.pool(ctx, wg, updates)
	}

	go m.cleaning(ctx, wg)
	wg.Wait()

	return nil
}

func (m *manager) pool(ctx context.Context, wg *sync.WaitGroup, updCh tgbotapi.UpdatesChannel) {
	defer wg.Done()
	logger := logging.FromContext(ctx).Named("manager.pool")
	for {
		select {
		case update := <-updCh:
			u, err := m.recvUser(update)
			if err != nil {
				logger.Errorf("recv user: %v", err)
				return
			}
			if update.Message != nil {
				if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, resource.TextChatNotAllowed)
					msg.ParseMode = tgbotapi.ModeMarkdown
					if _, err := m.tg.Send(msg); err != nil {
						logger.Errorf("send msg: %v", err)
					}
					return
				}
				if err := m.handleCommand(u, update); err != nil {
					logger.Errorf("handle command query: %v", err)
				}
			}
			if update.CallbackQuery != nil {
				if err := m.handleCallbackQuery(u, update); err != nil {
					logger.Errorf("handle callback query: %v", err)
				}
			}
		case <-ctx.Done():
			// shutdown
			return
		}
	}
}

func (m *manager) cleaning(ctx context.Context, wg *sync.WaitGroup) {
	timer := time.NewTicker(1 * time.Minute)
	defer wg.Done()
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// cleaning old building sessions
			m.mtx.RLock()
			for _, session := range m.userBuildingSessions {
				if time.Since(session.CreatedAt) > m.config.BuildingTimeout {
					session.Stop()
				}
			}

			// cleaning old playing sessions
			for _, session := range m.playingSessions {
				if time.Since(session.CreatedAt) > m.config.PlayingTimeout {
					session.Stop()
				}
			}
			m.mtx.RUnlock()
		}
	}
}

func (m *manager) handleCommand(u model.User, upd tgbotapi.Update) error {
	switch upd.Message.Text {
	case resource.CmdStart:
		if err := m.handleStartButton(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle start cmd: %v", err)
		}
	case resource.CmdRules:
		if err := m.handleRulesButton(upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle rules cmd: %v", err)
		}
	case resource.CmdProfile:
		if err := m.handleProfileCmd(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle profile cmd: %v", err)
		}
	case resource.ProfileButtonText:
		if err := m.handleProfileButton(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle profile button: %v", err)
		}
	case resource.CreateButtonText:
		if err := m.handleCreateButton(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle create button: %v", err)
		}
	case resource.JoinButtonText:
		if err := m.handleJoinButton(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle join button: %v", err)
		}
	case resource.LeaveButtonText:
		if err := m.handleBreakButton(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle leave button: %v", err)
		}
	case resource.RuleButtonText:
		if err := m.handleRulesButton(upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle rules button: %v", err)
		}
	case resource.CmdAddPlayer:
		if err := m.handleRegisterOfflinePlayer(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle add offline user cmd: %v", err)
		}
	default:
		if cb, ok := m.callback(u.Id); ok {
			if err := cb(upd.Message.Text); err != nil {
				return fmt.Errorf("execute cb: %v", err)
			}

			return nil
		}

		if session, ok := m.userBuildingSession(u.Id); ok {
			if err := session.Execute(upd); err != nil {
				return fmt.Errorf("execute building session: %v", err)
			}

			return nil
		}

		if session, ok := m.userPlayingSession(u.Id); ok {
			if err := session.Execute(u.Id, upd); err != nil {
				return fmt.Errorf("execute playing session: %v", err)
			}

			return nil
		}
	}

	return nil
}

func (m *manager) handleCallbackQuery(u model.User, upd tgbotapi.Update) error {
	if session, ok := m.userBuildingSession(u.Id); ok {
		if err := session.Execute(upd); err != nil {
			return fmt.Errorf("execute building cb: %v", err)
		}
	}

	if session, ok := m.userPlayingSession(u.Id); ok {
		if err := session.Execute(u.Id, upd); err != nil {
			return fmt.Errorf("execute playing cb: %v", err)
		}
	}

	return nil
}

func (m *manager) hash() (int64, error) {
	h := fnv.New32a()
	bytes, err := time.Now().MarshalBinary()
	if err != nil {
		return 0, fmt.Errorf("hash binary encode error: %v", err)
	}

	_, err = h.Write(bytes)
	if err != nil {
		return 0, fmt.Errorf("hash write error: %w", err)
	}

	return int64(h.Sum32() >> 20), nil
}

func (m *manager) handleRulesButton(chatId int64) error {
	msgText := resource.TextRulesMsg
	msg := tgbotapi.NewMessage(chatId, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *manager) handleStartButton(u model.User, chatId int64) error {
	msgText := fmt.Sprintf(resource.TextGreetingMsg, u.FirstName)
	msg := tgbotapi.NewMessage(chatId, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}
	return nil
}

func (m *manager) buildGameConfig(session *builder.Session, code int64) match.Config {
	config := match.Config{
		Timeout:    m.config.PlayingTimeout,
		Code:       code,
		Tg:         m.tg,
		DoneFn:     m.matchDoneFn,
		AuthorId:   session.AuthorId,
		RoundsNum:  session.RoundsNum,
		RoundTime:  session.RoundTime,
		Bloopses:   []resource.Bloops{},
		Categories: []string{},
		Letters:    []string{},
		Vote:       session.Vote,
		StatDb:     m.statDb,
	}

	for _, category := range session.Categories {
		if category.Status {
			config.Categories = append(config.Categories, category.Text)
		}
	}

	for _, letter := range session.Letters {
		if letter.Status {
			config.Letters = append(config.Letters, letter.Text)
		}
	}

	if session.Bloops {
		config.Bloopses = make([]resource.Bloops, len(resource.Bloopses))
		copy(config.Bloopses, resource.Bloopses)
	}

	return config
}

func (m *manager) builderDoneFn(session *builder.Session) error {
	defer func() {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		session.Stop()
		delete(m.userBuildingSessions, session.AuthorId)
	}()

	code, err := m.hash()
	if err != nil {
		return fmt.Errorf("hash: %v", err)
	}

	for {
		if _, ok := m.playingSession(code); !ok {
			session := match.NewSession(m.buildGameConfig(session, code))
			session.Run(m.ctx)
			m.mtx.Lock()
			m.playingSessions[code] = session
			m.mtx.Unlock()
			break
		}
	}

	msg := tgbotapi.NewMessage(session.ChatId, resource.TextCreationGameCompletedSuccessfulMsg)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	msg = tgbotapi.NewMessage(session.ChatId, strconv.Itoa(int(code)))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *manager) matchDoneFn(session *match.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for _, player := range session.Players {
		delete(m.userPlayingSessions, player.UserId)
	}

	session.Stop()
	delete(m.playingSessions, session.Code)

	return nil
}

func (m *manager) handleCreateButton(u model.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextSettingsMsg)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(resource.LeaveButton))
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	session, err := builder.NewSession(
		m.tg,
		chatId,
		u.Id,
		m.builderDoneFn,
		m.config.BuildingTimeout,
	)

	if err != nil {
		return fmt.Errorf("new builder session: %v", err)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.cmdCb, u.Id)
	m.userBuildingSessions[u.Id] = session
	session.Run(m.ctx)

	return nil
}

func (m *manager) handleBreakButton(u model.User, chatId int64) error {
	if session, ok := m.userPlayingSession(u.Id); ok {
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

func (m *manager) handleProfileButton(u model.User, chatId int64) error {
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

func (m *manager) handleProfileCmd(u model.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextSendProfileMsg)
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCallback(u.Id, func(username string) error {
		defer func() {
			m.mtx.Lock()
			defer m.mtx.Unlock()
			delete(m.cmdCb, u.Id)
		}()

		if strings.Contains(username, "@") {
			username = strings.TrimPrefix(username, "@")
		}

		u, err := m.userDb.FetchByUsername(username)
		if err != nil {
			if errors.Is(err, userDb.NotFoundErr) {
				msg := tgbotapi.NewMessage(chatId, "Пользователь не найден")
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

		return nil
	})

	return nil
}

func (m *manager) handleRegisterOfflinePlayer(u model.User, chatId int64) error {
	if session, ok := m.userPlayingSession(u.Id); ok {
		msg := tgbotapi.NewMessage(chatId, resource.TextSendOfflinePlayerUsernameMsg)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		m.registerCallback(u.Id, func(username string) error {
			defer func() {
				m.mtx.Lock()
				defer m.mtx.Unlock()
				delete(m.cmdCb, u.Id)
			}()

			u.FirstName = username

			if err := session.AddPlayer(match.NewPlayer(chatId, u, true)); err != nil {
				return err
			}

			msg := tgbotapi.NewMessage(chatId, resource.TextOfflinePlayerAdded)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}

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

func (m *manager) handleJoinButton(u model.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextSendJoinedCodeMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCallback(u.Id, func(msg string) error {
		n, err := strconv.Atoi(msg)
		if err != nil {
			return fmt.Errorf("strconv: %v", err)
		}

		if session, ok := m.playingSession(int64(n)); ok {
			if err := session.AddPlayer(match.NewPlayer(chatId, u, false)); err != nil {
				return fmt.Errorf("add player: %v", err)
			}

			greetingText := resource.TextJoinedGameMsg

			row := tgbotapi.NewKeyboardButtonRow()
			if session.AuthorId == u.Id {
				greetingText += resource.TextAuthorGreetingMsg
				row = append(row, resource.StartButton)
			}

			row = append(row, resource.LeaveButton)
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
			m.userPlayingSessions[u.Id] = session
			delete(m.cmdCb, u.Id)
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

func (m *manager) registerCallback(userId int64, fn func(string) error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.cmdCb[userId] = fn
}

func (m *manager) callback(userId int64) (func(msg string) error, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	cb, ok := m.cmdCb[userId]
	return cb, ok
}

func (m *manager) resetUserSessions(userId int64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.userBuildingSessions, userId)
	delete(m.userPlayingSessions, userId)
	delete(m.cmdCb, userId)
}

func (m *manager) userBuildingSession(userId int64) (*builder.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.userBuildingSessions[userId]

	return session, ok
}

func (m *manager) userPlayingSession(userId int64) (*match.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.userPlayingSessions[userId]

	return session, ok
}

func (m *manager) playingSession(code int64) (*match.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.playingSessions[code]

	return session, ok
}

func (m *manager) recvUser(upd tgbotapi.Update) (model.User, error) {
	var tgUser *tgbotapi.User
	var u model.User
	switch {
	case upd.CallbackQuery != nil:
		tgUser = upd.CallbackQuery.From
	case upd.Message != nil:
		tgUser = upd.Message.From
	default:
		return u, CommandNotFoundErr
	}

	u, err := m.userDb.Fetch(int64(tgUser.ID))
	if err != nil {
		if errors.Is(err, userDb.NotFoundErr) {
			newUser := model.User{
				Id:           int64(tgUser.ID),
				FirstName:    tgUser.FirstName,
				LastName:     tgUser.LastName,
				LanguageCode: tgUser.LanguageCode,
				Username:     tgUser.UserName,
				CreatedAt:    time.Now(),
			}

			if err := m.userDb.Store(newUser); err != nil {
				return u, fmt.Errorf("userdb store: %v", err)
			}
			u = newUser
		}
	}

	stat, err := m.statDb.FetchRateStat(u.Id)
	if err != nil {
		if errors.Is(err, statDb.NotFoundErr) {
			return u, nil
		}
		return u, fmt.Errorf("fetch profile stat: %v", err)
	}

	u.Stars = stat.Stars
	u.Bloops = stat.Bloops

	return u, nil
}
