package match

import (
	"bloop/internal/database/matchstate/model"
	"time"
)

type SerializedSession struct {
	Config
	Players []*model.Player `json:"players"`

	CreatedAt time.Time `json:"createdAt"`
}
