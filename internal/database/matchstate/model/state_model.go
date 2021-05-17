package model

import (
	"github.com/bloops-games/bloops/internal/bloopsbot/resource"
	"time"
)

type State struct {
	Timeout    time.Duration     `json:"timeout"`
	AuthorId   int64             `json:"authorId"`
	AuthorName string            `json:"authorName"`
	RoundsNum  int               `json:"roundsNum"`
	RoundTime  int               `json:"roundTime"`
	Categories []string          `json:"categories"`
	Letters    []string          `json:"letters"`
	Bloopses   []resource.Bloops `json:"bloopses"`
	Vote       bool              `json:"vote"`
	Code       int64             `json:"code"`

	State        uint8     `json:"state"`
	CurrRoundIdx int       `json:"currRoundIdx"`
	Players      []*Player `json:"players"`

	CreatedAt time.Time `json:"createdAt"`
}
