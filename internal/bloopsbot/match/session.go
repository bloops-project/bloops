package match

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	"github.com/bloops-games/bloops/internal/bloopsbot/util"
	"github.com/bloops-games/bloops/internal/database/matchstate/model"
	"github.com/bloops-games/bloops/internal/logging"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/valyala/fastrand"
)

const bloopsMaxWeight = 3

const (
	generateLetterTimes      = 10
	defaultInactiveFatalTime = 600
	defaultInactiveWarnTime  = 500
	defaultInactiveVoteTime  = 30
)

type QueryCallbackHandlerFn func(query *tgbotapi.CallbackQuery) error

const (
	StateKindWaiting uint8 = iota + 1
	StateKindPlaying
	StateKindProcessing
	StateKindFinished
)

var (
	ErrContextFatalClosed = fmt.Errorf("context closed")
	ErrValidation         = fmt.Errorf("validation errors")
)

func newVote() *vote {
	return &vote{pub: make(chan struct{}, 1)}
}

type PlayerScore struct {
	Player        model.Player
	Points        int
	TotalDuration time.Duration
	MinDuration   time.Duration
	Completed     int
	Rounds        int
}

type vote struct {
	thumbUp   int
	thumbDown int
	pub       chan struct{}
}

func NewSession(config Config) *Session {
	return &Session{
		Config:      config,
		tg:          config.Tg,
		Code:        config.Code,
		stateCh:     make(chan uint8, 1),
		sndCh:       make(chan tgbotapi.Chattable, 10),
		startCh:     make(chan struct{}, 1),
		stopCh:      make(chan struct{}, 1),
		passCh:      make(chan int64, 1),
		State:       StateKindWaiting,
		msgCallback: map[int]QueryCallbackHandlerFn{},
		doneFn:      config.DoneFn,
		warnFn:      config.WarnFn,
		timeout:     config.Timeout,
		CreatedAt:   time.Now(),
	}
}

type Session struct {
	Config Config

	Code      int64
	CreatedAt time.Time

	tg      *tgbotapi.BotAPI
	stateCh chan uint8

	mtx          sync.RWMutex
	msgCallback  map[int]QueryCallbackHandlerFn
	Players      []*model.Player
	CurrRoundIdx int
	State        uint8

	currRoundSeconds int
	bloopsPoints     int

	timeout time.Duration

	doneFn func(session *Session) error
	warnFn func(session *Session) error
	cancel func()

	sndCh      chan tgbotapi.Chattable
	startCh    chan struct{}
	stopCh     chan struct{}
	passCh     chan int64
	sema       sync.Once
	activeVote *vote
}

func (r *Session) Stop() {
	r.cancel()
}

func (r *Session) Run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	r.cancel = cancel
	logger := logging.FromContext(ctx)
	r.sema.Do(func() {
		go r.loop(ctx)
		go r.sendingPool(ctx)
	})
	logger.Infof("The game session created, code: %d, author: %s", r.Config.Code, r.Config.AuthorName)
}

func (r *Session) Favorites() []PlayerScore {
	var favorites []PlayerScore
	var max int

	scores := r.Scores()
	for _, score := range scores {
		if score.Points >= max {
			max = score.Points
			favorites = append(favorites, score)
		}
	}
	return favorites
}

func (r *Session) MoveState(kind uint8) {
	r.stateCh <- kind
}

func (r *Session) ChangeState(kind uint8) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.State = kind
}

func (r *Session) AlivePlayersLen() int {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	var n int
	for _, player := range r.Players {
		if player.IsPlaying() && !player.Offline {
			n++
		}
	}

	return n
}

func (r *Session) Execute(userID int64, upd tgbotapi.Update) error {
	if upd.CallbackQuery != nil {
		if err := r.executeCbQuery(upd.CallbackQuery); err != nil {
			return fmt.Errorf("execute msgCallback query: %w", err)
		}
	}

	if upd.Message != nil {
		if err := r.executeMessageQuery(userID, upd.Message); err != nil {
			return fmt.Errorf("execute message query: %w", err)
		}
	}

	return nil
}

