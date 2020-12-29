package bloopsbot

import (
	"bloop/internal/bloopsbot/builder"
	"bloop/internal/bloopsbot/match"
	"bloop/internal/bloopsbot/resource"
	"bloop/internal/bloopsbot/util"
	stateDb "bloop/internal/database/matchstate/database"
	matchstateModel "bloop/internal/database/matchstate/model"
	statDb "bloop/internal/database/stat/database"
	statModel "bloop/internal/database/stat/model"
	userDb "bloop/internal/database/user/database"
	userModel "bloop/internal/database/user/model"
	"bloop/internal/logging"
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type commandCbHandlerFunc func(string) error
type commandHandlerFunc = func(userModel.User, int64) error
type commandMiddlewareFunc = func(userModel.User, int64) (bool, error)

var (
	TgResponseTypeNotFoundErr = fmt.Errorf("tg response not found")
	CmdTextHandlerNotFoundErr = fmt.Errorf("command text handler not found")
)

func NewManager(tg *tgbotapi.BotAPI, config *Config, userDb *userDb.DB, statDb *statDb.DB, stateDb *stateDb.DB) *manager {
	return &manager{
		tg:                   tg,
		config:               config,
		userBuildingSessions: map[int64]*builder.Session{},
		userMatchSessions:    map[int64]*match.Session{},
		matchSessions:        map[int64]*match.Session{},
		commandCbHandlers:    map[int64]commandCbHandlerFunc{},
		commandHandlers:      map[string]commandHandler{},
		userDb:               userDb,
		statDb:               statDb,
		stateDb:              stateDb,
	}
}

type commandHandler struct {
	commandFn    commandHandlerFunc
	middlewareFn []commandMiddlewareFunc
}

func (t commandHandler) execute(u userModel.User, chatId int64) error {
	for _, f := range t.middlewareFn {
		ok, err := f(u, chatId)
		if err != nil {
			return fmt.Errorf("command handler execute: %v", err)
		}

		if !ok {
			return nil
		}
	}

	return t.commandFn(u, chatId)
}

type manager struct {
	mtx sync.RWMutex

	tg     *tgbotapi.BotAPI
	config *Config
	// key: userId active building session
	userBuildingSessions map[int64]*builder.Session
	// key: userId active playing session
	userMatchSessions map[int64]*match.Session
	// key: generated int64 code
	matchSessions map[int64]*match.Session
	// command callbacks
	commandCbHandlers map[int64]commandCbHandlerFunc
	// command handlers
	commandHandlers map[string]commandHandler

	userDb     *userDb.DB
	statDb     *statDb.DB
	stateDb    *stateDb.DB
	cancel     func()
	ctxSess    context.Context
	cancelSess func()
}

func (m *manager) Stop() {
	m.cancel()
}

func (m *manager) Run(ctx context.Context) error {
	var updates tgbotapi.UpdatesChannel
	ctx, cancel := context.WithCancel(ctx)
	logger := logging.FromContext(ctx)
	m.cancel = cancel
	m.ctxSess, m.cancelSess = context.WithCancel(context.Background())

	if m.config.BotWebhookHookUrl != "" {
		_, err := m.tg.SetWebhook(tgbotapi.NewWebhook(m.config.BotWebhookHookUrl + m.config.BotToken))
		if err != nil {
			return fmt.Errorf("tg bot set webhook: %v", err)
		}

		info, err := m.tg.GetWebhookInfo()
		if err != nil {
			return fmt.Errorf("get webhook info: %v", err)
		}

		if info.LastErrorDate != 0 {
			logger.Errorf("Telegram callback failed: %s", info.LastErrorMessage)
		}

		updates = m.tg.ListenForWebhook("/" + m.config.BotToken)
		go func() {
			if err := http.ListenAndServe(":4444", nil); err != nil {
				logger.Fatalf("listen and serve http stopped: %v", err)
				cancel()
			}
		}()
	} else {
		resp, err := m.tg.RemoveWebhook()
		if err != nil {
			return fmt.Errorf("remove webhook: %v", err)
		}

		if !resp.Ok {
			if resp.ErrorCode > 0 {
				return fmt.Errorf("remove webhook with error code %d and description %s", resp.ErrorCode, resp.Description)
			}
			return fmt.Errorf("remove webhook response not ok=)")
		}

		upd := tgbotapi.NewUpdate(0)
		upd.Timeout = int(m.config.TgBotPollTimeout.Seconds())
		up, err := m.tg.GetUpdatesChan(upd)
		if err != nil {
			return fmt.Errorf("tg get updates chan: %v", err)
		}
		updates = up
	}

	userMiddleware := []commandMiddlewareFunc{m.isActive}
	adminMiddleware := []commandMiddlewareFunc{m.isAdmin}
	// register text command handlers
	m.registerCommandHandler(resource.CmdStart, commandHandler{commandFn: m.handleStartButton, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.CmdFeedback, commandHandler{commandFn: m.handleFeedbackCommand, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.CmdRules, commandHandler{commandFn: m.handleRulesButton, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.CmdProfile, commandHandler{commandFn: m.handleProfileCmd, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.ProfileButtonText, commandHandler{commandFn: m.handleProfileButton, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.CreateButtonText, commandHandler{commandFn: m.handleCreateButton, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.JoinButtonText, commandHandler{commandFn: m.handleJoinButton, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.LeaveButtonText, commandHandler{commandFn: m.handleButtonExit, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.RuleButtonText, commandHandler{commandFn: m.handleRulesButton, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.CmdAddPlayer, commandHandler{commandFn: m.handleRegisterOfflinePlayerCmd, middlewareFn: userMiddleware})
	m.registerCommandHandler(resource.CmdBan, commandHandler{commandFn: m.handleBanCommand, middlewareFn: adminMiddleware})

	// deserialize not completed sessions
	if err := m.deserialize(); err != nil {
		return fmt.Errorf("deserialize: %v", err)
	}

	wg := &sync.WaitGroup{}
	poolWorkerNum := runtime.NumCPU()
	wg.Add(poolWorkerNum)

	for i := 0; i < poolWorkerNum; i++ {
		go m.pool(ctx, wg, updates)
	}

	wg.Wait()
	m.shutdown()
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

				if err := m.route(ctx, u, update); err != nil {
					if !errors.Is(err, match.ValidationErr) {
						logger.Errorf("handle command query: %v", err)
					}
				}
			}

			if update.CallbackQuery != nil {
				if err := m.handleCallbackQuery(ctx, u, update); err != nil {
					logger.Errorf("handle commandCbHandler query: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *manager) route(ctx context.Context, u userModel.User, upd tgbotapi.Update) error {
	logger := logging.FromContext(ctx).Named("bloopsbot.manager.route")
	logger.Infof("Command received from user %s, command %s", u.FirstName, upd.Message.Text)

	if handler, ok := m.commandHandler(upd.Message.Text); ok {
		if err := handler.execute(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("execute command text handler: %v", err)
		}

		return nil
	}

	if cb, ok := m.commandCbHandler(u.Id); ok {
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

	if session, ok := m.userMatchSession(u.Id); ok {
		if err := session.Execute(u.Id, upd); err != nil {
			return fmt.Errorf("execute playing session: %w", err)
		}

		return nil
	}

	return nil
}

func (m *manager) handleCallbackQuery(ctx context.Context, u userModel.User, upd tgbotapi.Update) error {
	logger := logging.FromContext(ctx).Named("bloopsbot.manager.handlerCallbackQuery")
	logger.Infof(
		"Command received from user %s, command %s, data %s",
		u.FirstName,
		upd.CallbackQuery.Message,
		upd.CallbackQuery.Data,
	)

	if session, ok := m.userBuildingSession(u.Id); ok {
		if err := session.Execute(upd); err != nil {
			return fmt.Errorf("execute building cb: %v", err)
		}
	}

	if session, ok := m.userMatchSession(u.Id); ok {
		if err := session.Execute(u.Id, upd); err != nil {
			return fmt.Errorf("execute playing cb: %v", err)
		}
	}

	return nil
}

func (m *manager) buildGameConfig(session *builder.Session, code int64) match.Config {
	config := match.Config{
		Timeout:    m.config.PlayingTimeout,
		Code:       code,
		Tg:         m.tg,
		DoneFn:     m.matchDoneFn,
		WarnFn:     m.matchWarnFn,
		AuthorId:   session.AuthorId,
		AuthorName: session.AuthorName,
		RoundsNum:  session.RoundsNum,
		RoundTime:  session.RoundTime,
		Bloopses:   []resource.Bloops{},
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

	if session.Bloops {
		config.Bloopses = make([]resource.Bloops, len(resource.Bloopses))
		copy(config.Bloopses, resource.Bloopses)
	}

	return config
}

func (m *manager) builderWarnFn(session *builder.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.userBuildingSessions, session.AuthorId)

	return nil
}

func (m *manager) builderDoneFn(session *builder.Session) error {
	defer func() {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		delete(m.userBuildingSessions, session.AuthorId)
	}()

	code, err := util.GenerateCodeHash()
	if err != nil {
		return fmt.Errorf("hash: %v", err)
	}

	for {
		if _, ok := m.matchSession(code); !ok {
			session := match.NewSession(m.buildGameConfig(session, code))
			session.Run(m.ctxSess)
			m.mtx.Lock()
			m.matchSessions[code] = session
			m.mtx.Unlock()
			break
		}
	}

	msg := tgbotapi.NewMessage(session.ChatId, resource.TextCreationGameCompletedSuccessfulMsg)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	if _, err := m.tg.Send(tgbotapi.NewStickerShare(session.ChatId, resource.GenerateSticker(true))); err != nil {
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

func (m *manager) matchWarnFn(session *match.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if err := m.serialize(session); err != nil {
		return fmt.Errorf("serialize match session: %v", err)
	}

	for _, player := range session.Players {
		delete(m.userMatchSessions, player.UserId)
	}

	delete(m.matchSessions, session.Code)

	return nil
}

func (m *manager) matchDoneFn(session *match.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if err := m.appendStat(session); err != nil {
		return fmt.Errorf("append stat: %v", err)
	}
	for _, player := range session.Players {
		delete(m.userMatchSessions, player.UserId)
	}

	session.Stop()
	delete(m.matchSessions, session.Code)

	return nil
}

func (m *manager) registerCommandHandler(cmd string, handler commandHandler) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.commandHandlers[cmd] = handler
}

func (m *manager) commandHandler(cmd string) (commandHandler, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	handler, ok := m.commandHandlers[cmd]
	return handler, ok
}

func (m *manager) registerCommandCbHandler(userId int64, fn commandCbHandlerFunc) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.commandCbHandlers[userId] = fn
}

func (m *manager) commandCbHandler(userId int64) (func(msg string) error, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	cb, ok := m.commandCbHandlers[userId]
	return cb, ok
}

func (m *manager) resetUserSessions(userId int64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.userBuildingSessions, userId)
	delete(m.userMatchSessions, userId)
	delete(m.commandCbHandlers, userId)
}

func (m *manager) userBuildingSession(userId int64) (*builder.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.userBuildingSessions[userId]

	return session, ok
}

func (m *manager) userMatchSession(userId int64) (*match.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.userMatchSessions[userId]

	return session, ok
}

func (m *manager) matchSession(code int64) (*match.Session, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	session, ok := m.matchSessions[code]

	return session, ok
}

func (m *manager) shutdown() {
	var ss int
	m.cancelSess()
	m.mtx.RLock()
	bn, ms := len(m.userBuildingSessions), len(m.matchSessions)
	m.mtx.RUnlock()
	ss = bn + ms
	ticker := time.NewTicker(200 * time.Millisecond)
	for ss > 0 {
		select {
		case <-ticker.C:
			m.mtx.RLock()
			bn, ms := len(m.userBuildingSessions), len(m.matchSessions)
			m.mtx.RUnlock()
			ss = bn + ms
		default:
		}
	}
}

func (m *manager) recvUser(upd tgbotapi.Update) (userModel.User, error) {
	var tgUser *tgbotapi.User
	var u userModel.User
	switch {
	case upd.CallbackQuery != nil:
		tgUser = upd.CallbackQuery.From
	case upd.Message != nil:
		tgUser = upd.Message.From
	default:
		return u, TgResponseTypeNotFoundErr
	}

	u, err := m.userDb.Fetch(int64(tgUser.ID))
	if err != nil {
		if errors.Is(err, userDb.NotFoundErr) {
			username := strings.TrimPrefix(tgUser.UserName, "@")
			adminUsername := m.config.Admin

			newUser := userModel.User{
				Id:           int64(tgUser.ID),
				FirstName:    tgUser.FirstName,
				LastName:     tgUser.LastName,
				LanguageCode: tgUser.LanguageCode,
				Username:     tgUser.UserName,
				Admin:        username == adminUsername,
				Status:       userModel.StatusActive,
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

func NewMatchSessionFromSerialized(
	ser matchstateModel.State,
	tg *tgbotapi.BotAPI,
	doneFn func(session *match.Session) error,
	warnFn func(session *match.Session) error,
) *match.Session {
	c := match.Config{
		AuthorId:   ser.AuthorId,
		AuthorName: ser.AuthorName,
		RoundsNum:  ser.RoundsNum,
		RoundTime:  ser.RoundTime,
		Categories: make([]string, len(ser.Categories)),
		Letters:    make([]string, len(ser.Letters)),
		Bloopses:   make([]resource.Bloops, len(ser.Bloopses)),
		Vote:       ser.Vote,
		Code:       ser.Code,
		Timeout:    ser.Timeout,
		Tg:         tg,
		DoneFn:     doneFn,
		WarnFn:     warnFn,
	}

	copy(c.Categories, ser.Categories)
	copy(c.Letters, ser.Letters)
	copy(c.Bloopses, ser.Bloopses)

	s := match.NewSession(c)
	s.State = ser.State
	s.CurrRoundIdx = ser.CurrRoundIdx
	s.Players = make([]*matchstateModel.Player, len(ser.Players))
	copy(s.Players, ser.Players)
	return s
}

func (m *manager) serialize(session *match.Session) error {
	s := matchstateModel.State{
		Timeout:      session.Config.Timeout,
		AuthorId:     session.Config.AuthorId,
		AuthorName:   session.Config.AuthorName,
		RoundsNum:    session.Config.RoundsNum,
		RoundTime:    session.Config.RoundTime,
		Vote:         session.Config.Vote,
		Code:         session.Config.Code,
		State:        session.State,
		CurrRoundIdx: session.CurrRoundIdx,
		CreatedAt:    session.CreatedAt,
		Categories:   make([]string, len(session.Config.Categories)),
		Letters:      make([]string, len(session.Config.Letters)),
		Bloopses:     make([]resource.Bloops, len(session.Config.Bloopses)),
		Players:      make([]*matchstateModel.Player, len(session.Players)),
	}

	copy(s.Categories, session.Config.Categories)
	copy(s.Letters, session.Config.Letters)
	copy(s.Bloopses, session.Config.Bloopses)
	copy(s.Players, session.Players)

	if err := m.stateDb.Add(s); err != nil {
		return fmt.Errorf("state db add: %v", err)
	}

	return nil
}

func (m *manager) deserialize() error {
	states, err := m.stateDb.FetchAll()
	if err != nil && !errors.Is(err, stateDb.EntryNotFoundErr) {
		return fmt.Errorf("stat db fetch all: %v", err)
	}
	m.mtx.Lock()
	for _, state := range states {
		session := NewMatchSessionFromSerialized(state, m.tg, m.matchDoneFn, m.matchWarnFn)
		session.Run(m.ctxSess)
		m.matchSessions[session.Config.Code] = session
		for _, player := range session.Players {
			if !player.Offline {
				m.userMatchSessions[player.UserId] = session
			}
		}
	}

	for _, session := range m.matchSessions {
		session.MoveState(session.State)
	}

	m.mtx.Unlock()

	if len(states) > 0 {
		if err := m.stateDb.Clean(); err != nil {
			if !errors.Is(err, stateDb.BucketNotFoundErr) {
				return fmt.Errorf("state db clean: %v", err)
			}
		}
	}

	return nil
}

func (m *manager) appendStat(session *match.Session) error {
	favorites := session.Favorites()
	var stats []statModel.Stat

	for _, player := range session.Players {
		stat := statModel.NewStat(player.UserId)
		if player.Offline {
			continue
		}

		for _, score := range favorites {
			if player.UserId == score.Player.UserId {
				stat.Conclusion = statModel.StatusFavorite
			}
		}

		stat.Categories = make([]string, len(session.Config.Categories))
		copy(stat.Categories, session.Config.Categories)

		stat.RoundsNum = session.Config.RoundsNum
		stat.PlayersNum = len(session.Players)

		var bestDuration, worstDuration, sumDuration, durationNum time.Duration = 2 << 31, 0, 0, 0
		var bestPoints, worstPoints, sumPoints, pointsNum = 0, 2 << 31, 0, 0

		for _, rate := range player.Rates {
			if !rate.Bloops {
				durationNum += 1
				if rate.Duration < bestDuration {
					bestDuration = rate.Duration
				}
				if rate.Duration > worstDuration {
					worstDuration = rate.Duration
				}
				sumDuration += rate.Duration
			} else {
				stat.Bloops = append(stat.Bloops, rate.BloopsName)
			}

			pointsNum += 1
			if rate.Points > bestPoints {
				bestPoints = rate.Points
			}
			if rate.Points < worstPoints {
				worstPoints = rate.Points
			}
			sumPoints += rate.Points
		}

		stat.SumPoints = sumPoints
		stat.BestPoints = bestPoints
		stat.WorstPoints = worstPoints

		if pointsNum > 0 {
			stat.AveragePoints = sumPoints / pointsNum
		}

		stat.BestDuration = bestDuration
		stat.WorstDuration = worstDuration

		if durationNum > 0 {
			stat.AverageDuration = sumDuration / durationNum
		}

		stat.SumDuration = sumDuration
		stats = append(stats, stat)
	}

	for _, stat := range stats {
		if err := m.statDb.Add(stat); err != nil {
			return fmt.Errorf("stat db add: %v", err)
		}
	}

	return nil
}
