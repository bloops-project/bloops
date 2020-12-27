package match

import (
	"bloop/internal/bloopsbot/resource"
	"bloop/internal/database/matchstate/model"
	"bloop/internal/strpool"
	"fmt"
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

func (r *Session) renderDropBloopsMsg(challenge *resource.Bloops) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.PartyPopper.String())
	buf.WriteString(" *БЛЮПС!*\n\n")
	buf.WriteString(fmt.Sprintf("*%s*", challenge.Name))
	buf.WriteString("\n")
	buf.WriteString(challenge.Task)
	buf.WriteString("\n\n")
	buf.WriteString(emoji.HundredPoints.String())
	buf.WriteString(" ")
	if challenge.Points >= 0 {
		buf.WriteString("+")
		buf.WriteString(strconv.Itoa(challenge.Points))
	} else {
		buf.WriteString(strconv.Itoa(challenge.Points))
	}
	buf.WriteString(" очков\n")
	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" ")
	if challenge.Seconds >= 0 {
		buf.WriteString("+")
		buf.WriteString(strconv.Itoa(challenge.Seconds))
	} else {
		buf.WriteString(strconv.Itoa(challenge.Seconds))
	}
	buf.WriteString(" сек\n\n")
	buf.WriteString("Расскажи о блюпсе игрокам и постарайся выполнить")

	return buf.String()
}

func (r *Session) renderStartMsg() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.GameDie.String())
	buf.WriteString(" Готов сыграть?\n\n")
	buf.WriteString("Нужно назвать все слова из списка категорий на выпавшую букву\n\n")
	buf.WriteString(emoji.Pen.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(r.Config.Categories)))
	buf.WriteString(" слов\n")
	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(r.currRoundSeconds))
	buf.WriteString(" секунд\n\n")
	buf.WriteString(emoji.CardIndex.String())
	buf.WriteString(" Категории:\n\n")
	buf.WriteString(r.renderCategories())
	buf.WriteString("\n\n")
	buf.WriteString(resource.TextClickStartBtnMsg)

	return buf.String()
}

func (r *Session) renderGameFavorites(favorites []PlayerScore) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.ChequeredFlag.String())
	buf.WriteString(" Игра завершена\n\n")
	buf.WriteString("*Список победителей*\n\n")

	for _, score := range favorites {
		buf.WriteString(emoji.SportsMedal.String())
		buf.WriteString(" ")
		buf.WriteString(score.Player.FormatFirstName())
		buf.WriteString(" - ")
		buf.WriteString(strconv.Itoa(score.Points))
		buf.WriteString(" очков\n")
	}

	return buf.String()
}

func (r *Session) renderScores() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(emoji.Trophy.String())
	buf.WriteString(" ")
	buf.WriteString(resource.TextLeaderboardHeader)

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
		buf.WriteString(strconv.Itoa(n + 1))
		buf.WriteString(". ")
		buf.WriteString(medalIcon(n))
		buf.WriteString("*")
		buf.WriteString(cell.Player.FormatFirstName())
		buf.WriteString("*, ")
		buf.WriteString(strconv.Itoa(cell.Points))
		buf.WriteString(" очков, ")
		buf.WriteString(strconv.Itoa(len(cell.Player.Rates)))
		buf.WriteString("/")
		buf.WriteString(strconv.Itoa(r.Config.RoundsNum))
		buf.WriteString("\n")
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
	buf.WriteString(emoji.Gear.String())
	buf.WriteString(" *Параметры*\n\n")
	buf.WriteString(emoji.ChequeredFlag.String())
	buf.WriteString(" Количество раундов: ")
	buf.WriteString(strconv.Itoa(r.Config.RoundsNum))
	buf.WriteString("\n")
	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" Время раунда: ")
	buf.WriteString(strconv.Itoa(r.Config.RoundTime))
	buf.WriteString(" сек\n")
	buf.WriteString(emoji.GemStone.String())
	buf.WriteString(" Блюпсы: ")
	if len(r.Config.Bloopses) > 0 {
		buf.WriteString("да")
	} else {
		buf.WriteString("нет")
	}
	buf.WriteString("\n")
	buf.WriteString(emoji.Loudspeaker.String())
	buf.WriteString(" Голосование: ")
	if r.Config.Vote {
		buf.WriteString("да")
	} else {
		buf.WriteString("нет")
	}
	buf.WriteString("\n\n")
	buf.WriteString(emoji.CardIndex.String())
	buf.WriteString(" Категории\n")
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
	buf.WriteString(player.FormatFirstName())
	buf.WriteString(" набирает ")
	buf.WriteString(strconv.Itoa(points))
	buf.WriteString("очков")

	return buf.String()
}
