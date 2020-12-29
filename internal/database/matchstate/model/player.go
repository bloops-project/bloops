package model

import (
	userModel "bloop/internal/database/user/model"
	"bloop/internal/strpool"
	"github.com/enescakir/emoji"
	"strconv"
)

type PlayerStateKind uint8

const (
	PlayerStateKindPlaying PlayerStateKind = iota + 1
	PlayerStateKindLeaving
)

func NewPlayer(chatId int64, user userModel.User, offline bool) *Player {
	return &Player{
		User:    user,
		UserId:  user.Id,
		ChatId:  chatId,
		Rates:   []*Rate{},
		State:   PlayerStateKindPlaying,
		Offline: offline,
	}
}

type Player struct {
	User    userModel.User  `json:"user"`
	State   PlayerStateKind `json:"state"`
	Offline bool            `json:"offline"`
	ChatId  int64           `json:"chatId"`
	UserId  int64           `json:"userId"`
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
		buf.WriteString(strconv.Itoa(p.User.Stars))
		buf.WriteString(emoji.Star.String())
	}

	return buf.String()
}