func (r *Session) isPossibleStart(userID int64, cmd string) bool {
	return r.State == StateKindWaiting && cmd == resource.StartButtonText && r.Config.AuthorID == userID
}

func (r *Session) executeMessageQuery(userID int64, query *tgbotapi.Message) error {
	if r.isPossibleStart(userID, query.Text) {
		if player, ok := r.findPlayer(userID); ok {
			msg := tgbotapi.NewMessage(player.ChatID, resource.TextGameStarted)
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(resource.RatingButton, resource.RulesButton),
				tgbotapi.NewKeyboardButtonRow(resource.LeaveMenuButton, resource.GameSettingButton),
			)
			msg.ParseMode = tgbotapi.ModeMarkdown
			if _, err := r.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}
		}

		if err := r.sendStartSticker(); err != nil {
			return fmt.Errorf("send start sticker: %w", err)
		}

		r.asyncBroadcast(resource.TextGameStarted, userID)

		r.stateCh <- StateKindPlaying
	}

	if query.Text == resource.RatingButtonText {
		if player, ok := r.findPlayer(userID); ok {
			msg := tgbotapi.NewMessage(player.ChatID, r.renderScores())
			msg.ParseMode = tgbotapi.ModeMarkdown
			if _, err := r.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}
		}
	}

	if query.Text == resource.GameSettingButtonText {
		if player, ok := r.findPlayer(userID); ok {
			msg := tgbotapi.NewMessage(player.ChatID, r.renderSetting())
			msg.ParseMode = tgbotapi.ModeMarkdown
			if _, err := r.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}
		}
	}

	return nil
}

func (r *Session) executeCbQuery(query *tgbotapi.CallbackQuery) error {
	if cb, ok := r.cbHandler(query.Message.MessageID); ok {
		if err := cb(query); err != nil {
			return fmt.Errorf("msgCallback: %w", err)
		}
		return nil
	}
	return fmt.Errorf("match.Session: msgCallback not found")
}

func (r *Session) registerCbHandler(messageID int, fn QueryCallbackHandlerFn) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.msgCallback[messageID] = fn
}

func (r *Session) cbHandler(messageID int) (QueryCallbackHandlerFn, bool) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	cb, ok := r.msgCallback[messageID]
	return cb, ok
}

func (r *Session) loop(ctx context.Context) {
	logger := logging.FromContext(ctx).Named("match.loop")
	defer r.shutdown(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case state := <-r.stateCh:
			switch state {
			case StateKindFinished:
				r.ChangeState(StateKindFinished)
				logger.Infof("Change state to finished %d, author: %s", r.Config.Code, r.Config.AuthorName)
				r.sendWhoFavoritesMsg()
				logger.Infof("Send favorites %d, author: %s", r.Config.Code, r.Config.AuthorName)
				logger.Infof("The game session is complete %d, author: %s", r.Config.Code, r.Config.AuthorName)
			case StateKindProcessing:
				logger.Infof(
					"Game session %d, author: %s, processing results",
					r.Config.Code,
					r.Config.AuthorName,
				)
				r.ChangeState(StateKindProcessing)
				logger.Infof(
					"Change state to processing %d, author: %s",
					r.Config.Code,
					r.Config.AuthorName,
				)

				if r.Config.RoundsNum == r.CurrRoundIdx+1 {
					r.stateCh <- StateKindFinished
					break
				}

				if r.AlivePlayersLen() == 0 {
					r.stateCh <- StateKindFinished
					break
				}

				r.sendRoundClosed()
				logger.Infof(
					"Send round closed message %d, author: %s",
					r.Config.Code,
					r.Config.AuthorName,
				)
				util.Sleep(3 * time.Second)
				r.nextRound()
				r.stateCh <- StateKindPlaying
			case StateKindPlaying:
				r.ChangeState(StateKindPlaying)
				logger.Infof("The game %d changed its State to playing, author: %s", r.Config.Code, r.Config.AuthorName)
				if err := r.playing(ctx); err != nil {
					if !errors.Is(err, ErrContextFatalClosed) {
						logger.Errorf("playing: %v", err)
						r.sendCrashMsg()
						r.Stop()
					}
				}
			}
		}
	}
}

