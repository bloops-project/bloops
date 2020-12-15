package game

import (
	"bloop/internal/bloop/strpool"
	"bloop/internal/logging"
	"context"
	"fmt"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/valyala/fastrand"
	"golang.org/x/sync/errgroup"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	generateLetterTimes      = 10
	defaultRoundTime         = 30
	defaultInactiveFatalTime = 600
	defaultInactiveWarnTime  = 100
	defaultInactiveVoteTime  = 30
)

type cbFn func(query *tgbotapi.CallbackQuery) error

type PlayerStateKind uint8

const (
	PlayerStateKindPlaying PlayerStateKind = iota + 1
	PlayerStateKindLeaving
)

func NewPlayer(chatId, userId int64, firstName string, offline bool) *Player {
	return &Player{
		UserId:    userId,
		ChatId:    chatId,
		FirstName: firstName,
		Rates:     []*Rate{},
		State:     PlayerStateKindPlaying,
		Offline:   offline,
	}
}

type Player struct {
	State     PlayerStateKind
	Offline   bool
	FirstName string
	UserId    int64
	ChatId    int64
	Rates     []*Rate
}

func (r *Player) IsPlaying() bool {
	return r.State == PlayerStateKindPlaying
}

type stateKind uint8

const (
	stateKindWaiting stateKind = iota + 1
	stateKindPlaying
	stateKindProcessing
	stateKindFinished
)

func newVote() *vote {
	return &vote{pub: make(chan struct{}, 1)}
}

type vote struct {
	thumbUp   int
	thumbDown int
	pub       chan struct{}
}

type Rate struct {
	Duration  time.Duration
	Points    int
	Completed bool
}

type PlayerScore struct {
	Player        Player
	Points        int
	TotalDuration time.Duration
	MinDuration   time.Duration
	Completed     int
	Rounds        int
}

type Config struct {
	AuthorId   int64
	RoundsNum  int
	Categories []string
	Letters    []string
	Vote       bool
	Tg         *tgbotapi.BotAPI
	Code       int64
	DoneFn     func(session *Session) error
	Timeout    time.Duration
}

func NewGame(config Config) *Session {
	return &Session{
		config:    config,
		AuthorId:  config.AuthorId,
		tg:        config.Tg,
		Code:      config.Code,
		stateCh:   make(chan stateKind, 1),
		state:     stateKindWaiting,
		cb:        map[int]cbFn{},
		sndCh:     make(chan tgbotapi.Chattable, 10),
		doneFn:    config.DoneFn,
		timeout:   config.Timeout,
		startCh:   make(chan struct{}, 1),
		stopCh:    make(chan struct{}, 1),
		passCh:    make(chan struct{}, 1),
		CreatedAt: time.Now(),
	}
}

type Session struct {
	mtx sync.RWMutex

	config       Config
	Players      []*Player
	CreatedAt    time.Time
	AuthorId     int64
	Code         int64
	tg           *tgbotapi.BotAPI
	stateCh      chan stateKind
	cb           map[int]cbFn
	currRoundIdx int
	timeout      time.Duration
	doneFn       func(session *Session) error
	cancel       func()
	sndCh        chan tgbotapi.Chattable
	startCh      chan struct{}
	stopCh       chan struct{}
	passCh       chan struct{}
	sema         sync.Once
	state        stateKind
	activeVote   *vote
}

func (r *Session) Stop() {
	r.cancel()
}

func (r *Session) Run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	r.cancel = cancel
	r.sema.Do(func() {
		go r.loop(ctx)
		for i := 0; i < runtime.NumCPU(); i++ {
			go r.sendingPool(ctx)
		}
	})
}

func (r *Session) Execute(userId int64, upd tgbotapi.Update) error {
	if upd.CallbackQuery != nil {
		if err := r.executeCbQuery(upd.CallbackQuery); err != nil {
			return fmt.Errorf("execute cb query: %v", err)
		}
	}

	if upd.Message != nil {
		if err := r.executeMessageQuery(userId, upd.Message); err != nil {
			return fmt.Errorf("execute message query: %v", err)
		}
	}

	return nil
}

