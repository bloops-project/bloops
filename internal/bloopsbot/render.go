package bloopsbot

import (
	"fmt"
	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	statModel "github.com/bloops-games/bloops/internal/database/stat/model"
	userModel "github.com/bloops-games/bloops/internal/database/user/model"
	"github.com/bloops-games/bloops/internal/strpool"
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
	_, _ = fmt.Fprintf(buf, "%s Профиль игрока *%s*\n\n", emoji.Alien.String(), u.FirstName)
	_, _ = fmt.Fprintf(buf, "%s Сыграно: %s\n", emoji.VideoGame.String(), strconv.Itoa(stat.Count))
	_, _ = fmt.Fprintf(buf, "%s Побед: %s\n", emoji.Star.String(), strconv.Itoa(stat.Stars))
	_, _ = fmt.Fprintf(
		buf,
		"%s Блюпсов открыто: %s/%s\n",
		emoji.GemStone.String(),
		strconv.Itoa(len(stat.Bloops)),
		strconv.Itoa(len(resource.BloopsKeys)),
	)
	_, _ = fmt.Fprintf(
		buf,
		"%s Лучшее время раунда: %s\n",
		emoji.Stopwatch.String(),
		stat.BestDuration.Round(100*time.Millisecond).String(),
	)
	_, _ = fmt.Fprintf(
		buf,
		"%s Среднее время раунда: %s\n",
		emoji.Stopwatch.String(),
		stat.AvgDuration.Round(100*time.Millisecond).String(),
	)
	_, _ = fmt.Fprintf(buf, "%s Лучший счет раунда: %s", emoji.HundredPoints.String(), strconv.Itoa(stat.BestPoints))

	return buf.String()
}
