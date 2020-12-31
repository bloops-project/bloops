package match

import (
	"bloop/internal/bloopsbot/resource"
	"bloop/internal/bloopsbot/util"
	"bloop/internal/database/matchstate/model"
	"bloop/internal/logging"
	"bloop/internal/strpool"
	"context"
	"fmt"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/valyala/fastrand"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

// notification of the player's readiness and sending the start button
func (r *Session) sendStartMsg(player *model.Player) error {
	msg := tgbotapi.NewMessage(player.ChatId, r.renderStartMsg())
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(resource.TextStartBtnData, resource.TextStartBtnData),
		),
	)
	msg.ParseMode = tgbotapi.ModeMarkdown
	output, err := r.tg.Send(msg)
	if err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	r.registerCbHandler(output.MessageID, func(query *tgbotapi.CallbackQuery) error {
		if query.Data == resource.TextStartBtnData {
			if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.TextStartBtnDataAnswer)); err != nil {
				return fmt.Errorf("send answer: %v", err)
			}
			r.startCh <- struct{}{}
		}

		r.mtx.Lock()
		defer r.mtx.Unlock()
		delete(r.msgCallback, output.MessageID)

		return nil
	})

	return nil
}

func (r *Session) checkBloopsSendMsg(player *model.Player) (int, error) {
	msg := tgbotapi.NewMessage(player.ChatId, emoji.GameDie.String()+"...")
	output, err := r.tg.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("send msg: %v", err)
	}
	util.Sleep(1 * time.Second)
	for i := 3; i > 0; i-- {
		msg := tgbotapi.NewEditMessageText(player.ChatId, output.MessageID, emoji.GameDie.String()+"..."+strconv.Itoa(i))
		if _, err := r.tg.Send(msg); err != nil {
			return output.MessageID, fmt.Errorf("send msg: %v", err)
		}
		util.Sleep(1 * time.Second)
	}

	return output.MessageID, nil
}

func (r *Session) sendDroppedBloopsesMsg(player *model.Player, bloops *resource.Bloops) error {
	msg := tgbotapi.NewMessage(player.ChatId, r.renderDropBloopsMsg(bloops))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(resource.TextChallengeBtnDataAnswer, resource.TextChallengeBtnDataAnswer),
		),
	)

	msg.ParseMode = tgbotapi.ModeMarkdown
	output, err := r.tg.Send(msg)
	if err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	r.registerCbHandler(output.MessageID, func(query *tgbotapi.CallbackQuery) error {
		if query.Data == resource.TextChallengeBtnDataAnswer {
			if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, resource.TextChallengeBtnDataAnswer)); err != nil {
				return fmt.Errorf("send answer: %v", err)
			}
			r.startCh <- struct{}{}
		}

		r.mtx.Lock()
		defer r.mtx.Unlock()
		delete(r.msgCallback, output.MessageID)

		return nil
	})

	return nil
}