func (r *Session) executeMessageQuery(userId int64, query *tgbotapi.Message) error {
	if r.isPossibleStart(userId, query.Text) {
		if len(r.Players) < 1 {
			r.asyncBroadcast(textValidationRequiresMoreOnePlayerMsg)
			return nil
		}

		r.stateCh <- stateKindPlaying
	}

	if query.Text == commandRatingText {
		if player, ok := r.findPlayer(userId); ok {
			msg := tgbotapi.NewMessage(player.ChatId, r.renderScores())
			msg.ParseMode = tgbotapi.ModeMarkdown
			if _, err := r.tg.Send(msg); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}
	}

	return nil
}

func (r *Session) executeCbQuery(query *tgbotapi.CallbackQuery) error {
	if cb, ok := r.cb[query.Message.MessageID]; ok {
		if err := cb(query); err != nil {
			return fmt.Errorf("cb: %v", err)
		}
	}
	return nil
}

func (r *Session) loop(ctx context.Context) {
	logger := logging.FromContext(ctx).Named("game.loop")
	defer close(r.startCh)
	defer close(r.passCh)
	defer close(r.stopCh)
	for {
		select {
		case <-ctx.Done():
			if err := r.doneFn(r); err != nil {
				logger.Errorf("done function: %v", err)
			}
			return
		case state := <-r.stateCh:
			switch state {
			case stateKindFinished:
				r.changeState(stateKindFinished)
				r.sendGameFavoriteMsg()

				sleep(5 * time.Second)

				r.syncBroadcast(r.renderScores())
				r.syncBroadcast(textGameClosedMsg)
				r.cancel()
			case stateKindProcessing:
				r.changeState(stateKindProcessing)
				// считаем результаты раунда
				if r.config.RoundsNum == r.currRoundIdx+1 {
					r.stateCh <- stateKindFinished
					break
				}

				if r.ActivePlayersLen() == 0 {
					r.stateCh <- stateKindFinished
					break
				}

				r.sendRoundClosed()
				sleep(5 * time.Second)
				r.nextRound()
				r.stateCh <- stateKindPlaying
			case stateKindPlaying:
				r.changeState(stateKindPlaying)
				if err := r.playing(ctx); err != nil {
					logger.Error(fmt.Errorf("playing: %v", err))
					r.sendCrashMsg()
					r.cancel()
				}
			}
		}
	}
}

func (r *Session) sendCrashMsg() {
	r.syncBroadcast(textBroadcastCrashMsg)
}

func (r *Session) sendingPool(ctx context.Context) {
	logger := logging.FromContext(ctx).Named("game.sendingPool")
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
			if player.UserId == exclude[i] {
				continue OuterLoop
			}
		}

		msg := tgbotapi.NewMessage(player.ChatId, msg)
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
			if player.UserId == exclude[i] {
				continue OuterLoop
			}
		}

		msg := tgbotapi.NewMessage(player.ChatId, msg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		r.sndCh <- msg
	}
}

func (r *Session) isPossibleStart(userId int64, cmd string) bool {
	return r.state == stateKindWaiting && cmd == commandStartText && r.AuthorId == userId
}

func (r *Session) renderScores() string {
	var tmpl string

	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	strBuf.WriteString(emoji.Trophy.String())
	strBuf.WriteString(" ")
	strBuf.WriteString(textLeaderboardHeader)

	var medalPlaceEmoji = func(n int) string {
		var medal string
		switch n {
		case 0:
			medal = emoji.FirstPlaceMedal.String()
		case 1:
			medal = emoji.SecondPlaceMedal.String()
		case 2:
			medal = emoji.ThirdPlaceMedal.String()
		default:
		}

		return medal
	}

	for n, cell := range r.Scores() {
		if n <= 2 {
			tmpl = fmt.Sprintf(
				textLeaderboardMedal,
				n+1,
				medalPlaceEmoji(n),
				cell.Player.FirstName,
				cell.Points,
				len(cell.Player.Rates),
			)
		} else {
			tmpl = fmt.Sprintf(textLeaderboardLine, n+1, cell.Player.FirstName, cell.Points, len(cell.Player.Rates))
		}

		strBuf.WriteString(tmpl)
	}

	return strBuf.String()
}

