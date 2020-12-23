package builder

import (
	"bloop/internal/bloops/resource"
	"bloop/internal/logging"
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
	"sync"
	"time"
)

const (
	defaultRoundsNum = 3
	minCategoriesNum = 3
	defaultRoundTime = 30
)

type CommandKind uint8

const (
	CommandKindPrevAction CommandKind = iota + 1
	CommandKindNextAction
	CommandKindDoneAction
)

type QueryCallbackFn func(query *tgbotapi.CallbackQuery) error

type stateKind uint8

const (
	stateKindCategories stateKind = iota + 1
	stateKindRounds
	stateKindRoundTime
	stateKindLetters
	stateBloops
	stateKindVote
	stateKindDone
)

var stages = []stateKind{stateKindCategories, stateKindRounds, stateKindRoundTime, stateKindLetters, stateBloops, stateKindVote, stateKindDone}

func NewSession(
	tg *tgbotapi.BotAPI,
	chatId int64,
	authorId int64,
	doneFn func(session *Session) error,
	timeout time.Duration,
) (*Session, error) {
	state := newStateMachine(stages...)
	s := &Session{
		tg:              tg,
		state:           state,
		messageCh:       make(chan struct{}, 1),
		ChatId:          chatId,
		AuthorId:        authorId,
		RoundsNum:       defaultRoundsNum,
		RoundTime:       defaultRoundTime,
		timeout:         timeout,
		doneFn:          doneFn,
		controlHandlers: map[string]QueryCallbackFn{},
		CreatedAt:       time.Now(),
	}

	s.Categories = make([]resource.Category, len(resource.Categories))
	copy(s.Categories, resource.Categories)

	s.Letters = make([]resource.Letter, len(resource.Letters))
	copy(s.Letters, resource.Letters)

	if err := s.bindControlCommand(resource.InlineNextText, CommandKindNextAction); err != nil {
		return nil, fmt.Errorf("bind control command: %v", err)
	}

	if err := s.bindControlCommand(resource.InlinePrevText, CommandKindPrevAction); err != nil {
		return nil, fmt.Errorf("bind control command: %v", err)
	}

	if err := s.bindControlCommand(resource.InlineDoneText, CommandKindDoneAction); err != nil {
		return nil, fmt.Errorf("bind control command: %v", err)
	}

	return s, nil
}

type Session struct {
	mtx sync.RWMutex

	AuthorId   int64
	Categories []resource.Category
	Letters    []resource.Letter
	RoundsNum  int
	RoundTime  int
	Vote       bool
	Bloops     bool
	ChatId     int64
	CreatedAt  time.Time

	tg        *tgbotapi.BotAPI
	state     *stateMachine
	messageCh chan struct{}
	sema      sync.Once

	messageId int

	timeout         time.Duration
	controlHandlers map[string]QueryCallbackFn
	cancel          func()
	doneFn          func(session *Session) error
}

func (bs *Session) Run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, bs.timeout)
	bs.cancel = cancel
	bs.sema.Do(func() {
		go bs.loop(ctx)
		bs.messageCh <- struct{}{}
	})
}

func (bs *Session) Stop() {
	bs.cancel()
}

func (bs *Session) Execute(upd tgbotapi.Update) error {
	bs.mtx.Lock()
	defer bs.mtx.Unlock()

	if upd.CallbackQuery != nil {
		if err := bs.executeCbQuery(upd.CallbackQuery); err != nil {
			return fmt.Errorf("execute cb query: %v", err)
		}
	}

	if upd.Message != nil {
		if err := bs.executeMessageQuery(upd.Message); err != nil {
			return fmt.Errorf("execute message query: %v", err)
		}
	}

	return nil
}

func (bs *Session) executeMessageQuery(query *tgbotapi.Message) error {
	if bs.state.curr() == stateKindCategories {
		bs.Categories = append(bs.Categories, resource.Category{
			Text:   query.Text,
			Status: true,
		})

		msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatId, bs.messageId, bs.menuInlineButtons(bs.renderInlineCategories()))
		if _, err := bs.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
	}

	return nil
}

func (bs *Session) executeCbQuery(query *tgbotapi.CallbackQuery) error {
	if query.Message.MessageID != bs.messageId {
		return fmt.Errorf("callback with message id %d not found", query.Message.MessageID)
	}

	if bs.isControlCmd(query.Data) {
		fn := bs.controlHandlers[query.Data]
		if err := fn(query); err != nil {
			return fmt.Errorf("fn : %v", err)
		}

		return nil
	}

	switch bs.state.curr() {
	case stateKindCategories:
		if err := bs.clickOnCategories(query); err != nil {
			return fmt.Errorf("click on categories: %v", err)
		}
	case stateKindRounds:
		if err := bs.clickOnRoundsNum(query); err != nil {
			return fmt.Errorf("click on rounds num: %v", err)
		}
	case stateKindRoundTime:
		if err := bs.clickOnRoundTime(query); err != nil {
			return fmt.Errorf("click on round time: %v", err)
		}
	case stateKindLetters:
		if err := bs.clickOnLetter(query); err != nil {
			return fmt.Errorf("click on letter: %v", err)
		}
	case stateBloops:
		if err := bs.clickOnBloops(query); err != nil {
			return fmt.Errorf("click on challenges: %v", err)
		}
	case stateKindVote:
		if err := bs.clickOnVote(query); err != nil {
			return fmt.Errorf("click on vote: %v", err)
		}
	}

	return nil
}