func (r *Session) sendingPool(ctx context.Context) {
	defer close(r.sndCh)
	wg := &sync.WaitGroup{}
	wg.Add(runtime.NumCPU() / 2)
	for i := 0; i < runtime.NumCPU()/2; i++ {
		r.sendingWorker(ctx, wg)
	}
	wg.Wait()
}

func (r *Session) sendingWorker(ctx context.Context, wg *sync.WaitGroup) {
	logger := logging.FromContext(ctx).Named("match.sendingWorker")
	defer wg.Done()
	for {
		select {
		case msg := <-r.sndCh:
			if _, err := r.tg.Send(msg); err != nil {
				logger.Errorf("send tg: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *Session) shutdown(ctx context.Context) {
	logger := logging.FromContext(ctx).Named("match.shutdown")
	defer func() {
		close(r.startCh)
		close(r.passCh)
		close(r.stopCh)
		close(r.stateCh)
	}()

	if time.Since(r.CreatedAt) <= r.timeout {
		if r.getState() != StateKindFinished {
			r.mtx.RLock()
		OuterLoop:
			for _, player := range r.Players {
				player := player
				if !player.IsPlaying() || player.Offline {
					continue OuterLoop
				}

				msg := tgbotapi.NewMessage(player.ChatID, resource.TextMatchWarnMsg)
				msg.ParseMode = tgbotapi.ModeMarkdown
				if _, err := r.tg.Send(msg); err != nil {
					continue OuterLoop
				}
			}

			r.mtx.RUnlock()

			if err := r.warnFn(r); err != nil {
				logger.Errorf("done function: %v", err)
			}

			return
		}

		if err := r.doneFn(r); err != nil {
			logger.Errorf("done function: %v", err)
		}
	}

	logger.Infof("The game session closed, author: %s", r.Config.AuthorName)
}

func (r *Session) playing(ctx context.Context) error {
	logger := logging.FromContext(ctx).Named("match.Session.playing")
PlayerLoop:
	for {
		// choosing the next player
		player, ok := r.nextPlayer()
		if !ok {
			r.stateCh <- StateKindProcessing
			return nil
		}
		logger.Infof("Next playing %s Game session %d, author: %s", player.User.FirstName, r.Config.Code, r.Config.AuthorName)
		rate := &model.Rate{}

		r.currRoundSeconds = r.Config.RoundTime
		r.bloopsPoints = 0

		// send "next player" asyncBroadcast message
		nextPlayerMsg := fmt.Sprintf(resource.TextNextPlayerMsg, player.FormatFirstName())
		r.syncBroadcast(nextPlayerMsg)

		util.Sleep(2 * time.Second)
		if r.Config.IsBloops() {
			logger.Infof("Checking bloops, game session %d, author: %s", r.Config.Code, r.Config.AuthorName)
			msg := tgbotapi.NewMessage(player.ChatID, "Проверяем, выпадет ли блюпс?")
			if _, err := r.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %w", err)
			}

			messageID, err := r.checkBloopsSendMsg(player)
			if err != nil {
				return fmt.Errorf("send ready set go for bloopses: %w", err)
			}

			if r.dice() {
				logger.Infof(
					"The bloops dropped for %s, game session %d, author: %s",
					player.User.FirstName,
					r.Config.Code,
					r.Config.AuthorName,
				)

				rate.Bloops = true
				msg := tgbotapi.NewDeleteMessage(player.ChatID, messageID)
				if _, err := r.tg.Send(msg); err != nil {
					return fmt.Errorf("send msg: %w", err)
				}

				nextBloops, _ := r.randBloopses()
				r.bloopsPoints = nextBloops.Points
				r.currRoundSeconds = r.Config.RoundTime + nextBloops.Seconds
				bloops := &nextBloops
				rate.BloopsName = bloops.Name

				if err := r.sendDroppedBloopsesMsg(player, bloops); err != nil {
					return fmt.Errorf("send bloopsbot: %w", err)
				}

				logger.Infof(
					"The Bloops is %s for player %s Game session %d, author: %s",
					bloops.Name,
					player.User.FirstName,
					r.Config.Code,
					r.Config.AuthorName,
				)

				timerFatal := time.NewTimer(defaultInactiveFatalTime * time.Second)
				timerWarn := time.NewTimer(defaultInactiveWarnTime * time.Second)
			ChallengeNext:
				for {
					select {
					case <-r.startCh:
						timerWarn.Stop()
						timerFatal.Stop()
						break ChallengeNext
					case <-timerWarn.C:
						timerWarn.Stop()
						r.syncBroadcast(fmt.Sprintf(
							"Игрок %s должен нажать на кнопку Понятно в течение %d сек",
							player.FormatFirstName(),
							defaultInactiveFatalTime-defaultInactiveWarnTime,
						))
					case <-timerFatal.C:
						timerFatal.Stop()
						r.syncBroadcast(fmt.Sprintf(
							"%s не начал раунд в течение %d сек, он пропускает ход",
							player.FormatFirstName(),
							defaultInactiveFatalTime,
						))
						r.RemovePlayer(player.UserID)
						continue PlayerLoop
					case <-ctx.Done():
						return ErrContextFatalClosed
					case userID := <-r.passCh:
						if userID == player.UserID {
							continue PlayerLoop
						}
					}
				}
			} else {
				msg := tgbotapi.NewEditMessageText(player.ChatID, messageID, emoji.GameDie.String()+" Блюпс не выпал")
				if _, err := r.tg.Send(msg); err != nil {
					return fmt.Errorf("send msg: %w", err)
				}
				util.Sleep(1 * time.Second)
			}
		}
		logger.Infof(
			"Sending round start msg for player %s, game session %d, author: %s",
			player.User.FirstName,
			r.Config.Code,
			r.Config.AuthorName,
		)
		// send start button and register start button handler
		if err := r.sendStartMsg(player); err != nil {
			return fmt.Errorf("send start msg: %w", err)
		}

		timerFatal := time.NewTimer(defaultInactiveFatalTime * time.Second)
		timerWarn := time.NewTimer(defaultInactiveWarnTime * time.Second)
	SessionStart:
		for {
			select {
			case <-r.startCh:
				timerWarn.Stop()
				timerFatal.Stop()
				break SessionStart
			case <-timerWarn.C:
				timerWarn.Stop()
				r.syncBroadcast(fmt.Sprintf(
					"Игрок %s должен нажать на кнопку старта в течение %d сек",
					player.FormatFirstName(),
					defaultInactiveFatalTime-defaultInactiveWarnTime,
				))
			case <-timerFatal.C:
				timerFatal.Stop()
				r.syncBroadcast(fmt.Sprintf(
					"%s не начал раунд в течение %d сек, он пропускает ход",
					player.FormatFirstName(),
					defaultInactiveFatalTime,
				))
				r.RemovePlayer(player.UserID)
				continue PlayerLoop
			case <-ctx.Done():
				return ErrContextFatalClosed
			case userID := <-r.passCh:
				if userID == player.UserID {
					continue PlayerLoop
				}
			}
		}

		logger.Infof(
			"Player %s ready, game session %d, author: %s",
			player.User.FirstName,
			r.Config.Code,
			r.Config.AuthorName,
		)
		//  generating the letter that the words begin with
		if err := r.sendLetterMsg(player); err != nil {
			return fmt.Errorf("generate and send letter msg: %w", err)
		}

		logger.Infof(
			"Sending letter for player %s, Game session %d, author: %s",
			player.User.FirstName,
			r.Config.Code,
			r.Config.AuthorName,
		)

		if err := r.sendReadyMsg(player); err != nil {
			return fmt.Errorf("send ready msg: %w", err)
		}

		logger.Infof(
			"Game session %d, author: %s, sending ready set go for player %s",
			r.Config.Code,
			r.Config.AuthorName,
			player.User.FirstName,
		)

		logger.Infof(
			"Game session %d, author: %s, ticker start for player %s",
			r.Config.Code,
			r.Config.AuthorName,
			player.User.FirstName,
		)

		// create ticker. Update player timer every 1sec
		secs, timeSince, err := r.ticker(ctx, player)
		if err != nil {
			return fmt.Errorf("ticker: %w", err)
		}

		logger.Infof(
			"Game session %d, author: %s, player %s push stop or time over",
			r.Config.Code,
			r.Config.AuthorName,
			player.User.FirstName,
		)

		var reward int
		if secs > 0 {
			reward = r.bloopsPoints
		}

		rate.Duration = time.Since(timeSince)
		rate.Points = secs + reward
		rate.Completed = secs > 0
		logger.Infof(
			"Game session %d, author: %s, player get a %d points",
			r.Config.Code,
			r.Config.AuthorName,
			rate.Points,
		)

		// vote features
		if r.Config.Vote {
			logger.Infof(
				"Game session %d, author: %s, vote starting for player %s",
				r.Config.Code,
				r.Config.AuthorName,
				player.User.FirstName,
			)
			if rate.Points <= 0 {
				logger.Infof(
					"Game session %d, author: %s, points < 0, vote cancelled %s",
					r.Config.Code,
					r.Config.AuthorName,
					player.User.FirstName,
				)
				r.syncBroadcast("Игрок не успел справиться с заданием, голосование отменено")
			} else {
				if err := r.votes(ctx, rate); err != nil {
					return fmt.Errorf("votes: %w", err)
				}
			}
		}

		if _, err := r.tg.Send(tgbotapi.NewStickerShare(player.ChatID, resource.GenerateSticker(rate.Points > 0))); err != nil {
			return fmt.Errorf("send sticker: %w", err)
		}

		r.mtx.Lock()
		player.Rates = append(player.Rates, rate)

		//  remove the bloops that played
		if rate.Points > 0 && rate.BloopsName != "" {
			for idx, bloops := range r.Config.Bloopses {
				if bloops.Name == rate.BloopsName {
					r.Config.Bloopses = append(r.Config.Bloopses[:idx], r.Config.Bloopses[idx+1:]...)
				}
			}
		}

		r.mtx.Unlock()

		logger.Infof(
			"Game session %d, author: %s, rate append for player %s",
			r.Config.Code,
			r.Config.AuthorName,
			player.User.FirstName,
		)
		util.Sleep(2 * time.Second)
		// send data on the round players
		r.sndCh <- tgbotapi.NewMessage(player.ChatID, fmt.Sprintf(resource.TextStopPlayerRoundMsg, rate.Points))
		logger.Infof(
			"Game session %d, author: %s, round closed for player %s",
			r.Config.Code,
			r.Config.AuthorName,
			player.User.FirstName,
		)
		r.asyncBroadcast(r.renderPlayerGetPoints(player, rate.Points), player.UserID)
		util.Sleep(5 * time.Second)
	}
}

// updating the player's timer and registering callbacks to stop the timer
func (r *Session) ticker(ctx context.Context, player *model.Player) (int, time.Time, error) {
	secs := r.currRoundSeconds
	messageID, err := r.sendFreezeTimerMsg(player, secs)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("send timer msg: %w", err)
	}

	// register stop button handler
	r.registerCbHandler(messageID, func(query *tgbotapi.CallbackQuery) error {
		if query.Data == resource.TextStopBtnData {
			if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.TextStopBtnDataAnswer)); err != nil {
				return fmt.Errorf("send answer msg: %w", err)
			}

			r.stopCh <- struct{}{}
			r.mtx.Lock()
			defer r.mtx.Unlock()
			delete(r.msgCallback, messageID)
		}

		return nil
	})
	since := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
OuterLoop:
	for {
		select {
		case <-ctx.Done():
			return 0, time.Time{}, ErrContextFatalClosed
		case userID := <-r.passCh:
			if userID == player.UserID {
				break OuterLoop
			}
		case <-r.stopCh:
			break OuterLoop
		case <-ticker.C:
			// subtract 1 second each tick
			secs--

			// updating timer
			if err := r.sendWorkingTimerMsg(player, messageID, secs); err != nil {
				return 0, time.Time{}, fmt.Errorf("update timer msg: %w", err)
			}

			if secs <= 0 {
				break OuterLoop
			}
		}
	}

	return secs, since, nil
}

