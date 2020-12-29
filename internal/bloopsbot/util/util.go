package util

import (
	"fmt"
	"hash/fnv"
	"math"
	"time"
)

func Sleep(t time.Duration) {
	timer := time.NewTimer(t)
	defer timer.Stop()
	<-timer.C
}

func Noun(number int, one, two, five string) string {
	n := int(math.Abs(float64(number)))
	n %= 100
	if n >= 5 && n <= 20 {
		return five
	}
	n %= 10
	if n == 1 {
		return one
	}
	if n >= 2 && n <= 4 {
		return two
	}
	return five
}

func GenerateCodeHash() (int64, error) {
	h := fnv.New32a()
	bytes, err := time.Now().MarshalBinary()
	if err != nil {
		return 0, fmt.Errorf("hash binary encode error: %v", err)
	}

	_, err = h.Write(bytes)
	if err != nil {
		return 0, fmt.Errorf("hash write error: %w", err)
	}

	return int64(h.Sum32() >> 20), nil
}
