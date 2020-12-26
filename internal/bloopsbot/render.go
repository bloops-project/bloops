package bloopsbot

import (
	"bloop/internal/bloopsbot/resource"
	statModel "bloop/internal/database/stat/model"
	userModel "bloop/internal/database/user/model"
	"bloop/internal/strpool"
	"github.com/enescakir/emoji"
	"strconv"
	"time"
)

func renderProfile(u userModel.User, stat statModel.AggregationStat) string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()
	buf.WriteString(emoji.Alien.String())
	buf.WriteString(" Профиль игрока ")
	buf.WriteString("*")
	buf.WriteString(u.FirstName)
	buf.WriteString("*")
	buf.WriteString("\n\n")
	buf.WriteString(emoji.VideoGame.String())
	buf.WriteString(" Сыграно: ")
	buf.WriteString(strconv.Itoa(stat.Count))
	buf.WriteString("\n")
	buf.WriteString(emoji.Star.String() + " Побед: ")
	buf.WriteString(strconv.Itoa(stat.Stars))
	buf.WriteString("\n")
	buf.WriteString(emoji.GemStone.String() + " Блюпсов открыто: ")
	buf.WriteString(strconv.Itoa(len(stat.Bloops)))
	buf.WriteString("/")
	buf.WriteString(strconv.Itoa(len(resource.Bloopses)))
	buf.WriteString("\n")
	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" Лучшее время раунда: ")
	buf.WriteString(stat.BestDuration.Round(100 * time.Millisecond).String())
	buf.WriteString("\n")
	buf.WriteString(emoji.Stopwatch.String())
	buf.WriteString(" Среднее время раунда: ")
	buf.WriteString(stat.AvgDuration.Round(100 * time.Millisecond).String())
	buf.WriteString("\n")
	buf.WriteString(emoji.HundredPoints.String())
	buf.WriteString(" Лучший счет раунда: ")
	buf.WriteString(strconv.Itoa(stat.BestPoints))

	return buf.String()
}