func (r *Session) votes(ctx context.Context, rate *model.Rate) error {
	// create new active vote
	r.activeVote = newVote()

	// for storing the message id
	voteMessages := map[int64]int{}

	// send vote buttons and register callbacks
	if err := r.sendVotesMsg(voteMessages); err != nil {
		return fmt.Errorf("broadcast vote buttons and register msgCallback: %w", err)
	}

	timer := time.NewTimer(defaultInactiveVoteTime * time.Second)
	defer timer.Stop()
	defer close(r.activeVote.pub)

VoteLoop:
	for {
		select {
		case <-ctx.Done():
			return ErrContextFatalClosed
		case <-timer.C:
			break VoteLoop
		case <-r.activeVote.pub:
			// updating data in the voting buttons
			if err := r.sendChangingVotesMsg(voteMessages); err != nil {
				return fmt.Errorf("broadcast votes: %w", err)
			}
			//  if all active players have voted, then we finish processing the votes
			if r.didEveryoneVote() {
				break VoteLoop
			}
		}
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()
	// deleting all vote callbacks
	for _, messageID := range voteMessages {
		delete(r.msgCallback, messageID)
	}

	if r.activeVote.thumbUp < r.activeVote.thumbDown {
		rate.Points = 0
		rate.Completed = false
	}

	return nil
}

// Calculating the player rating
func (r *Session) Scores() []PlayerScore {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	scores := make([]PlayerScore, len(r.Players))
	for i, player := range r.Players {
		playerScore := PlayerScore{
			Player: *player,
			Rounds: len(player.Rates),
		}

		for _, rate := range player.Rates {
			playerScore.Points += rate.Points
			if rate.Completed {
				playerScore.Completed++
			}

			if rate.Duration < playerScore.MinDuration {
				playerScore.MinDuration = rate.Duration
			}

			playerScore.TotalDuration += rate.Duration
		}

		scores[i] = playerScore
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Points > scores[j].Points
	})

	return scores
}

