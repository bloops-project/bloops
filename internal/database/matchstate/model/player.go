package model

import (
	"strconv"

	userModel "github.com/bloops-games/bloops/internal/database/user/model"
	"github.com/bloops-games/bloops/internal/strpool"
	"github.com/enescakir/emoji"
)

type PlayerStateKind uint8

const (
	PlayerStateKindPlaying PlayerStateKind = iota + 1
	PlayerStateKindLeaving
)

func NewPlayer(chatID int64, user userModel.User, offline bool) *Player {
	return &Player{
		User:    user,
		UserID:  user.ID,
		ChatID:  chatID,
		Rates:   []*Rate{},
		State:   PlayerStateKindPlaying,
		Offline: offline,
	}
}

type Player struct {
	User    userModel.User  `json:"user"`
	State   PlayerStateKind `json:"state"`
	Offline bool            `json:"offline"`
	ChatID  int64           `json:"chatId"`
	UserID  int64           `json:"userID"`
	Rates   []*Rate         `json:"rates"`
}

func (p *Player) IsPlaying() bool {
	return p.State == PlayerStateKindPlaying
}

func (p *Player) FormatFirstName() string {
	buf := strpool.Get()
	defer func() {
		buf.Reset()
		strpool.Put(buf)
	}()

	buf.WriteString(p.User.FirstName)
	if !p.Offline && p.User.Stars > 0 {
		buf.WriteString(" - ")
		buf.WriteString("(")
		buf.WriteString(strconv.Itoa(p.User.Stars))
		buf.WriteString(emoji.Star.String())
		buf.WriteString(")")
	}

	return buf.String()
}
