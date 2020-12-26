package model

import "time"

type User struct {
	Id           int64     `json:"id"`
	Admin        bool      `json:"admin"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	LanguageCode string    `json:"languageCode"`
	Username     string    `json:"username"`
	CreatedAt    time.Time `json:"createdAt"`
	Stars        int
	Bloops       int
}