//  Select a player who hasn't played in this round yet
func (r *Session) nextPlayer() (*model.Player, bool) {
	var players []*model.Player
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	for _, player := range r.Players {
		if player.IsPlaying() && len(player.Rates) <= r.CurrRoundIdx {
			players = append(players, player)
		}
	}

	if len(players) == 0 {
		return nil, false
	}

	rnd := fastrand.Uint32n(uint32(len(players)))
	return players[rnd], true
}

func (r *Session) didEveryoneVote() bool {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	var playersNum int
	for _, player := range r.Players {
		if player.IsPlaying() && !player.Offline {
			playersNum++
		}
	}

	return r.activeVote.thumbUp+r.activeVote.thumbDown == playersNum
}

func (r *Session) findPlayer(userID int64) (*model.Player, bool) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	for _, player := range r.Players {
		if player.UserID == userID {
			return player, true
		}
	}

	return nil, false
}

// register new player and send asyncBroadcast message about it
func (r *Session) AddPlayer(player *model.Player) error {
	if player, ok := r.addPlayer(player); ok {
		registerPlayerMsg := fmt.Sprintf(resource.TextPlayerJoinedGameMsg, player.FormatFirstName())
		r.asyncBroadcast(registerPlayerMsg, player.UserID)
	}

	return nil
}

