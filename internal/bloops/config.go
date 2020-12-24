package bloops

import (
	"bloop/internal/database"
	"time"
)

type Config struct {
	Debug            bool          `envconfig:"BLOOP_DEBUG" default:"false"`
	CacheItems       int           `envconfig:"BLOOP_CACHE_USERS_NUM" default:"1024"`
	Token            string        `envconfig:"BLOOP_TOKEN"`
	BuildingTimeout  time.Duration `envconfig:"BLOOP_BUILDING_TIMEOUT" default:"60m"`
	PlayingTimeout   time.Duration `envconfig:"BLOOP_PLAYING_TIMEOUT" default:"24h"`
	TgBotPollTimeout time.Duration `envconfig:"BLOOP_TG_BOT_POLL_TIMEOUT" default:"60s"`
	Db               database.Config
}
