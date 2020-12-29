package builder

import (
	"bloop/internal/bloopsbot/resource"
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

type QueryCallbackHandlerFunc func(query *tgbotapi.CallbackQuery) error

type stateKind uint8

const (
	stateKindCategories stateKind = iota + 1
	stateKindRoundsNum
	stateKindRoundTime
	stateKindLetters
	stateKindBloops
	stateKindVote
	stateKindDone
)

var stages = []stateKind{
	stateKindCategories,
	stateKindRoundsNum,
	stateKindRoundTime,
	stateKindLetters,
	stateKindBloops,
	stateKindVote,
	stateKindDone,
}

func NewSession(
	tg *tgbotapi.BotAPI,
	chatId int64,
	authorId int64,
	authorName string,
	doneFn func(session *Session) error,
	warnFn func(session *Session) error,
	timeout time.Duration,
) (*Session, error) {
	state := newStateMachine(stages...)
	s := &Session{
		tg:              tg,
		state:           state,
		messageCh:       make(chan struct{}, 1),
		ChatId:          chatId,
		AuthorId:        authorId,
		AuthorName:      authorName,
		RoundsNum:       defaultRoundsNum,
		RoundTime:       defaultRoundTime,
		timeout:         timeout,
		doneFn:          doneFn,
		warnFn:          warnFn,
		controlHandlers: map[string]QueryCallbackHandlerFunc{},
		actionHandlers:  map[stateKind]QueryCallbackHandlerFunc{},
		CreatedAt:       time.Now(),
	}

	s.Categories = make([]resource.Category, len(resource.Categories))
	copy(s.Categories, resource.Categories)

	s.Letters = make([]resource.Letter, len(resource.Letters))
	copy(s.Letters, resource.Letters)

	s.handleControlCb(resource.BuilderInlineNextData, s.clickOnNext)
	s.handleControlCb(resource.BuilderInlinePrevData, s.clickOnPrev)
	s.handleControlCb(resource.BuilderInlineDoneData, s.clickOnDone)

	s.handleActionCb(stateKindCategories, s.clickOnCategories)
	s.handleActionCb(stateKindRoundTime, s.clickOnRoundTime)
	s.handleActionCb(stateKindRoundsNum, s.clickOnRoundsNum)
	s.handleActionCb(stateKindLetters, s.clickOnLetters)
	s.handleActionCb(stateKindBloops, s.clickOnBloops)
	s.handleActionCb(stateKindVote, s.clickOnVote)

	return s, nil
}

type Session struct {
	mtx sync.RWMutex

	AuthorId   int64
	AuthorName string
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
	controlHandlers map[string]QueryCallbackHandlerFunc
	actionHandlers  map[stateKind]QueryCallbackHandlerFunc
	cancel          func()
	doneFn          func(session *Session) error
	warnFn          func(session *Session) error
}

func (bs *Session) Run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, bs.timeout)
	bs.cancel = cancel
	logger := logging.FromContext(ctx)
	bs.sema.Do(func() {
		go bs.loop(ctx)
		bs.messageCh <- struct{}{}
	})

	logger.Infof("Building session has started, author: %s", bs.AuthorName)
}

func (bs *Session) Stop() {
	defer close(bs.messageCh)
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

func (bs *Session) handleControlCb(command string, fn QueryCallbackHandlerFunc) {
	bs.controlHandlers[command] = fn
}

func (bs *Session) handleActionCb(kind stateKind, fn QueryCallbackHandlerFunc) {
	bs.actionHandlers[kind] = fn
}

func (bs *Session) isControlCmd(queryData string) bool {
	for cmd := range bs.controlHandlers {
		if cmd == queryData {
			return true
		}
	}

	return false
}

func (bs *Session) executeCbQuery(query *tgbotapi.CallbackQuery) error {
	if query.Message.MessageID != bs.messageId {
		return fmt.Errorf("callback with message id %d not found", query.Message.MessageID)
	}

	if bs.isControlCmd(query.Data) {
		fn := bs.controlHandlers[query.Data]
		if err := fn(query); err != nil {
			return fmt.Errorf("execute control handler: %v", err)
		}

		return nil
	}

	kind := bs.state.curr()
	fn, ok := bs.actionHandlers[kind]
	if !ok {
		return fmt.Errorf("action handler not found")
	}

	if err := fn(query); err != nil {
		return fmt.Errorf("action handle: %v", err)
	}

	return nil
}

func (bs *Session) loop(ctx context.Context) {
	logger := logging.FromContext(ctx).Named("builder.loop")
	defer bs.shutdown(ctx)
	for {
		select {
		case <-ctx.Done():
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
			case stateKindRoundsNum:
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
			case stateKindBloops:
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

func (bs *Session) shutdown(ctx context.Context) bool {
	logger := logging.FromContext(ctx)
	if time.Since(bs.CreatedAt) <= bs.timeout {
		if bs.state.state != stateKindDone {
			if _, err := bs.tg.Send(tgbotapi.NewMessage(bs.AuthorId, resource.TextBuilderWarnMsg)); err != nil {
				logger.Errorf("send msg: %v", err)
			}

			if err := bs.warnFn(bs); err != nil {
				logger.Errorf("done function: %v", err)
			}

			return true
		}
		if err := bs.doneFn(bs); err != nil {
			logger.Errorf("done function: %v", err)
		}
	}
	logger.Infof("Building session is complete, author: %s", bs.AuthorName)
	return false
}

func (bs *Session) clickOnPrev(query *tgbotapi.CallbackQuery) error {
	bs.state.prev()
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlinePrevText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnNext(query *tgbotapi.CallbackQuery) error {
	bs.state.next()
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %v", err)
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnDone(query *tgbotapi.CallbackQuery) error {
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineDoneText)); err != nil {
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

func (bs *Session) clickOnLetters(query *tgbotapi.CallbackQuery) error {
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

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineNextText)); err != nil {
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

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineNextText)); err != nil {
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
