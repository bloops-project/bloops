package match

import (
	"bloop/internal/bloops/resource"
	statDb "bloop/internal/stat/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"time"
)

type Config struct {
	AuthorId   int64                        `json:"authorId"`
	AuthorName string                       `json:"authorName"`
	RoundsNum  int                          `json:"roundsNum"`
	RoundTime  int                          `json:"roundTime"`
	Categories []string                     `json:"categories"`
	Letters    []string                     `json:"letters"`
	Bloopses   []resource.Bloops            `json:"bloopses"`
	Vote       bool                         `json:"vote"`
	Code       int64                        `json:"code"`
	Tg         *tgbotapi.BotAPI             `json:"-"`
	DoneFn     func(session *Session) error `json:"-"`
	Timeout    time.Duration                `json:"-"`
	StatDb     *statDb.DB                   `json:"-"`
}

func (c Config) IsBloops() bool {
	return len(c.Bloopses) > 0
}
