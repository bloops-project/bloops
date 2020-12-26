package bloopsbot

import (
	"bloop/internal/database"
	"time"
)

type Config struct {
	Admin string `envconfig:"BLOOP_ADMIN_USERNAME" default:"false"`
	// Logging all requests and responses from telegram
	Debug bool `envconfig:"BLOOP_DEBUG" default:"false"`
	// Number of items in the cache
	CacheSize int `envconfig:"BLOOP_CACHE_SIZE" default:"1024"`
	// Telegram bot token
	Token string `envconfig:"BLOOP_TOKEN"`
	// Waiting time to complete the game creation session
	BuildingTimeout time.Duration `envconfig:"BLOOP_BUILDING_TIMEOUT" default:"60m"`
	// Waiting time for the game session to end
	PlayingTimeout   time.Duration `envconfig:"BLOOP_PLAYING_TIMEOUT" default:"24h"`
	TgBotPollTimeout time.Duration `envconfig:"BLOOP_TG_BOT_POLL_TIMEOUT" default:"60s"`
	Db               database.Config
}
