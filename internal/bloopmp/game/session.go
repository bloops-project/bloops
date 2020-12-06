package game

import (
	"context"
	"fmt"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type gameCb func(eventId uuid.UUID, cb *tgbotapi.CallbackQuery) error

type stateKind uint8

const (
	gameStateWaiting stateKind = iota + 1
	gameStatePlaying
	gameStateProcessing
	gameStateFinished
)

type rate struct {
	secs      int
	points    int
	completed bool
}

type PlayerScore struct {
	UserId    int64
	Username  string
	FirstName string
	Score     int
	TotalSecs int
	BestSecs  int
	Completed int
	Rounds    int
}

type Config struct {
	AuthorId   int64
	RoundsNum  int
	Categories []string
	Letters    []string
}

func NewGame(config Config, tg *tgbotapi.BotAPI, authorId int64) *Session {
	return &Session{
		config:   config,
		AuthorId: authorId,
		tg:       tg,
		stateCh:  make(chan stateKind, 1),
		state:    gameStateWaiting,
		cb:       map[int64]func(query *tgbotapi.CallbackQuery) error{},
	}
}

type Session struct {
	mtx sync.RWMutex

	config   Config
	tg       *tgbotapi.BotAPI // tg api instance
	AuthorId int64

	state        stateKind
	stateCh      chan stateKind
	players      []*player
	cb           map[int64]func(query *tgbotapi.CallbackQuery) error
	currRoundIdx int
	createdAt    time.Time
}

func (r *Session) broadcast(msg string, exclude ...int64) {
OuterLoop:
	for _, player := range r.players {
		for i := range exclude {
			if player.userId == exclude[i] {
				continue OuterLoop
			}
		}
		msg := tgbotapi.NewMessage(player.userId, msg)
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			continue
		}
	}
}

func (r *Session) Run(ctx context.Context) {
	go r.loop(ctx)
}

func (r *Session) Execute(userId int64, upd tgbotapi.Update) error {
	if upd.CallbackQuery != nil {
		if err := r.executeCb(userId, upd.CallbackQuery); err != nil {
			return err
		}
	}

	if upd.Message != nil {
		if err := r.executeCommand(userId, upd.Message); err != nil {
			return err
		}
	}

	return nil
}

func (r *Session) executeCommand(userId int64, query *tgbotapi.Message) error {
	if query.Text == "Начать" {
		r.stateCh <- gameStatePlaying
	}

	return nil
}

func (r *Session) executeCb(userId int64, query *tgbotapi.CallbackQuery) error {
	if cb, ok := r.cb[userId]; ok {
		if err := cb(query); err != nil {
			return err
		}
	}
	return nil
}

func (r *Session) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// send user winner
			return
		case state := <-r.stateCh:
			switch state {
			case gameStateFinished:
				r.state = gameStateFinished
				return
			case gameStateProcessing:
				r.state = gameStateProcessing
				// считаем результаты раунда
				if r.config.RoundsNum == r.currRoundIdx+1 {
					r.stateCh <- gameStateFinished
				}

				player := r.favorite()

				r.broadcast(fmt.Sprintf(
					"Раунд %d завершен. Фаворит раунда: %s - %d очков",
					r.currRoundIdx+1,
					player.firstName,
					player.rates[r.currRoundIdx].points,
				))

				r.nextRound()
				r.stateCh <- gameStatePlaying
			case gameStatePlaying:
				r.state = gameStatePlaying
				// playing
				if err := r.playing(); err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}

