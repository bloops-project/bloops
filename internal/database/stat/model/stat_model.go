package model

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusFavorite    Status = "favorite"
	StatusParticipant Status = "participant"
)

func NewStat(userID int64) Stat {
	return Stat{ID: uuid.New(), UserID: userID, Conclusion: StatusParticipant, CreatedAt: time.Now()}
}

type Stat struct {
	ID     uuid.UUID `json:"-"`
	UserID int64     `json:"userID"`

	WorstDuration   time.Duration `json:"worstDuration"`
	AverageDuration time.Duration `json:"averageDuration"`
	BestDuration    time.Duration `json:"bestDuration"`
	SumDuration     time.Duration `json:"sumDuration"`

	AveragePoints int `json:"averagePoints"`
	SumPoints     int `json:"points"`
	WorstPoints   int `json:"worstPoints"`
	BestPoints    int `json:"bestPoints"`

	RoundsNum  int       `json:"roundsNum"`
	Conclusion Status    `json:"conclusion"`
	Categories []string  `json:"categories"`
	Bloops     []string  `json:"bloopsbot"`
	PlayersNum int       `json:"playersNum"`
	Vote       bool      `json:"vote"`
	CreatedAt  time.Time `json:"createdAt"`
}

type RateStat struct {
	Stars  int
	Bloops int
}

type AggregationStat struct {
	Count         int
	Stars         int
	Bloops        []string
	AvgDuration   time.Duration
	WorstDuration time.Duration
	BestDuration  time.Duration
	AvgPoints     int
	BestPoints    int
	WorstPoints   int
}
