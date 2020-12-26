package model

import "time"

type Rate struct {
	Duration   time.Duration `json:"duration"`
	Points     int           `json:"points"`
	Completed  bool          `json:"completed"`
	Bloops     bool          `json:"bloopsbot"`
	BloopsName string        `json:"bloopsName"`
}