func (bs *Session) loop(ctx context.Context) {
	logger := logging.FromContext(ctx).Named("builder.loop")
	for {
		select {
		case <-ctx.Done():
			if err := bs.doneFn(bs); err != nil {
				logger.Errorf("done function: %v", err)
			}

			return
		case <-bs.messageCh:
			switch bs.state.curr() {
			case stateKindCategories:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextChooseCategories)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineCategories())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send categories: %v", err)
				}
				bs.messageId = output.MessageID
			case stateKindRounds:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextChooseRoundsNum)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderRoundsNum())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send round num: %v", err)
				}
				bs.messageId = output.MessageID
			case stateKindRoundTime:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextRoundTime)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderRoundsTime())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send round num: %v", err)
				}
				bs.messageId = output.MessageID
			case stateKindLetters:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextDeleteComplexLetters)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineLetters())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send letters: %v", err)
				}
				bs.messageId = output.MessageID
			case stateBloops:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextBloopsAllowed)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineBloops())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send letters: %v", err)
				}
				bs.messageId = output.MessageID
			case stateKindVote:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextVoteAllowed)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineVote())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send vote: %v", err)
				}
				bs.messageId = output.MessageID
			case stateKindDone:
				msg := tgbotapi.NewMessage(bs.ChatId, resource.TextConfigurationDone)
				msg.ReplyMarkup = bs.menuInlineButtons(tgbotapi.NewInlineKeyboardMarkup())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send done: %v", err)
				}
				bs.messageId = output.MessageID
			}
		}
	}
}

func (bs *Session) clickOnPrev(query *tgbotapi.CallbackQuery) error {
	bs.state.prev()
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.InlinePrevText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnNext(query *tgbotapi.CallbackQuery) error {
	bs.state.next()
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.InlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnDone(query *tgbotapi.CallbackQuery) error {
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.InlineDoneText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	if bs.numCategoriesIncluded() < minCategoriesNum {
		msg := tgbotapi.NewMessage(bs.ChatId, resource.TextAddLeastCategoryToComplete)
		if _, err := bs.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		return nil
	}

	if !bs.lettersExist() {
		msg := tgbotapi.NewMessage(bs.ChatId, resource.TextAddLeastOneLetterToComplete)
		if _, err := bs.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		return nil
	}

	bs.cancel()

	return nil
}

func (bs *Session) bindControlCommand(command string, kind CommandKind) error {
	var err error
	switch kind {
	case CommandKindDoneAction:
		bs.Handle(command, bs.clickOnDone)
	case CommandKindNextAction:
		bs.Handle(command, bs.clickOnNext)
	case CommandKindPrevAction:
		bs.Handle(command, bs.clickOnPrev)
	default:
		err = fmt.Errorf("control command kind not found")
	}

	return err
}

func (bs *Session) Handle(command string, fn QueryCallbackFn) {
	bs.controlHandlers[command] = fn
}

func (bs *Session) isControlCmd(command string) bool {
	for cmd := range bs.controlHandlers {
		if cmd == command {
			return true
		}
	}

	return false
}

func (bs *Session) clickOnCategories(query *tgbotapi.CallbackQuery) error {
	var answer string
	for i, category := range bs.Categories {
		if query.Data == category.Text {
			bs.Categories[i].Status = !category.Status
			if bs.Categories[i].Status {
				answer = fmt.Sprintf(resource.TextAddedCategory, category.Text)
			} else {
				answer = fmt.Sprintf(resource.TextDeletedCategory, category.Text)
			}
		}
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, answer)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatId, bs.messageId, bs.menuInlineButtons(bs.renderInlineCategories()))
	if _, err := bs.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (bs *Session) clickOnRoundTime(query *tgbotapi.CallbackQuery) error {
	n, err := strconv.Atoi(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %v", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, fmt.Sprintf(resource.TextRoundTimeAnswer, n))); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	bs.RoundTime = n
	bs.state.next()
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnRoundsNum(query *tgbotapi.CallbackQuery) error {
	n, err := strconv.Atoi(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %v", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, fmt.Sprintf(resource.TextRoundsNumAnswer, n))); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	bs.RoundsNum = n
	bs.state.next()
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnLetter(query *tgbotapi.CallbackQuery) error {
	var answer string
	for i, letter := range bs.Letters {
		if query.Data == letter.Text {
			bs.Letters[i].Status = !letter.Status
			if bs.Letters[i].Status {
				answer = fmt.Sprintf(resource.TextAddedLetter, letter.Text)
			} else {
				answer = fmt.Sprintf(resource.TextDeletedLetter, letter.Text)
			}
		}
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, answer)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatId, bs.messageId, bs.menuInlineButtons(bs.renderInlineLetters()))
	if _, err := bs.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (bs *Session) clickOnBloops(query *tgbotapi.CallbackQuery) error {
	value, err := strconv.ParseBool(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %v", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.InlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	bs.Bloops = value
	bs.state.next()
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnVote(query *tgbotapi.CallbackQuery) error {
	value, err := strconv.ParseBool(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %v", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.InlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}

	bs.Vote = value
	bs.state.next()
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) lettersExist() bool {
	for _, letter := range bs.Letters {
		if letter.Status {
			return true
		}
	}

	return false
}

func (bs *Session) numCategoriesIncluded() int {
	var n int
	for _, category := range bs.Categories {
		if category.Status {
			n++
		}
	}

	return n
}