// create and append new player with State "Playing"
func (r *Session) addPlayer(player *model.Player) (*model.Player, bool) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for _, p := range r.Players {
		if p.ChatID == player.ChatID && p.UserID == player.UserID && p.FormatFirstName() == player.FormatFirstName() {
			if player.State == model.PlayerStateKindLeaving {
				player.State = model.PlayerStateKindPlaying
				return nil, true
			}

			return nil, false
		}
	}

	r.Players = append(r.Players, player)

	return player, true
}

// remove player from game and send asyncBroadcast message about it
func (r *Session) RemovePlayer(userID int64) {
	player, ok := r.findPlayer(userID)
	if ok {
		r.asyncBroadcast(fmt.Sprintf(resource.TextPlayerLeftGameMsg, player.FormatFirstName()))
		r.removePlayer(userID)
		if r.AlivePlayersLen() == 0 && r.getState() == StateKindFinished {
			r.Stop()
			return
		}
		r.passCh <- player.UserID
	}
}

// set PlayerStateKindLeaving status
func (r *Session) removePlayer(userID int64) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	for _, p := range r.Players {
		if p.UserID == userID {
			p.State = model.PlayerStateKindLeaving
		}
	}
}

// change vote condition and publish changes

func (r *Session) thumbUp() {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.activeVote.thumbUp++
	r.activeVote.pub <- struct{}{}
}

