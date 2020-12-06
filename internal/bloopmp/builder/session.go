package builder

import (
	"context"
	"fmt"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
	"sync"
	"time"
)

type stateKind uint8

const (
	stateKindCategories stateKind = iota + 1
	stateKindRounds
	stateKindLetters
)

const DefaultRoundNum = 3

var stages = []stateKind{stateKindCategories, stateKindRounds, stateKindLetters}

var (
	lettersList = []string{"А", "Б", "В", "Г", "Д", "Е", "Ж", "З", "И", "К", "Л", "М", "Н", "О", "П", "Р", "С", "Т",
		"У", "Ф", "Х", "Ц", "Ч", "Ш", "Э", "Ю", "Я"}

	categoriesList = []string{"Страна", "Город", "Овощ или фрукт", "Имя", "Знаменитость", "Бренд", "Животное",
		"Технология", "Литературное произведение", "Любое слово"}

	roundsList = []int{1, 2, 3, 4, 5}
)

type Category struct {
	Text   string
	Status bool
}

type Letter struct {
	Text   string
	Status bool
}

func NewSession(
	tg *tgbotapi.BotAPI,
	chatId int64,
	authorId int64,
	done func(session *Session) error,
	abort func(session *Session) error,
	timeout time.Duration,
) (*Session, error) {
	state := newStateMachine(stages...)
	s := &Session{
		tg:              tg,
		state:           state,
		messageCh:       make(chan struct{}, 1),
		ChatId:          chatId,
		AuthorId:        authorId,
		RoundsNum:       DefaultRoundNum,
		timeout:         timeout,
		doneFn:          done,
		abortFn:         abort,
		controlHandlers: map[string]CommandHandlerFn{},
		doneCh:          make(chan struct{}, 1),
		CreatedAt:       time.Now(),
	}

	for _, category := range categoriesList {
		s.Categories = append(s.Categories, &Category{
			Text: category,
		})
	}

	for _, letter := range lettersList {
		s.Letters = append(s.Letters, &Letter{
			Text:   letter,
			Status: true,
		})
	}

	if err := s.bindControlCommand(commandNextText, CommandKindNextAction); err != nil {
		return nil, err
	}

	if err := s.bindControlCommand(commandPrevText, CommandKindPrevAction); err != nil {
		return nil, err
	}

	if err := s.bindControlCommand(commandDoneText, CommandKindDoneAction); err != nil {
		return nil, err
	}

	return s, nil
}

type Session struct {
	tg              *tgbotapi.BotAPI
	state           *stateMachine
	messageCh       chan struct{}
	doneCh          chan struct{}
	runSema         sync.Once
	AuthorId        int64
	Categories      []*Category
	Letters         []*Letter
	ChatId          int64
	RoundsNum       int
	messageId       int
	CreatedAt       time.Time
	timeout         time.Duration
	controlHandlers map[string]CommandHandlerFn
	cancel          func()
	abortFn         func(session *Session) error
	doneFn          func(session *Session) error
}

func (bs *Session) Run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, bs.timeout)
	bs.cancel = cancel
	bs.runSema.Do(func() {
		go bs.loop(ctx)
		bs.messageCh <- struct{}{}
	})
}

func (bs *Session) Stop() {
	bs.cancel()
}

func (bs *Session) Execute(upd tgbotapi.Update) error {
	if upd.CallbackQuery != nil {
		if err := bs.executeCbQuery(upd.CallbackQuery); err != nil {
			return err
		}
	}

	if upd.Message != nil {
		if err := bs.executeMessageQuery(upd.Message); err != nil {
			return err
		}
	}

	return nil
}

func (bs *Session) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if err := bs.abortFn(bs); err != nil {
				fmt.Println(err)
			}
			return
		case <-bs.doneCh:
			if err := bs.doneFn(bs); err != nil {
				fmt.Println(err)
			}
			return
		case <-bs.messageCh:
			switch bs.state.curr() {
			case stateKindCategories:
				msg := tgbotapi.NewMessage(bs.ChatId, textChooseCategories)
				msg.ReplyMarkup = bs.appendControlButtons(bs.categoriesMarkup())
				output, err := bs.tg.Send(msg)
				if err != nil {
					fmt.Println(err)
				}
				bs.messageId = output.MessageID
			case stateKindRounds:
				msg := tgbotapi.NewMessage(bs.ChatId, textChooseRoundsNum)
				msg.ReplyMarkup = bs.appendControlButtons(bs.roundsMarkup())
				output, err := bs.tg.Send(msg)
				if err != nil {
					fmt.Println(err)
				}
				bs.messageId = output.MessageID
			case stateKindLetters:
				msg := tgbotapi.NewMessage(bs.ChatId, textDeleteComplexLetters)
				msg.ReplyMarkup = bs.appendControlButtons(bs.lettersMarkup())
				output, err := bs.tg.Send(msg)
				if err != nil {
					fmt.Println(err)
				}
				bs.messageId = output.MessageID
			}
		}
	}
}

