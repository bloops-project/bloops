package bloopsbot

import (
	"bloop/internal/bloopsbot/builder"
	"bloop/internal/bloopsbot/match"
	"bloop/internal/bloopsbot/resource"
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
	"hash/fnv"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var CommandNotFoundErr = fmt.Errorf("command not found")

func NewManager(tg *tgbotapi.BotAPI, config *Config, userDb *userDb.DB, statDb *statDb.DB, stateDb *stateDb.DB) *manager {
	return &manager{
		tg:                   tg,
		config:               config,
		userBuildingSessions: map[int64]*builder.Session{},
		userPlayingSessions:  map[int64]*match.Session{},
		playingSessions:      map[int64]*match.Session{},
		cmdCb:                map[int64]func(string) error{},
		userDb:               userDb,
		statDb:               statDb,
		stateDb:              stateDb,
	}
}

type manager struct {
	mtx sync.RWMutex

	tg     *tgbotapi.BotAPI
	config *Config
	// key: userId active building session
	userBuildingSessions map[int64]*builder.Session
	// key: userId active playing session
	userPlayingSessions map[int64]*match.Session
	// key: generated int64 code
	playingSessions map[int64]*match.Session
	// command callbacks
	cmdCb      map[int64]func(string) error
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
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.ctxSess, m.cancelSess = context.WithCancel(context.Background())
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = int(m.config.TgBotPollTimeout.Seconds())
	updates, err := m.tg.GetUpdatesChan(upd)
	if err != nil {
		return fmt.Errorf("tg get updates chan: %v", err)
	}

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
		session := NewFromSerialized(state, m.tg, m.matchDoneFn, m.matchWarnFn)
		session.Run(m.ctxSess)
		m.playingSessions[session.Config.Code] = session
		for _, player := range session.Players {
			m.userPlayingSessions[player.UserId] = session
		}
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

func (m *manager) shutdown() {
	var ss int
	m.cancelSess()
	m.mtx.RLock()
	bn, ms := len(m.userBuildingSessions), len(m.playingSessions)
	m.mtx.RUnlock()
	ss = bn + ms
	ticker := time.NewTicker(200 * time.Millisecond)
	for ss > 0 {
		select {
		case <-ticker.C:
			m.mtx.RLock()
			bn, ms := len(m.userBuildingSessions), len(m.playingSessions)
			m.mtx.RUnlock()
			ss = bn + ms
		default:
		}
	}
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

type HandleBtnFn = func()

func (m *manager) handleCommand(u userModel.User, upd tgbotapi.Update) error {
	switch upd.Message.Text {
	case resource.CmdStart:
		if err := m.handleStartButton(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle start cmd: %v", err)
		}
	case resource.CmdFeedback:
		if err := m.handleFeedbackCommand(u, upd.Message.Chat.ID); err != nil {
			return fmt.Errorf("handle feedback command: %v", err)
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
		if err := m.handleButtonExit(u, upd.Message.Chat.ID); err != nil {
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

func (m *manager) handleCallbackQuery(u userModel.User, upd tgbotapi.Update) error {
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

func (m *manager) handleStartButton(u userModel.User, chatId int64) error {
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

	code, err := m.hash()
	if err != nil {
		return fmt.Errorf("hash: %v", err)
	}

	for {
		if _, ok := m.playingSession(code); !ok {
			session := match.NewSession(m.buildGameConfig(session, code))
			session.Run(m.ctxSess)
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

func (m *manager) matchWarnFn(session *match.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if err := m.serialize(session); err != nil {
		return fmt.Errorf("serialize match session: %v", err)
	}

	for _, player := range session.Players {
		delete(m.userPlayingSessions, player.UserId)
	}

	delete(m.playingSessions, session.Code)

	return nil
}

func (m *manager) matchDoneFn(session *match.Session) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if err := m.appendStat(session); err != nil {
		return fmt.Errorf("append stat: %v", err)
	}
	for _, player := range session.Players {
		delete(m.userPlayingSessions, player.UserId)
	}

	session.Stop()
	delete(m.playingSessions, session.Code)

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
	delete(m.cmdCb, u.Id)
	m.userBuildingSessions[u.Id] = session
	session.Run(m.ctxSess)

	return nil
}

func (m *manager) handleButtonExit(u userModel.User, chatId int64) error {
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

func (m *manager) handleProfileCmd(u userModel.User, chatId int64) error {
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

		return nil
	})

	return nil
}

func (m *manager) handleRegisterOfflinePlayer(u userModel.User, chatId int64) error {
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

			if err := session.AddPlayer(matchstateModel.NewPlayer(chatId, u, true)); err != nil {
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

func (m *manager) handleFeedbackCommand(u userModel.User, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, resource.TextFeedbackMsg)
	msg.ReplyMarkup = resource.CommonButtons
	if _, err := m.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	m.registerCallback(u.Id, func(msg string) error {
		defer func() {
			m.mtx.Lock()
			defer m.mtx.Unlock()
			delete(m.cmdCb, u.Id)
		}()

		if m.config.Admin != "" {
			admin, err := m.userDb.FetchByUsername(m.config.Admin)
			if err != nil {
				return fmt.Errorf("fetch by username: %v", err)
			}
			if _, err := m.tg.Send(tgbotapi.NewMessage(admin.Id, fmt.Sprintf("Прилетел фидбек от пользователя: %s", msg))); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}

		return nil
	})

	return nil
}

func (m *manager) handleJoinButton(u userModel.User, chatId int64) error {
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

func (m *manager) recvUser(upd tgbotapi.Update) (userModel.User, error) {
	var tgUser *tgbotapi.User
	var u userModel.User
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
			username := strings.TrimPrefix(tgUser.UserName, "@")
			adminUsername := m.config.Admin

			newUser := userModel.User{
				Id:           int64(tgUser.ID),
				FirstName:    tgUser.FirstName,
				LastName:     tgUser.LastName,
				LanguageCode: tgUser.LanguageCode,
				Username:     tgUser.UserName,
				Admin:        username == adminUsername,
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

func NewFromSerialized(
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
