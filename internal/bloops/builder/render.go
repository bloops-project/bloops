package builder

import (
	"bloop/internal/bloops/resource"
	"fmt"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
)

const (
	maxSmallCellsRow = 6
	maxLargeCellsRow = 3
)

func (bs *Session) renderInlineBloops() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(resource.TextVoteYes, "true"),
		tgbotapi.NewInlineKeyboardButtonData(resource.TextVoteNo, "false"),
	))
}

func (bs *Session) renderInlineVote() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(resource.TextVoteYes, "true"),
		tgbotapi.NewInlineKeyboardButtonData(resource.TextVoteNo, "false"),
	))
}

func (bs *Session) renderInlineLetters() tgbotapi.InlineKeyboardMarkup {
	var btn tgbotapi.InlineKeyboardButton
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	for _, letter := range bs.Letters {
		if len(row) == maxSmallCellsRow {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
			row = tgbotapi.NewInlineKeyboardRow()
		}

		if letter.Status {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CheckMarkButton.String()+" "+letter.Text, letter.Text)
		} else {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CrossMark.String()+" "+letter.Text, letter.Text)
		}

		row = append(row, btn)
	}

	if len(row) > 0 {
		markup.InlineKeyboard = append(markup.InlineKeyboard, row)
	}

	return markup
}

func (bs *Session) renderRoundsTime() tgbotapi.InlineKeyboardMarkup {
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	for _, n := range resource.RoundTimes {
		row = append(
			row,
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %d", emoji.Stopwatch.String(), n),
				strconv.Itoa(n),
			),
		)
	}
	markup.InlineKeyboard = append(markup.InlineKeyboard, row)

	return markup
}

func (bs *Session) renderRoundsNum() tgbotapi.InlineKeyboardMarkup {
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	var numFn = func(n int) string {
		var str string
		switch n {
		case 1:
			str = emoji.Keycap1.String()
		case 2:
			str = emoji.Keycap2.String()
		case 3:
			str = emoji.Keycap3.String()
		case 4:
			str = emoji.Keycap4.String()
		case 5:
			str = emoji.Keycap5.String()
		}

		return str
	}
	for _, n := range resource.RoundsNum {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(numFn(n), strconv.Itoa(n)))
	}
	markup.InlineKeyboard = append(markup.InlineKeyboard, row)

	return markup
}

func (bs *Session) renderInlineCategories() tgbotapi.InlineKeyboardMarkup {
	var btn tgbotapi.InlineKeyboardButton
	markup := tgbotapi.NewInlineKeyboardMarkup()
	row := tgbotapi.NewInlineKeyboardRow()
	for _, category := range bs.Categories {
		if len(row) == maxLargeCellsRow {
			markup.InlineKeyboard = append(markup.InlineKeyboard, row)
			row = tgbotapi.NewInlineKeyboardRow()
		}

		if category.Status {
			btn = tgbotapi.NewInlineKeyboardButtonData(emoji.CheckMarkButton.String()+" "+category.Text, category.Text)
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

func (bs *Session) menuInlineButtons(markup tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	row := tgbotapi.NewInlineKeyboardRow()

	if !bs.state.isMin() {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(resource.BuilderInlinePrevText, resource.BuilderInlinePrevData))
	}

	if !bs.state.isMax() {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(resource.BuilderInlineNextText, resource.BuilderInlineNextData))
	} else {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(resource.BuilderInlineDoneText, resource.BuilderInlineDoneData))
	}

	markup.InlineKeyboard = append(markup.InlineKeyboard, row)

	return markup
}