func (r *Session) sendGameFavoriteMsg() {
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	var (
		favorites []PlayerScore
		max       int
	)

	scores := r.Scores()
	for _, score := range scores {
		if score.Points >= max {
			max = score.Points
			favorites = append(favorites, score)
		}
	}

	strBuf.WriteString(emoji.ChequeredFlag.String())
	strBuf.WriteString(" Игра завершена\n\n")
	strBuf.WriteString("*Список победителей*\n\n")
	for _, score := range favorites {
		strBuf.WriteString(fmt.Sprintf("%s %s - %d очков\n", emoji.Star.String(), score.Player.FirstName, score.Points))
	}

	r.syncBroadcast(strBuf.String())
}

func (r *Session) sendRoundClosed() {
	r.syncBroadcast(fmt.Sprintf(textRoundFavoriteMsg, r.currRoundIdx+1))
}

// notification of the player's readiness and sending the start button
func (r *Session) sendStartMsg(player *Player) error {
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	strBuf.WriteString(fmt.Sprintf("Игрок *%s*\n\n", player.FirstName))
	strBuf.WriteString(emoji.GameDie.String())
	strBuf.WriteString(" Готов сыграть?\n\n")
	strBuf.WriteString("Нужно назвать все слова из списка категорий на выпавшую букву\n\n")
	strBuf.WriteString(emoji.Pen.String())
	strBuf.WriteString(" ")
	strBuf.WriteString(strconv.Itoa(len(r.config.Categories)))
	strBuf.WriteString(" слов\n")
	strBuf.WriteString(emoji.Stopwatch.String())
	strBuf.WriteString(" ")
	strBuf.WriteString(strconv.Itoa(defaultRoundTime))
	strBuf.WriteString(" секунд\n\n")
	strBuf.WriteString(emoji.CardIndex.String())
	strBuf.WriteString(" ")
	strBuf.WriteString("Список категорий:\n\n")
	strBuf.WriteString(r.buildCategoriesStr())
	strBuf.WriteString("\n\n")
	strBuf.WriteString(textClickStartBtnMsg)

	msg := tgbotapi.NewMessage(player.ChatId, strBuf.String())
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(textStartBtnData, textStartBtnData),
		),
	)
	msg.ParseMode = tgbotapi.ModeMarkdown
	output, err := r.tg.Send(msg)
	if err != nil {
		return fmt.Errorf("end msg: %v", err)
	}

	r.cb[output.MessageID] = func(query *tgbotapi.CallbackQuery) error {
		defer func() {
			r.mtx.Lock()
			defer r.mtx.Unlock()
			delete(r.cb, output.MessageID)
		}()

		if query.Data == textStartBtnData {
			if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, textStartBtnDataAnswer)); err != nil {
				return fmt.Errorf("send answer: %v", err)
			}

			r.startCh <- struct{}{}
		}

		return nil
	}

	return nil
}

func (r *Session) buildCategoriesStr() string {
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	for i, category := range r.config.Categories {
		strBuf.WriteString(fmt.Sprintf("%d. %s", i+1, category))
		if i != len(r.config.Categories)-1 {
			strBuf.WriteString("\n")
		}
	}

	return strBuf.String()
}

