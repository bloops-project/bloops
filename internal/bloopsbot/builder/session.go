package builder

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	"github.com/bloops-games/bloops/internal/logging"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	defaultRoundsNum = 1
	minCategoriesNum = 3
	defaultRoundTime = 30
)

type QueryCallbackHandlerFunc func(query *tgbotapi.CallbackQuery) error

type stateKind uint8

const (
	stateKindCategories stateKind = iota + 1
	stateKindRoundsNum
	stateKindLetters
	stateKindBloops
	stateKindVote
	stateKindDone
)

var stages = []stateKind{
	stateKindCategories,
	stateKindRoundsNum,
	stateKindLetters,
	stateKindBloops,
	stateKindVote,
	stateKindDone,
}

func NewSession(
	tg *tgbotapi.BotAPI,
	chatID int64,
	authorID int64,
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
		ChatID:          chatID,
		AuthorID:        authorID,
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
	s.handleActionCb(stateKindRoundsNum, s.clickOnRoundsNum)
	s.handleActionCb(stateKindLetters, s.clickOnLetters)
	s.handleActionCb(stateKindBloops, s.clickOnBloops)
	s.handleActionCb(stateKindVote, s.clickOnVote)

	return s, nil
}

type Session struct {
	mtx sync.RWMutex

	AuthorID   int64
	AuthorName string
	Categories []resource.Category
	Letters    []resource.Letter
	RoundsNum  int
	RoundTime  int
	Vote       bool
	Bloops     bool
	ChatID     int64
	CreatedAt  time.Time

	tg        *tgbotapi.BotAPI
	state     *stateMachine
	messageCh chan struct{}
	sema      sync.Once

	messageID int

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
			return fmt.Errorf("execute cb query: %w", err)
		}
	}

	if upd.Message != nil {
		if err := bs.executeMessageQuery(upd.Message); err != nil {
			return fmt.Errorf("execute message query: %w", err)
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

		msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatID, bs.messageID, bs.menuInlineButtons(bs.renderInlineCategories()))
		if _, err := bs.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %w", err)
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
	if query.Message.MessageID != bs.messageID {
		return fmt.Errorf("callback with message id %d not found", query.Message.MessageID)
	}

	if bs.isControlCmd(query.Data) {
		fn := bs.controlHandlers[query.Data]
		if err := fn(query); err != nil {
			return fmt.Errorf("execute control handler: %w", err)
		}

		return nil
	}

	kind := bs.state.curr()
	fn, ok := bs.actionHandlers[kind]
	if !ok {
		return fmt.Errorf("action handler not found")
	}

	if err := fn(query); err != nil {
		return fmt.Errorf("action handle: %w", err)
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
				logger.Infof("Building session, sending categories, author %s", bs.AuthorName)
				msg := tgbotapi.NewMessage(bs.ChatID, resource.TextChooseCategories)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineCategories())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send categories: %v", err)
				}
				bs.messageID = output.MessageID
			case stateKindRoundsNum:
				logger.Infof("Building session, sending rounds number, author %s", bs.AuthorName)
				msg := tgbotapi.NewMessage(bs.ChatID, resource.TextChooseRoundsNum)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderRoundsNum())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send round num: %v", err)
				}
				bs.messageID = output.MessageID
			case stateKindLetters:
				logger.Infof("Building session, sending letters, author %s", bs.AuthorName)
				msg := tgbotapi.NewMessage(bs.ChatID, resource.TextDeleteComplexLetters)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineLetters())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send letters: %v", err)
				}
				bs.messageID = output.MessageID
			case stateKindBloops:
				logger.Infof("Building session, sending bloopses, author %s", bs.AuthorName)
				msg := tgbotapi.NewMessage(bs.ChatID, resource.TextBloopsAllowed)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineBloops())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send letters: %v", err)
				}
				bs.messageID = output.MessageID
			case stateKindVote:
				logger.Infof("Building session, sending vote, author %s", bs.AuthorName)
				msg := tgbotapi.NewMessage(bs.ChatID, resource.TextVoteAllowed)
				msg.ReplyMarkup = bs.menuInlineButtons(bs.renderInlineVote())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send vote: %v", err)
				}
				bs.messageID = output.MessageID
			case stateKindDone:
				logger.Infof("Building session, sending done action, author %s", bs.AuthorName)
				msg := tgbotapi.NewMessage(bs.ChatID, resource.TextConfigurationDone)
				msg.ReplyMarkup = bs.menuInlineButtons(tgbotapi.NewInlineKeyboardMarkup())
				output, err := bs.tg.Send(msg)
				if err != nil {
					logger.Errorf("send done: %v", err)
				}
				bs.messageID = output.MessageID
			}
		}
	}
}

func (bs *Session) shutdown(ctx context.Context) bool {
	logger := logging.FromContext(ctx)
	if time.Since(bs.CreatedAt) <= bs.timeout {
		if bs.state.state != stateKindDone {
			if _, err := bs.tg.Send(tgbotapi.NewMessage(bs.AuthorID, resource.TextBuilderWarnMsg)); err != nil {
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
		return fmt.Errorf("send answer msg: %w", err)
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnNext(query *tgbotapi.CallbackQuery) error {
	bs.state.next()
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %w", err)
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnDone(query *tgbotapi.CallbackQuery) error {
	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineDoneText)); err != nil {
		return fmt.Errorf("send answer msg: %w", err)
	}

	if bs.numCategoriesIncluded() < minCategoriesNum {
		msg := tgbotapi.NewMessage(bs.ChatID, resource.TextAddLeastCategoryToComplete)
		if _, err := bs.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %w", err)
		}

		return nil
	}

	if !bs.lettersExist() {
		msg := tgbotapi.NewMessage(bs.ChatID, resource.TextAddLeastOneLetterToComplete)
		if _, err := bs.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %w", err)
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
		return fmt.Errorf("send answer msg: %w", err)
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatID, bs.messageID, bs.menuInlineButtons(bs.renderInlineCategories()))
	if _, err := bs.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func (bs *Session) clickOnRoundsNum(query *tgbotapi.CallbackQuery) error {
	n, err := strconv.Atoi(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %w", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, fmt.Sprintf(resource.TextRoundsNumAnswer, n))); err != nil {
		return fmt.Errorf("send answer msg: %w", err)
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
		return fmt.Errorf("send answer msg: %w", err)
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatID, bs.messageID, bs.menuInlineButtons(bs.renderInlineLetters()))
	if _, err := bs.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func (bs *Session) clickOnBloops(query *tgbotapi.CallbackQuery) error {
	value, err := strconv.ParseBool(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %w", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %w", err)
	}

	bs.Bloops = value
	bs.state.next()
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnVote(query *tgbotapi.CallbackQuery) error {
	value, err := strconv.ParseBool(query.Data)
	if err != nil {
		return fmt.Errorf("strconv: %w", err)
	}

	if _, err := bs.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.BuilderInlineNextText)); err != nil {
		return fmt.Errorf("send answer msg: %w", err)
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
