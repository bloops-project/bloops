package match

import (
	"bloop/internal/bloops/resource"
	statDb "bloop/internal/stat/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"time"
)

type Config struct {
	AuthorId   int64
	RoundsNum  int
	RoundTime  int
	Categories []string
	Letters    []string
	Bloopses   []resource.Bloops
	Vote       bool
	Tg         *tgbotapi.BotAPI
	Code       int64
	DoneFn     func(session *Session) error
	Timeout    time.Duration
	StatDb     *statDb.DB
}

func (c Config) IsBloops() bool {
	return len(c.Bloopses) > 0
}