// select the letter that the player needs to call the words
func (r *Session) sendLetterMsg(player *model.Player) error {
	buf := strpool.Get()

	output, err := r.tg.Send(tgbotapi.NewMessage(player.ChatId, resource.TextStartLetterMsg))
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
		for buf.String() == sentMsg {
			buf.Reset()
			idx := fastrand.Uint32n(uint32(len(r.Config.Letters)))
			buf.WriteString(resource.TextStartLetterMsg)
			buf.WriteString(r.Config.Letters[idx])
			sentLetter = r.Config.Letters[idx]
		}

		sndCh <- buf.String()
		sentMsg = buf.String()
		util.Sleep(300 * time.Millisecond)
	}

	buf.Reset()
	strpool.Put(buf)

	r.renderStartHelpMsg(player, sentLetter)

	r.syncBroadcast(buf.String(), player.UserId)

	close(sndCh)

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// send ready -> set -> go steps
func (r *Session) sendReadyMsg(player *model.Player) error {
	var messageId int
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.Keycap3.String())
	buf.WriteString(" ...")
	{
		msg := tgbotapi.NewMessage(player.ChatId, buf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown

		output, err := r.tg.Send(msg)
		if err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
		messageId = output.MessageID
		util.Sleep(1 * time.Second)
	}

	buf.Reset()
	buf.WriteString(emoji.Keycap2.String())
	buf.WriteString(" На старт")
	{
		msg := tgbotapi.NewEditMessageText(player.ChatId, messageId, buf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		util.Sleep(1 * time.Second)
	}

	buf.Reset()
	buf.WriteString(emoji.Keycap1.String())
	buf.WriteString(" Внимание")

	{
		msg := tgbotapi.NewEditMessageText(player.ChatId, messageId, buf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		util.Sleep(1 * time.Second)
	}

	buf.Reset()
	buf.WriteString(emoji.Rocket.String())
	buf.WriteString(" Марш!")

	{
		msg := tgbotapi.NewEditMessageText(player.ChatId, messageId, buf.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := r.tg.Send(msg); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}
	}

	return nil
}

func (r *Session) sendFreezeTimerMsg(player *model.Player, secs int) (int, error) {
	var messageId int
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(secs))
	buf.WriteString(" сек")
	msg := tgbotapi.NewMessage(player.ChatId, resource.TextStopButton)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buf.String(), resource.TextTimerBtnData),
			tgbotapi.NewInlineKeyboardButtonData(resource.TextStopBtnData, resource.TextStopBtnData),
		),
	)

	output, err := r.tg.Send(msg)
	if err != nil {
		return messageId, fmt.Errorf("send msg: %v", err)
	}

	return output.MessageID, nil
}

// formatting stop, timer button and send it
func (r *Session) sendWorkingTimerMsg(player *model.Player, messageId, secs int) error {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(secs))
	buf.WriteString(" сек")

	msg := tgbotapi.NewEditMessageReplyMarkup(
		player.ChatId,
		messageId,
		tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buf.String(), resource.TextTimerBtnData),
				tgbotapi.NewInlineKeyboardButtonData(resource.TextStopBtnData, resource.TextStopBtnData),
			),
		),
	)

	if _, err := r.tg.Send(msg); err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	return nil
}

func (r *Session) sendVotesMsg(voteMessages map[int64]int) error {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			thumbUpButton(r.activeVote.thumbUp),
			thumbDownButton(r.activeVote.thumbDown),
		),
	)

	// creating a voting system and defining callbacks for voting
	for _, player := range r.Players {
		if player.IsPlaying() && !player.Offline {
			msg := tgbotapi.NewMessage(player.ChatId, resource.TextVoteMsg)
			msg.ReplyMarkup = markup
			// sending the thumbs up and thumbs down buttons
			output, err := r.tg.Send(msg)
			if err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
			// registering callbacks for voting
			voteMessages[player.ChatId] = output.MessageID
			r.registerCbHandler(output.MessageID, func(query *tgbotapi.CallbackQuery) error {
				switch query.Data {
				case resource.TextThumbUp:
					r.thumbUp()
				case resource.TextThumbDown:
					r.thumbDown()
				default:
				}

				if _, err := r.tg.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, query.Data)); err != nil {
					return fmt.Errorf("send answer msg: %v", err)
				}

				return nil
			})
		}
	}

	return nil
}