func (r *Session) playing() error {
	strBuilder := strings.Builder{}
	for {
		player, ok := r.nextPlayer()
		if !ok {
			r.stateCh <- gameStateProcessing
			return nil
		}

		r.broadcast(fmt.Sprintf("Следующий играет %s - *%s*", emoji.GameDie.String(), player.firstName), player.userId)

		stopCh := make(chan struct{}, 1)
		startCh := make(chan struct{}, 1)
		nextCh := make(chan struct{}, 1)
		timer := time.NewTicker(1 * time.Second)

		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Старт", "Старт"),
			),
		)

		msg := tgbotapi.NewMessage(player.chatId, "Нажми Старт как будешь готов")
		msg.ReplyMarkup = markup
		if _, err := r.tg.Send(msg); err != nil {
			return err
		}

		r.cb[player.userId] = func(query *tgbotapi.CallbackQuery) error {
			if query.Data == "Старт" {
				if _, err := r.tg.AnswerCallbackQuery(
					tgbotapi.NewCallback(query.ID, "Старт!")); err != nil {
					return err
				}
				startCh <- struct{}{}
			}

			return nil
		}

		<-startCh

		r.cb[player.userId] = func(query *tgbotapi.CallbackQuery) error {
			if query.Data == "Стоп" {
				if _, err := r.tg.AnswerCallbackQuery(
					tgbotapi.NewCallback(query.ID, "Стоп!")); err != nil {
					return err
				}
				stopCh <- struct{}{}
			}
			return nil
		}

		strBuilder.Reset()
		strBuilder.WriteString("*Нужно назвать по одному слову:* ")
		for i, category := range r.config.Categories {
			strBuilder.WriteString(category)
			if i != len(r.config.Categories)-1 {
				strBuilder.WriteString(", ")
			}
		}

		categoryMsg := tgbotapi.NewMessage(player.chatId, strBuilder.String())
		categoryMsg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(categoryMsg); err != nil {
			fmt.Println(err)
		}

		strBuilder.Reset()
		time.Sleep(1 * time.Second)
		strBuilder.WriteString("На букву - ")
		letterMsg := tgbotapi.NewMessage(player.chatId, strBuilder.String())
		outputLetterMsg, err := r.tg.Send(letterMsg)
		if err != nil {
			fmt.Println(err)
		}

		rand.NewSource(time.Now().UnixNano())

		for i := 0; i < 10; i++ {
			strBuilder.Reset()
			letterIdx := rand.Int31n(int32(len(r.config.Letters)))
			strBuilder.WriteString("На букву - ")
			strBuilder.WriteString(r.config.Letters[letterIdx])
			msg := tgbotapi.NewEditMessageText(player.chatId, outputLetterMsg.MessageID, strBuilder.String())
			if _, err := r.tg.Send(msg); err != nil {
				fmt.Println(err)
			}
		}

		secs := 30

		markup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(secs)+" Таймер", "Таймер"),
				tgbotapi.NewInlineKeyboardButtonData("Стоп", "Стоп"),
			),
		)

		gameMsg := tgbotapi.NewEditMessageReplyMarkup(player.chatId, outputLetterMsg.MessageID, markup)
		outputGameMsg, err := r.tg.Send(gameMsg)
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(2 * time.Second)

	OuterLoop:
		for {
			select {
			case <-timer.C:
				secs--
				if secs <= 0 {
					stopCh <- struct{}{}
				}

				markup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(secs)+" Таймер", "Таймер"),
						tgbotapi.NewInlineKeyboardButtonData("Стоп", "Стоп"),
					),
				)

				gameMsg := tgbotapi.NewEditMessageReplyMarkup(player.chatId, outputGameMsg.MessageID, markup)
				if _, err := r.tg.Send(gameMsg); err != nil {
					fmt.Println(err)
				}
			case <-stopCh:
				score := r.calcPlayerScore(secs)
				rate := &rate{
					secs:      30 - secs,
					points:    score,
					completed: score > 0,
				}

				player.rates = append(player.rates, rate)

				msg := tgbotapi.NewMessage(player.chatId, fmt.Sprintf("Завершено! Ты набрал %d очков!", rate.points))
				if _, err := r.tg.Send(msg); err != nil {
					fmt.Println(err)
				}

				r.broadcast(fmt.Sprintf("%s набирает %d очков", player.firstName, rate.points), player.userId)

				nextCh <- struct{}{}
				break OuterLoop
			}
		}

		delete(r.cb, player.userId)
		<-nextCh
	}
}

func (r *Session) calcPlayerScore(secs int) int {
	var score int
	switch {
	case secs <= 0:
		score = 0
	case secs >= 25:
		score = 100
	case secs >= 20 && secs < 25:
		score = 80
	case secs >= 15 && secs < 20:
		score = 60
	case secs >= 10 && secs < 15:
		score = 40
	case secs >= 5 && secs < 10:
		score = 20
	case secs > 0 && secs < 5:
		score = 10
	}

	return score
}

func (r *Session) nextRound() {
	r.currRoundIdx++
}

func (r *Session) Scores() []PlayerScore {
	scores := make([]PlayerScore, len(r.players))
	for i, player := range r.players {
		playerScore := PlayerScore{
			UserId:    player.userId,
			FirstName: player.firstName,
			Rounds:    len(player.rates),
		}

		for _, rate := range player.rates {
			playerScore.Score += rate.points
			if rate.completed {
				playerScore.Completed++
			}

			if rate.secs < playerScore.BestSecs {
				playerScore.BestSecs = rate.secs
			}

			playerScore.TotalSecs += rate.secs
		}

		scores[i] = playerScore
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores
}

func (r *Session) favorite() *player {
	players := make([]*player, len(r.players))
	copy(players, r.players)
	sort.Slice(players, func(i, j int) bool {
		return players[i].rates[r.currRoundIdx].points > players[j].rates[r.currRoundIdx].points &&
			players[i].rates[r.currRoundIdx].secs < players[j].rates[r.currRoundIdx].secs
	})

	return players[0]
}

func (r *Session) nextPlayer() (*player, bool) {
	for i := range r.players {
		if len(r.players[i].rates) == r.currRoundIdx {
			return r.players[i], true
		}
	}
	return nil, false
}

func (r *Session) RegisterPlayer(userId, chatId int64, firstName string) {
	r.registerPlayer(userId, chatId, firstName)
}

func (r *Session) registerPlayer(userId, chatId int64, firstName string) {
	player := newPlayer(chatId, userId, firstName)
	r.players = append(r.players, player)
	r.broadcast(fmt.Sprintf("Игрок %s присоединился к игре", player.firstName), player.userId)
}

func (r *Session) Leave(userId int64) {
	r.leave(userId)
}

func (r *Session) leave(userId int64) {
	for i, player := range r.players {
		if player.userId == userId {
			r.broadcast(fmt.Sprintf("Игрок %s покинул игру", r.players[i].firstName))
			r.players = append(r.players[:i], r.players[i+1:]...)
			break
		}
	}
}