// send ready -> set -> go steps
func (r *Session) sendReadyMsg(player *Player) error {
	var messageId int
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	strBuf.WriteString(emoji.Keycap3.String())
	strBuf.WriteString(" ...")
	{
		msg := tgbotapi.NewMessage(player.ChatId, strBuf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown

		output, err := r.tg.Send(msg)
		if err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
		messageId = output.MessageID
		sleep(1 * time.Second)
	}

	strBuf.Reset()
	strBuf.WriteString(emoji.Keycap2.String())
	strBuf.WriteString(" На старт")
	{
		msg := tgbotapi.NewEditMessageText(player.ChatId, messageId, strBuf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		sleep(1 * time.Second)
	}

	strBuf.Reset()
	strBuf.WriteString(emoji.Keycap1.String())
	strBuf.WriteString(" Внимание")

	{
		msg := tgbotapi.NewEditMessageText(player.ChatId, messageId, strBuf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		sleep(1 * time.Second)
	}

	strBuf.Reset()
	strBuf.WriteString(emoji.Rocket.String())
	strBuf.WriteString(" Марш!")

	{
		msg := tgbotapi.NewEditMessageText(player.ChatId, messageId, strBuf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
	}

	return nil
}

// select the letter that the player needs to call the words
func (r *Session) generateAndSendLetterMsg(player *Player) error {
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	output, err := r.tg.Send(tgbotapi.NewMessage(player.ChatId, textStartLetterMsg))
	if err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	sndCh := make(chan string, 1)

	g := errgroup.Group{}
	g.Go(func() error {
		for msg := range sndCh {
			if _, err := r.tg.Send(tgbotapi.NewEditMessageText(player.ChatId, output.MessageID, msg)); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}

		return nil
	})

	var sentMsg, sentLetter string
	for i := 0; i < generateLetterTimes; i++ {
		for strBuf.String() == sentMsg {
			strBuf.Reset()
			idx := fastrand.Uint32n(uint32(len(r.config.Letters)))
			strBuf.WriteString(textStartLetterMsg)
			strBuf.WriteString(r.config.Letters[idx])
			sentLetter = r.config.Letters[idx]
		}

		sndCh <- strBuf.String()
		sentMsg = strBuf.String()
	}

	strBuf.Reset()
	strBuf.WriteString(fmt.Sprintf("Игрок %s должен назвать слова:\n\n", player.FirstName))
	strBuf.WriteString(r.buildCategoriesStr())
	strBuf.WriteString("\n\n")
	strBuf.WriteString(fmt.Sprintf("На букву: *%s*", sentLetter))
	r.syncBroadcast(strBuf.String(), player.UserId)

	close(sndCh)

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (r *Session) playing(ctx context.Context) error {
PlayerLoop:
	for {
		// choosing the next player
		player, ok := r.nextPlayer()
		if !ok {
			r.stateCh <- stateKindProcessing
			return nil
		}

		// send "next player" asyncBroadcast message
		nextPlayerMsg := fmt.Sprintf(textNextPlayerMsg, emoji.GameDie.String(), player.FirstName)
		r.syncBroadcast(nextPlayerMsg, player.UserId)

		// send start button and register start button callback
		if err := r.sendStartMsg(player); err != nil {
			return fmt.Errorf("send start msg: %v", err)
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
					player.FirstName,
					defaultInactiveFatalTime-defaultInactiveWarnTime,
				))
			case <-timerFatal.C:
				timerFatal.Stop()
				r.syncBroadcast(fmt.Sprintf(
					"%s не начал раунд в течение %d сек, он пропускает ход",
					player.FirstName,
					defaultInactiveFatalTime,
				))
				continue PlayerLoop
			case <-ctx.Done():
				return nil
			}
		}

		//  generating the letter that the words begin with
		if err := r.generateAndSendLetterMsg(player); err != nil {
			return fmt.Errorf("generate and send letter msg: %v", err)
		}

		if err := r.sendReadyMsg(player); err != nil {
			return fmt.Errorf("send ready msg: %v", err)
		}

		// create ticker. Update player timer every 1sec
		secs, timeSince, err := r.ticker(ctx, player)
		if err != nil {
			return fmt.Errorf("ticker: %v", err)
		}

		// fill rate struct
		rate := &Rate{
			Duration:  time.Since(timeSince), // calculate the duration for the accuracy of determining the winners
			Points:    secs,
			Completed: secs > 0,
		}

		// vote features
		if r.config.Vote {
			err, done := r.votes(ctx, rate)
			if done {
				return fmt.Errorf("votes: %v", err)
			}
		}

		r.mtx.Lock()
		player.Rates = append(player.Rates, rate)
		r.mtx.Unlock()

		// send data on the round players
		r.sndCh <- tgbotapi.NewMessage(player.ChatId, fmt.Sprintf(textStopPlayerRoundMsg, rate.Points))

		roundFinishedMsg := fmt.Sprintf(textStopPlayerRoundBroadcastMsg, player.FirstName, rate.Points)
		r.asyncBroadcast(roundFinishedMsg, player.UserId)
		sleep(5 * time.Second)
	}
}

func (r *Session) votes(ctx context.Context, rate *Rate) (error, bool) {
	// create new active vote
	r.activeVote = newVote()

	// for storing the message id
	voteMessages := map[int64]int{}

	// send vote buttons and register callbacks
	if err := r.broadcastVoteButtonsAndRegisterCb(voteMessages); err != nil {
		return fmt.Errorf("broadcast vote buttons and register cb: %v", err), true
	}

	timer := time.NewTimer(defaultInactiveVoteTime * time.Second)
	defer timer.Stop()
	defer close(r.activeVote.pub)

VoteLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, true
		case <-timer.C:
			break VoteLoop
		case <-r.activeVote.pub:
			// updating data in the voting buttons
			if err := r.broadcastVotes(voteMessages); err != nil {
				return fmt.Errorf("broadcast votes: %v", err), true
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
	for _, messageId := range voteMessages {
		delete(r.cb, messageId)
	}

	switch {
	case r.activeVote.thumbUp == 0 && r.activeVote.thumbDown == 0, r.activeVote.thumbUp < r.activeVote.thumbDown:
		// if players believe that an active player failed, they lose all points for the round
		rate.Points = 0
		rate.Completed = false
	case r.activeVote.thumbUp == r.activeVote.thumbDown:
		rate.Points = rate.Points / 2
		rate.Completed = true
	default:
	}

	return nil, false
}

func (r *Session) broadcastVoteButtonsAndRegisterCb(voteMessages map[int64]int) error {
	r.mtx.Lock()
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			thumbUpButton(r.activeVote.thumbUp),
			thumbDownButton(r.activeVote.thumbDown),
		),
	)

	// creating a voting system and defining callbacks for voting
	for _, player := range r.Players {
		if player.IsPlaying() && !player.Offline {
			msg := tgbotapi.NewMessage(player.ChatId, textVoteMsg)
			msg.ReplyMarkup = markup
			// sending the thumbs up and thumbs down buttons
			output, err := r.tg.Send(msg)
			if err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
			// registering callbacks for voting
			voteMessages[player.ChatId] = output.MessageID
			r.cb[output.MessageID] = func(query *tgbotapi.CallbackQuery) error {
				switch query.Data {
				case textThumbUp:
					r.thumbUp()
				case textThumbDown:
					r.thumbDown()
				default:
				}

				if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, query.Data)); err != nil {
					return fmt.Errorf("send answer msg: %v", err)
				}

				return nil
			}

		}
	}

	r.mtx.Unlock()
	return nil
}

