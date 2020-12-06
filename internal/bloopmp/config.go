package bloopmp

import "time"

type Config struct {
	Debug        bool          `envconfig:"BLOOP_DEBUG" default:"true"`
	BuildingTime time.Duration `envconfig:"BLOOP_BUILDING_TIME" default:"60m"`
	PlayingTime  time.Duration `envconfig:"BLOOP_PLAYING_TIME" default:"24h"`
}