func (bs *Session) clickOnPrev(query *tgbotapi.CallbackQuery) error {
	bs.state.prev()
	if _, err := bs.tg.AnswerCallbackQuery(
		tgbotapi.NewCallback(query.ID, commandPrevText)); err != nil {
		return err
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnNext(query *tgbotapi.CallbackQuery) error {
	bs.state.next()
	if _, err := bs.tg.AnswerCallbackQuery(
		tgbotapi.NewCallback(query.ID, commandNextText)); err != nil {
		return err
	}
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnDone(query *tgbotapi.CallbackQuery) error {
	if _, err := bs.tg.AnswerCallbackQuery(
		tgbotapi.NewCallback(query.ID, commandDoneText)); err != nil {
		return err
	}

	if !bs.categoriesExist() {
		msg := tgbotapi.NewMessage(bs.ChatId, textAddLeastCategoryToComplete)
		if _, err := bs.tg.Send(msg); err != nil {
			return err
		}

		return nil
	}

	if !bs.lettersExist() {
		msg := tgbotapi.NewMessage(bs.ChatId, textAddLeastOneLetterToComplete)
		if _, err := bs.tg.Send(msg); err != nil {
			return err
		}

		return nil
	}

	bs.doneCh <- struct{}{}

	return nil
}

type CommandKind uint8

const (
	CommandKindPrevAction CommandKind = iota + 1
	CommandKindNextAction
	CommandKindDoneAction
)

type CommandHandlerFn func(query *tgbotapi.CallbackQuery) error

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

func (bs *Session) Handle(command string, fn CommandHandlerFn) {
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
	for _, category := range bs.Categories {
		if query.Data == category.Text {
			category.Status = !category.Status
			if category.Status {
				answer = fmt.Sprintf(textAddedCategory, category.Text)
			} else {
				answer = fmt.Sprintf(textDeletedCategory, category.Text)
			}
		}
	}

	if _, err := bs.tg.AnswerCallbackQuery(
		tgbotapi.NewCallback(query.ID, answer)); err != nil {
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatId, bs.messageId, bs.appendControlButtons(bs.categoriesMarkup()))
	if _, err := bs.tg.Send(msg); err != nil {
		return err
	}

	return nil
}

func (bs *Session) clickOnRoundsNum(query *tgbotapi.CallbackQuery) error {
	n, err := strconv.Atoi(query.Data)
	if err != nil {
		return err
	}

	if _, err := bs.tg.AnswerCallbackQuery(
		tgbotapi.NewCallback(query.ID, fmt.Sprintf(textRoundNum, n))); err != nil {
		return err
	}

	bs.RoundsNum = n
	bs.state.next()
	bs.messageCh <- struct{}{}

	return nil
}

func (bs *Session) clickOnLetter(query *tgbotapi.CallbackQuery) error {
	var answer string
	for _, letter := range bs.Letters {
		if query.Data == letter.Text {
			letter.Status = !letter.Status
			if letter.Status {
				answer = fmt.Sprintf(textAddedLetter, letter.Text)
			} else {
				answer = fmt.Sprintf(textDeletedLetter, letter.Text)
			}
		}
	}

	if _, err := bs.tg.AnswerCallbackQuery(
		tgbotapi.NewCallback(query.ID, answer)); err != nil {
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatId, bs.messageId, bs.appendControlButtons(bs.lettersMarkup()))
	if _, err := bs.tg.Send(msg); err != nil {
		return err
	}

	return nil
}

func (bs *Session) executeMessageQuery(query *tgbotapi.Message) error {
	if bs.state.curr() == stateKindCategories {
		bs.Categories = append(bs.Categories, &Category{
			Text:   query.Text,
			Status: true,
		})

		msg := tgbotapi.NewEditMessageReplyMarkup(bs.ChatId, bs.messageId, bs.appendControlButtons(bs.categoriesMarkup()))
		if _, err := bs.tg.Send(msg); err != nil {
			return err
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
			return err
		}

		return nil
	}

	switch bs.state.curr() {
	case stateKindCategories:
		if err := bs.clickOnCategories(query); err != nil {
			return err
		}
	case stateKindRounds:
		if err := bs.clickOnRoundsNum(query); err != nil {
			return err
		}
	case stateKindLetters:
		if err := bs.clickOnLetter(query); err != nil {
			return err
		}
	}

	return nil
}

func (bs *Session) lettersMarkup() tgbotapi.InlineKeyboardMarkup {
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	for _, category := range bs.Letters {
		if len(row) == 6 {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
			row = tgbotapi.NewInlineKeyboardRow()
		}
		var btn tgbotapi.InlineKeyboardButton
		if category.Status {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CheckMark.String()+" "+category.Text, category.Text)
		} else {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CrossMark.String()+" "+category.Text, category.Text)
		}

		row = append(row, btn)
	}

	if len(row) > 0 {
		markup.InlineKeyboard = append(markup.InlineKeyboard, row)
	}

	return markup
}

func (bs *Session) roundsMarkup() tgbotapi.InlineKeyboardMarkup {
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	for _, n := range roundsList {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(n), strconv.Itoa(n)))
	}
	markup.InlineKeyboard = append(markup.InlineKeyboard, row)

	return markup
}

func (bs *Session) categoriesMarkup() tgbotapi.InlineKeyboardMarkup {
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	for _, category := range bs.Categories {
		if len(row) == 3 {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
			row = tgbotapi.NewInlineKeyboardRow()
		}
		var btn tgbotapi.InlineKeyboardButton
		if category.Status {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CheckMark.String()+" "+category.Text, category.Text)
		} else {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CrossMark.String()+" "+category.Text, category.Text)
		}

		row = append(row, btn)
	}

	if len(row) > 0 {
		markup.InlineKeyboard = append(markup.InlineKeyboard, row)
	}

	return markup
}

func (bs *Session) appendControlButtons(markup tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	row := tgbotapi.NewInlineKeyboardRow()

	if !bs.state.isMin() {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(commandPrevText, commandPrevText))
	}

	if !bs.state.isMax() {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(commandNextText, commandNextText))
	} else {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(commandDoneText, commandDoneText))
	}

	markup.InlineKeyboard = append(markup.InlineKeyboard, row)

	return markup
}

func (bs *Session) lettersExist() bool {
	for _, letter := range bs.Letters {
		if letter.Status {
			return true
		}
	}

	return false
}

func (bs *Session) categoriesExist() bool {
	for _, category := range bs.Categories {
		if category.Status {
			return true
		}
	}

	return false
}