func (r *Session) broadcastVotes(voteMessages map[int64]int) error {
	r.mtx.RLock()
	// send all users changes in votes so that all players can see the overall result
	for chatId, messageId := range voteMessages {
		msg := tgbotapi.NewEditMessageReplyMarkup(
			chatId,
			messageId,
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					thumbUpButton(r.activeVote.thumbUp),
					thumbDownButton(r.activeVote.thumbDown),
				),
			),
		)

		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
	}
	r.mtx.RUnlock()
	return nil
}

func (r *Session) sendTimerMsg(player *Player) (int, error) {
	var messageId int
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	strBuf.WriteString(emoji.Stopwatch.String())
	strBuf.WriteString(" ")
	strBuf.WriteString(strconv.Itoa(defaultRoundTime))
	strBuf.WriteString(" сек")
	msg := tgbotapi.NewMessage(player.ChatId, textStopButton)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(strBuf.String(), textTimerBtnData),
			tgbotapi.NewInlineKeyboardButtonData(textStopBtnData, textStopBtnData),
		),
	)

	output, err := r.tg.Send(msg)
	if err != nil {
		return messageId, fmt.Errorf("send msg: %v", err)
	}

	return output.MessageID, nil
}

// formatting stop, timer button and send it
func (r *Session) updateTimerMsg(player *Player, messageId, secs int) error {
	strBuf := strpool.Get()
	defer func() {
		strBuf.Reset()
		strpool.Put(strBuf)
	}()

	strBuf.WriteString(emoji.Stopwatch.String())
	strBuf.WriteString(" ")
	strBuf.WriteString(strconv.Itoa(secs))
	strBuf.WriteString(" сек")

	msg := tgbotapi.NewEditMessageReplyMarkup(
		player.ChatId,
		messageId,
		tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(strBuf.String(), textTimerBtnData),
				tgbotapi.NewInlineKeyboardButtonData(textStopBtnData, textStopBtnData),
			),
		),
	)

	if _, err := r.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

