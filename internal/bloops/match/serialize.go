package match

import "time"

type SerializedSession struct {
	Config
	Players   []*Player `json:"players"`
	CreatedAt time.Time `json:"createdAt"`
}
