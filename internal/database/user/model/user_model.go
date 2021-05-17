package model

import "time"

type Status uint8

const (
	StatusActive Status = iota + 1
	StatusBanned
)

type User struct {
	ID           int64     `json:"id"`
	Admin        bool      `json:"admin"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	LanguageCode string    `json:"languageCode"`
	Username     string    `json:"username"`
	CreatedAt    time.Time `json:"createdAt"`
	Status       Status    `json:"banned"`
	Stars        int
	Bloops       int
}