// updating the player's timer and registering callbacks to stop the timer
func (r *Session) ticker(ctx context.Context, player *Player) (int, time.Time, error) {
	secs := defaultRoundTime
	messageId, err := r.sendTimerMsg(player)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("send timer msg: %v", err)
	}

	// register stop button callback
	r.cb[messageId] = func(query *tgbotapi.CallbackQuery) error {
		defer delete(r.cb, messageId)
		if query.Data == textStopBtnData {
			if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, textStopBtnDataAnswer)); err != nil {
				return fmt.Errorf("send answer msg: %v", err)
			}

			r.stopCh <- struct{}{}
		}

		return nil
	}

	since := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

OuterLoop:
	for {
		select {
		case <-ctx.Done():
			break OuterLoop
		case <-r.passCh:
			break OuterLoop
		case <-r.stopCh:
			break OuterLoop
		case <-ticker.C:
			// subtract 1 second each tick
			secs--
			if secs <= 0 {
				break OuterLoop
			}
			// updating timer
			if err := r.updateTimerMsg(player, messageId, secs); err != nil {
				return 0, time.Time{}, fmt.Errorf("update timer msg: %v", err)
			}
		}
	}

	return secs, since, nil
}

func (r *Session) nextRound() {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.currRoundIdx++
}

func (r *Session) changeState(kind stateKind) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.state = kind
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

// calculating the round's favorite
func (r *Session) roundFavorite() *Player {
	r.mtx.RLock()
	players := make([]*Player, len(r.Players))
	r.mtx.RUnlock()
	copy(players, r.Players)

	sort.Slice(players, func(i, j int) bool {
		return players[i].Rates[r.currRoundIdx].Points > players[j].Rates[r.currRoundIdx].Points &&
			players[i].Rates[r.currRoundIdx].Duration < players[j].Rates[r.currRoundIdx].Duration
	})

	return players[0]
}

func (r *Session) findPlayer(userId int64) (*Player, bool) {
	for _, player := range r.Players {
		if player.UserId == userId {
			return player, true
		}
	}

	return nil, false
}

//  Select a player who hasn't played in this round yet
func (r *Session) nextPlayer() (*Player, bool) {
	var players []*Player
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	for _, player := range r.Players {
		if player.IsPlaying() && len(player.Rates) == r.currRoundIdx {
			players = append(players, player)
		}
	}

	if len(players) == 0 {
		return nil, false
	}

	rand := fastrand.Uint32n(uint32(len(players)))
	return players[rand], true
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

func (r *Session) ActivePlayersLen() int {
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

// register new player and send asyncBroadcast message about it
func (r *Session) AddPlayer(player *Player) error {
	if player, ok := r.addPlayer(player); ok {
		registerPlayerMsg := fmt.Sprintf(textPlayerJoinedGameMsg, player.FirstName)
		r.asyncBroadcast(registerPlayerMsg, player.UserId)
	}

	return nil
}

// create and append new player with state "Playing"
func (r *Session) addPlayer(player *Player) (*Player, bool) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	for _, p := range r.Players {
		if p.ChatId == player.ChatId && p.UserId == player.UserId && p.FirstName == player.FirstName {
			return nil, false
		}
	}

	r.Players = append(r.Players, player)

	return player, true
}

// remove player from game and send asyncBroadcast message about it
func (r *Session) RemovePlayer(userId int64) {
	players := r.removePlayer(userId)
	for _, player := range players {
		r.asyncBroadcast(fmt.Sprintf(textPlayerLeftGameMsg, player.FirstName))
	}
}

// set PlayerStateKindLeaving status
func (r *Session) removePlayer(userId int64) []*Player {
	var players []*Player
	r.mtx.Lock()
	defer r.mtx.Unlock()
	for _, p := range r.Players {
		if p.UserId == userId {
			p.State = PlayerStateKindLeaving
			players = append(players, p)
		}
	}

	return players
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
