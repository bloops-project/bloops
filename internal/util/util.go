package util

import (
	"time"
)

func Sleep(t time.Duration) {
	timer := time.NewTimer(t)
	defer timer.Stop()
	<-timer.C
}
