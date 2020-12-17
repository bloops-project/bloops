package bloop

import (
	"bloop/internal/bloop/builder"
	"bloop/internal/bloop/game"
	"bloop/internal/cache"
	"bloop/internal/logging"
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"hash/fnv"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func NewManager(tg *tgbotapi.BotAPI, userCache cache.Cache, config *Config) *Manager {
	return &Manager{
		tg:                   tg,
		userCache:            userCache,
		config:               config,
		userBuildingSessions: map[int64]*builder.Session{},
		userPlayingSessions:  map[int64]*game.Session{},
		playingSessions:      map[int64]*game.Session{},
		cmdCb:                map[int64]func(string) error{},
	}
}

type Manager struct {
	mtx sync.RWMutex

	ctx       context.Context
	tg        *tgbotapi.BotAPI
	config    *Config
	userCache cache.Cache
	// key: userId
	userBuildingSessions map[int64]*builder.Session
	// key: userId
	userPlayingSessions map[int64]*game.Session
	// key: generated int64 code
	playingSessions map[int64]*game.Session
	cmdCb           map[int64]func(string) error // @TODO memory leak
	cancel          func()
}

func (m *Manager) Stop() {
	m.cancel()
}

func (m *Manager) Run(ctx context.Context) error {
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

func (m *Manager) pool(ctx context.Context, wg *sync.WaitGroup, updCh tgbotapi.UpdatesChannel) {
	defer wg.Done()
	logger := logging.FromContext(ctx).Named("manager.pool")
	for {
		select {
		case update := <-updCh:
			user := m.recvUser(update)
			if update.Message != nil {
				if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, textChatNotAllowed)
					msg.ParseMode = tgbotapi.ModeMarkdown
					if _, err := m.tg.Send(msg); err != nil {
						logger.Errorf("send msg: %v", err)
					}
					return
				}
				if err := m.handleCommand(user, update); err != nil {
					logger.Errorf("handle command query: %v", err)
				}
			}
			if update.CallbackQuery != nil {
				if err := m.handleCallbackQuery(user, update); err != nil {
					logger.Errorf("handle callback query: %v", err)
				}
			}
		case <-ctx.Done():
			// shutdown
			return
		}
	}
}

