package bloopmp

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

func NewUser(usr *tgbotapi.User) *User {
	return &User{
		Id:           int64(usr.ID),
		FirstName:    usr.FirstName,
		LastName:     usr.LastName,
		LanguageCode: usr.LanguageCode,
		Username:     usr.UserName,
	}
}

type User struct {
	Id           int64
	FirstName    string
	LastName     string
	LanguageCode string
	Username     string
}