func (r *Session) sendChangingVotesMsg(voteMessages map[int64]int) error {
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

func (r *Session) sendRoundClosed() {
	r.syncBroadcast(fmt.Sprintf(resource.TextRoundFavoriteMsg, r.CurrRoundIdx+1))
}

func (r *Session) sendWhoFavoritesMsg() {
	favorites := r.Favorites()

	r.asyncBroadcast(r.renderGameFavorites(favorites))
}

func (r *Session) sendCrashMsg() {
	r.syncBroadcast(resource.TextBroadcastCrashMsg)
}

const (
	maxSmallCellsRow = 6
	maxLargeCellsRow = 3
	maxXlCellsRow    = 2
)

const (
	treasuresNum = 3
	rewardsNum   = 6
	attempts     = 1
)

func newOpenedReward() *openedReward {
	return &openedReward{items: map[int]struct{}{}}
}

type openedReward struct {
	mtx sync.RWMutex

	items map[int]struct{}
}

func (o *openedReward) assign(n int) {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	o.items[n] = struct{}{}
}

func (o *openedReward) exist(n int) bool {
	o.mtx.RLock()
	defer o.mtx.RUnlock()
	_, ok := o.items[n]
	return ok
}

func (o *openedReward) equal(n int) bool {
	o.mtx.RLock()
	defer o.mtx.RUnlock()
	return len(o.items) == n
}

func (r *Session) sendChoiceBloopsMsg(ctx context.Context, player *model.Player) error {
	bloops := make([]string, rewardsNum)

	for i := 0; i < rewardsNum; i++ {
		if i < treasuresNum {
			idx := fastrand.Uint32n(uint32(len(resource.Bloopses)))
			bloops[i] = resource.Bloopses[idx].Name
		} else {
			bloops[i] = "Обычный раунд"
		}
	}

	rand.Shuffle(len(bloops), func(i, j int) {
		bloops[i], bloops[j] = bloops[j], bloops[i]
	})

	markup := tgbotapi.NewInlineKeyboardMarkup()
	{
		row := tgbotapi.NewInlineKeyboardRow()
		for idx := range bloops {
			if len(row) == maxXlCellsRow {
				markup.InlineKeyboard = append(markup.InlineKeyboard, row)
				row = tgbotapi.NewInlineKeyboardRow()
			}

			row = append(row, tgbotapi.NewInlineKeyboardButtonData(emoji.GemStone.String()+" Неизвестная карта", strconv.Itoa(idx)))
		}

		if len(row) > 0 {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
		}
	}

	msg := tgbotapi.NewMessage(player.ChatId, "Выбери карту, тебе может попасться блюпс")
	msg.ReplyMarkup = markup
	output, err := r.tg.Send(msg)
	if err != nil {
		return fmt.Errorf("send msg: %v", err)
	}

	opened := newOpenedReward()

	mtx := sync.RWMutex{}
	r.registerCbHandler(output.MessageID, func(query *tgbotapi.CallbackQuery) error {
		logger := logging.FromContext(ctx).Named("match.sendChoiceBloopsMsg")
		defer func() {
			if opened.equal(attempts) {
				util.Sleep(3 * time.Second)

				r.mtx.Lock()
				delete(r.msgCallback, output.MessageID)
				r.mtx.Unlock()

				if _, err := r.tg.Send(tgbotapi.NewDeleteMessage(player.ChatId, output.MessageID)); err != nil {
					logger.Errorf("send msg: %v", err)
				}

				return
			}
		}()

		if opened.equal(attempts) {
			return nil
		}

		n, err := strconv.Atoi(query.Data)
		if err != nil {
			return fmt.Errorf("strconv: %v", err)
		}

		mtx.RLock()

		var cbConfig tgbotapi.CallbackConfig
		if bloops[n] != emoji.CrossMark.String() {
			cbConfig = tgbotapi.NewCallback(query.ID, "Нашел!")
		} else {
			cbConfig = tgbotapi.NewCallback(query.ID, "Тут ничего!")
		}

		mtx.RUnlock()
		if _, err := r.tg.AnswerCallbackQuery(cbConfig); err != nil {
			return fmt.Errorf("send answer: %v", err)
		}

		opened.assign(n)

		markup := tgbotapi.NewInlineKeyboardMarkup()
		row := tgbotapi.NewInlineKeyboardRow()
		mtx.RLock()
		for idx := range bloops {
			if len(row) == maxXlCellsRow {
				markup.InlineKeyboard = append(markup.InlineKeyboard, row)
				row = tgbotapi.NewInlineKeyboardRow()
			}
			if opened.exist(idx) {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(bloops[idx], bloops[idx]))
			} else {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(emoji.GemStone.String()+" Неизвестная карточка", strconv.Itoa(idx)))
			}
		}

		mtx.RUnlock()
		if len(row) > 0 {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
		}

		if _, err := r.tg.Send(tgbotapi.NewEditMessageReplyMarkup(player.ChatId, output.MessageID, markup)); err != nil {
			return fmt.Errorf("send msg: %v", err)
		}

		return nil
	})

	return nil
}