func (m *Manager) cleaning(ctx context.Context, wg *sync.WaitGroup) {
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

func (m *Manager) handleCommand(user *User, upd tgbotapi.Update) error {
	switch upd.Message.Text {
	case cmdStart:
		if err := m.handleStartButton(user, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle start cmd: %v", err)
		}
	case cmdRules:
		if err := m.handleRulesButton(upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle rules cmd: %v", err)
		}
	case createButtonLabel:
		if err := m.handleCreateButton(user, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle create button: %v", err)
		}
	case joinButtonLabel:
		if err := m.handleJoinButton(user, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle join button: %v", err)
		}
	case leaveButtonLabel:
		if err := m.handleBreakButton(user, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle leave button: %v", err)
		}
	case rulesLabel:
		if err := m.handleRulesButton(upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle rules button: %v", err)
		}
	case cmdAddPlayer:
		if err := m.handleRegisterOfflinePlayer(user, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle add offline user cmd: %v", err)
		}
	default:
		if cb, ok := m.commandCb(user.Id); ok {
			if err := cb(upd.Message.Text); err != nil {
				return fmt.Errorf("execute cb: %v", err)
			}

			return nil
		}

		if session, ok := m.userBuildingSession(user.Id); ok {
			if err := session.Execute(upd); err != nil {
				return fmt.Errorf("execute building session: %v", err)
			}

			return nil
		}

		if session, ok := m.userPlayingSession(user.Id); ok {
			if err := session.Execute(user.Id, upd); err != nil {
				return fmt.Errorf("execute playing session: %v", err)
			}

			return nil
		}
	}

	return nil
}

func (m *Manager) handleCallbackQuery(user *User, upd tgbotapi.Update) error {
	if session, ok := m.userBuildingSessions[user.Id]; ok {
		if err := session.Execute(upd); err != nil {
			return fmt.Errorf("execute building cb: %v", err)
		}
	}

	if session, ok := m.userPlayingSessions[user.Id]; ok {
		if err := session.Execute(user.Id, upd); err != nil {
			return fmt.Errorf("execute playing cb: %v", err)
		}
	}

	return nil
}

func (m *Manager) hash() (int64, error) {
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

func (m *Manager) handleRulesButton(chatId int64) error {
	msgText := textRulesMsg
	msg := tgbotapi.NewMessage(chatId, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *Manager) handleStartButton(user *User, chatId int64) error {
	msgText := fmt.Sprintf(textGreetingMsg, user.FirstName)
	msg := tgbotapi.NewMessage(chatId, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}
	return nil
}

func (m *Manager) buildGameConfig(session *builder.Session, code int64) game.Config {
	config := game.Config{
		Timeout:    m.config.PlayingTimeout,
		Code:       code,
		Tg:         m.tg,
		DoneFn:     m.playingDoneFn,
		AuthorId:   session.AuthorId,
		RoundsNum:  session.RoundsNum,
		Categories: []string{},
		Letters:    []string{},
		Vote:       session.Vote,
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

	return config
}

func (m *Manager) buildingDoneFn(session *builder.Session) error {
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
			session := game.NewGame(m.buildGameConfig(session, code))
			session.Run(m.ctx)
			m.mtx.Lock()
			m.playingSessions[code] = session
			m.mtx.Unlock()
			break
		}
	}

	msg := tgbotapi.NewMessage(session.ChatId, textCreationGameCompletedSuccessfulMsg)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	msg = tgbotapi.NewMessage(session.ChatId, strconv.Itoa(int(code)))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *Manager) playingDoneFn(session *game.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for _, player := range session.Players {
		delete(m.userPlayingSessions, player.UserId)
	}

	session.Stop()
	delete(m.playingSessions, session.Code)

	return nil
}

func (m *Manager) handleCreateButton(user *User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, textSettingsMsg)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(leaveMenuButton))
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	session, err := builder.NewSession(
		m.tg,
		chatId,
		user.Id,
		m.buildingDoneFn,
		m.config.BuildingTimeout,
	)

	if err != nil {
		return fmt.Errorf("new session: %v", err)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.userBuildingSessions[user.Id] = session
	session.Run(m.ctx)

	return nil
}

func (m *Manager) handleBreakButton(user *User, chatId int64) error {
	if session, ok := m.userPlayingSession(user.Id); ok {
		session.RemovePlayer(user.Id)
	}

	m.resetUserSessions(user.Id)

	msg := tgbotapi.NewMessage(chatId, textLeavingSessionsMsg)
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (m *Manager) handleRegisterOfflinePlayer(u *User, chatId int64) error {
	if session, ok := m.userPlayingSession(u.Id); ok {
		msg := tgbotapi.NewMessage(chatId, textSendOfflinePlayerUsernameMsg)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		m.mtx.Lock()
		defer m.mtx.Unlock()

		m.cmdCb[u.Id] = func(username string) error {
			defer func() {
				m.mtx.Lock()
				defer m.mtx.Unlock()
				delete(m.cmdCb, u.Id)
			}()

			if err := session.AddPlayer(game.NewPlayer(chatId, u.Id, username, true)); err != nil {
				return err
			}

			msg := tgbotapi.NewMessage(chatId, textOfflinePlayerAdded)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}

			return nil
		}
	} else {
		msg := tgbotapi.NewMessage(chatId, textGameRoomNotFound)
		if _, err := m.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
	}

	return nil
}

func (m *Manager) handleJoinButton(u *User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, textSendJoinedCodeMsg)
	msg.ReplyMarkup = commonMenuButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.cmdCb[u.Id] = func(msg string) error {
		n, err := strconv.Atoi(msg)
		if err != nil {
			return fmt.Errorf("strconv: %v", err)
		}

		if session, ok := m.playingSession(int64(n)); ok {
			if err := session.AddPlayer(game.NewPlayer(chatId, u.Id, u.FirstName, false)); err != nil {
				return fmt.Errorf("add player: %v", err)
			}

			greetingText := textJoinedGameMsg

			row := tgbotapi.NewKeyboardButtonRow()
			if session.AuthorId == u.Id {
				greetingText += textAuthorGreetingMsg
				row = append(row, startGameButton)
			}

			row = append(row, leaveMenuButton)
			msg := tgbotapi.NewMessage(chatId, greetingText)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(row, tgbotapi.NewKeyboardButtonRow(ratingButton))

			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}

			m.mtx.Lock()
			m.userPlayingSessions[u.Id] = session
			delete(m.cmdCb, u.Id)
			m.mtx.Unlock()
		} else {
			msg := tgbotapi.NewMessage(chatId, textGameRoomNotFoundMsg)
			if _, err := m.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}

		return nil
	}

	return nil
}

func (m *Manager) commandCb(userId int64) (func(msg string) error, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	cb, ok := m.cmdCb[userId]
	return cb, ok
}

func (m *Manager) resetUserSessions(userId int64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.userBuildingSessions, userId)
	delete(m.userPlayingSessions, userId)
}

func (m *Manager) userBuildingSession(userId int64) (*builder.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.userBuildingSessions[userId]

	return session, ok
}

func (m *Manager) userPlayingSession(userId int64) (*game.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.userPlayingSessions[userId]

	return session, ok
}

func (m *Manager) playingSession(code int64) (*game.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.playingSessions[code]

	return session, ok
}

func (m *Manager) fetchTgUser(tgUser *tgbotapi.User) *User {
	u, ok := m.userCache.Get(int64(tgUser.ID))
	if !ok {
		newUser := NewUser(tgUser)
		m.userCache.Add(newUser.Id, newUser)
		u = newUser
	}

	return u.(*User)
}

func (m *Manager) recvUser(upd tgbotapi.Update) *User {
	var (
		user   *User
		tgUser *tgbotapi.User
	)

	if upd.CallbackQuery != nil {
		tgUser = upd.CallbackQuery.From
	}

	if upd.Message != nil {
		tgUser = upd.Message.From
	}

	if tgUser != nil {
		return m.fetchTgUser(tgUser)
	}

	return user
}

func NewUser(usr *tgbotapi.User) *User {
	return &User{
		Id:           int64(usr.ID),
		FirstName:    usr.FirstName,
		LastName:     usr.LastName,
		LanguageCode: usr.LanguageCode,
		Username:     usr.UserName,
	}
}

type User struct {
	Id           int64
	FirstName    string
	LastName     string
	LanguageCode string
	Username     string
}
