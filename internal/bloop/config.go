package bloop

import "time"

type Config struct {
	Debug            bool          `envconfig:"BLOOP_DEBUG" default:"true"`
	CacheUsersNum    int           `envconfig:"BLOOP_CACHE_USERS_NUM" default:"256"`
	Token            string        `envconfig:"BLOOP_TOKEN"`
	BuildingTimeout  time.Duration `envconfig:"BLOOP_BUILDING_TIMEOUT" default:"60m"`
	PlayingTimeout   time.Duration `envconfig:"BLOOP_PLAYING_TIMEOUT" default:"24h"`
	TgBotPollTimeout time.Duration `envconfig:"BLOOP_TG_BOT_POLL_TIMEOUT" default:"60s"`
}
