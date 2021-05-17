package match

import (
	"fmt"
	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	"github.com/bloops-games/bloops/internal/database/matchstate/model"
	"github.com/bloops-games/bloops/internal/strpool"
	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
)

func thumbUpButton(n int) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d %s", n, resource.TextThumbUp), resource.TextThumbUp)
}

func thumbDownButton(n int) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d %s", n, resource.TextThumbDown), resource.TextThumbDown)
}

func (r *Session) renderDropBloopsMsg(bloops *resource.Bloops) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	_, _ = fmt.Fprintf(buf, "%s *БЛЮПС!*\n\n", emoji.PartyPopper.String())
	_, _ = fmt.Fprintf(buf, "*%s*\n", bloops.Name)
	_, _ = fmt.Fprintf(buf, "%s\n\n", bloops.Task)
	_, _ = fmt.Fprintf(buf, "%s ", emoji.HundredPoints.String())

	if bloops.Points >= 0 {
		_, _ = fmt.Fprintf(buf, "+%s", strconv.Itoa(bloops.Points))
	} else {
		buf.WriteString(strconv.Itoa(bloops.Points))
	}

	_, _ = fmt.Fprintf(buf, " очков\n%s ", emoji.Stopwatch.String())

	if bloops.Seconds >= 0 {
		_, _ = fmt.Fprintf(buf, "+%s", strconv.Itoa(bloops.Seconds))
	} else {
		buf.WriteString(strconv.Itoa(bloops.Seconds))
	}
	buf.WriteString(" сек\n\nРасскажи о блюпсе игрокам и постарайся выполнить")

	return buf.String()
}

func (r *Session) renderStartMsg() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	_, _ = fmt.Fprintf(buf, "%s Готов сыграть?\n\n", emoji.GameDie.String())
	_, _ = fmt.Fprintf(buf, "Нужно назвать все слова из списка категорий на выпавшую букву\n\n")
	_, _ = fmt.Fprintf(buf, "%s %s слов\n", emoji.Pen.String(), strconv.Itoa(len(r.Config.Categories)))
	_, _ = fmt.Fprintf(buf, "%s %s секунд\n\n", emoji.Stopwatch.String(), strconv.Itoa(r.currRoundSeconds))
	_, _ = fmt.Fprintf(buf, "%s Категории:\n\n", emoji.CardIndex.String())
	_, _ = fmt.Fprintf(buf, "%s\n\n%s", r.renderCategories(), resource.TextClickStartBtnMsg)

	return buf.String()
}

func (r *Session) renderGameFavorites(favorites []PlayerScore) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	_, _ = fmt.Fprintf(buf, "%s Игра завершена\n\n*Список победителей*\n\n", emoji.ChequeredFlag.String())

	for _, score := range favorites {
		_, _ = fmt.Fprintf(
			buf,
			"%s %s - %s очков\n",
			emoji.SportsMedal.String(),
			score.Player.FormatFirstName(),
			strconv.Itoa(score.Points),
		)
	}

	return buf.String()
}

func (r *Session) renderScores() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	_, _ = fmt.Fprintf(buf, "%s %s", emoji.Trophy.String(), resource.TextLeaderboardHeader)

	var medalIcon = func(n int) string {
		var medal string
		if n == 0 {
			medal = emoji.FirstPlaceMedal.String()
		} else if n == 1 {
			medal = emoji.SecondPlaceMedal.String()
		} else if n == 2 {
			medal = emoji.ThirdPlaceMedal.String()
		}

		return medal
	}

	for n, cell := range r.Scores() {
		_, _ = fmt.Fprintf(
			buf,
			"%s. %s*%s*, %s очков, %s/%s\n",
			strconv.Itoa(n+1),
			medalIcon(n),
			cell.Player.FormatFirstName(),
			strconv.Itoa(cell.Points),
			strconv.Itoa(len(cell.Player.Rates)),
			strconv.Itoa(r.Config.RoundsNum),
		)
	}

	return buf.String()
}

func (r *Session) renderCategories() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	for i, category := range r.Config.Categories {
		buf.WriteString(strconv.Itoa(i + 1))
		buf.WriteString(". ")
		buf.WriteString(category)
		buf.WriteString("\n")
	}

	return buf.String()
}

func (r *Session) renderSetting() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()
	_, _ = fmt.Fprintf(buf, "%s *Параметры*\n\n", emoji.Gear.String())
	_, _ = fmt.Fprintf(
		buf,
		"%s  Количество раундов: %s\n",
		emoji.ChequeredFlag.String(),
		strconv.Itoa(r.Config.RoundsNum),
	)
	_, _ = fmt.Fprintf(buf, "%s Время раунда: %s сек\n", emoji.Stopwatch.String(), strconv.Itoa(r.Config.RoundTime))
	_, _ = fmt.Fprintf(buf, "%s Блюпсы: ", emoji.GemStone.String())

	if len(r.Config.Bloopses) > 0 {
		buf.WriteString("да")
	} else {
		buf.WriteString("нет")
	}
	buf.WriteString("\n")
	_, _ = fmt.Fprintf(buf, "%s Голосование: ", emoji.Loudspeaker.String())

	if r.Config.Vote {
		buf.WriteString("да")
	} else {
		buf.WriteString("нет")
	}

	buf.WriteString("\n\n")
	_, _ = fmt.Fprintf(buf, "%s Категории\n", emoji.CardIndex.String())
	buf.WriteString(r.renderCategories())

	return buf.String()
}

func (r *Session) renderPlayers() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	for i, player := range r.Players {
		buf.WriteString(player.FormatFirstName())
		if i < len(r.Players)-1 {
			buf.WriteString(",")
		}
	}

	return buf.String()
}

func (r *Session) renderPlayerGetPoints(player *model.Player, points int) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	_, _ = fmt.Fprintf(buf, "%s набирает %s очков", player.FormatFirstName(), strconv.Itoa(points))

	return buf.String()
}

func (r *Session) renderStartHelpMsg(player *model.Player, sentLetter string) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	_, _ = fmt.Fprintf(buf, "Игрок  %s должен назвать слова:\n\n", player.FormatFirstName())
	_, _ = fmt.Fprintf(buf, "%s\n\n", r.renderCategories())
	_, _ = fmt.Fprintf(buf, "На букву: *%s*", sentLetter)

	return buf.String()
}