func (r *Session) thumbDown() {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.activeVote.thumbDown++
	r.activeVote.pub <- struct{}{}
}

func (r *Session) dice() bool {
	return (int)(fastrand.Uint32n(10)+1) < 7
}

func (r *Session) randBloopses() (resource.Bloops, bool) {
	if len(r.Config.Bloopses) == 0 {
		return resource.Bloops{}, false
	}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 3; i++ {
		rand.Shuffle(len(r.Config.Bloopses), func(i, j int) {
			r.Config.Bloopses[i], r.Config.Bloopses[j] = r.Config.Bloopses[j], r.Config.Bloopses[i]
		})
	}

	return r.Config.Bloopses[0], true
}

// nolint
func (r *Session) randWeightedBloopses() resource.Bloops {
	var max float64 = -1
	var result resource.Bloops
	var mn, mx uint32

	for mn == mx {
		p1, p2 := fastrand.Uint32n(uint32(len(r.Config.Bloopses))), fastrand.Uint32n(uint32(len(r.Config.Bloopses)))
		if p1 > p2 {
			mx, mn = p1, p2
		} else {
			mn, mx = p1, p2
		}
	}

	for _, challenge := range r.Config.Bloopses[mn:mx] {
		rndNum := float64(fastrand.Uint32n(bloopsMaxWeight)) / bloopsMaxWeight
		rnd := math.Pow(rndNum, 1/float64(challenge.Weight))
		if rnd > max {
			max = rnd
			result = challenge
		}
	}

	return result
}

func (r *Session) getState() uint8 {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	return r.State
}

func (r *Session) nextRound() {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.CurrRoundIdx++
}

func (r *Session) syncBroadcast(msg string, exclude ...int64) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
OuterLoop:
	for _, player := range r.Players {
		player := player
		if !player.IsPlaying() || player.Offline {
			continue OuterLoop
		}

		for i := range exclude {
			if player.UserID == exclude[i] {
				continue OuterLoop
			}
		}

		msg := tgbotapi.NewMessage(player.ChatID, msg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			continue OuterLoop
		}
	}
}

func (r *Session) asyncBroadcast(msg string, exclude ...int64) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
OuterLoop:
	for _, player := range r.Players {
		player := player
		if !player.IsPlaying() || player.Offline {
			continue OuterLoop
		}

		for i := range exclude {
			if player.UserID == exclude[i] {
				continue OuterLoop
			}
		}

		msg := tgbotapi.NewMessage(player.ChatID, msg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		r.sndCh <- msg
	}
}
