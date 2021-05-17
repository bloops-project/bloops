package database

import (
	"encoding/json"
	"fmt"
	"github.com/bloops-games/bloops/internal/byteutil"
	"github.com/bloops-games/bloops/internal/cache"
	"github.com/bloops-games/bloops/internal/database"
	"github.com/bloops-games/bloops/internal/database/stat/model"
	bolt "go.etcd.io/bbolt"
	"time"
)

const prefix = "stat"

var pLen = len(prefix)
var NotFoundErr = fmt.Errorf("not found")

func New(db *database.DB, cache cache.Cache) *DB {
	return &DB{sDB: db, cache: cache}
}

type DB struct {
	sDB *database.DB

	cache cache.Cache
}

func (db *DB) BytesBucket(userId int64) []byte {
	b := make([]byte, pLen+2<<5) // prefix + uint64
	copy(b, prefix[:])
	copy(b[pLen:], byteutil.EncodeInt64ToBytes(userId))
	return b
}

func (db *DB) SerialBucket(userID int64) string {
	return fmt.Sprintf("%s%d", prefix, userID)
}

func (db *DB) FetchRateStat(userId int64) (model.RateStat, error) {
	var rates model.RateStat
	stats, err := db.FetchByUserId(userId)
	if err != nil {
		return rates, fmt.Errorf("fetch by userId: %w", err)
	}
	var bloops []string
	for _, stat := range stats {
		if stat.Conclusion == model.StatusFavorite {
			rates.Stars++
		}
	BloopLoop:
		for _, bloop := range stat.Bloops {
			for _, bloop1 := range bloops {
				if bloop == bloop1 {
					continue BloopLoop
				}
			}
			bloops = append(bloops, bloop)
		}
	}

	rates.Bloops = len(bloops)
	return rates, nil
}

func (db *DB) FetchProfileStat(userId int64) (model.AggregationStat, error) {
	var aggregationStat model.AggregationStat
	var sumPoints, pointsNum int
	var sumDuration time.Duration

	stats, err := db.FetchByUserId(userId)
	if err != nil {
		return aggregationStat, fmt.Errorf("fetch by userId: %w", err)
	}
	for _, stat := range stats {
		if stat.BestPoints > aggregationStat.BestPoints {
			aggregationStat.BestPoints = stat.BestPoints
		}

		if aggregationStat.WorstPoints == 0 {
			aggregationStat.WorstPoints = stat.WorstPoints
		} else if stat.WorstPoints < aggregationStat.WorstPoints {
			aggregationStat.WorstPoints = stat.WorstPoints
		}

		if aggregationStat.BestDuration == 0 {
			aggregationStat.BestDuration = stat.BestDuration
		} else if stat.BestDuration < aggregationStat.BestDuration {
			aggregationStat.BestDuration = stat.BestDuration
		}

		if stat.WorstDuration > aggregationStat.WorstDuration {
			aggregationStat.WorstDuration = stat.WorstDuration
		}

		sumDuration += stat.SumDuration
		sumPoints += stat.SumPoints
		pointsNum += 1
		if stat.Conclusion == model.StatusFavorite {
			aggregationStat.Stars++
		}

	BloopLoop:
		for _, bloop := range stat.Bloops {
			for _, bloop1 := range aggregationStat.Bloops {
				if bloop == bloop1 {
					continue BloopLoop
				}
			}
			aggregationStat.Bloops = append(aggregationStat.Bloops, bloop)
		}
		aggregationStat.Count++
	}

	if pointsNum > 0 {
		aggregationStat.AvgPoints = sumPoints / pointsNum
	}

	if pointsNum > 0 {
		aggregationStat.AvgDuration = time.Duration(sumDuration.Nanoseconds() / int64(pointsNum))
	}

	return aggregationStat, nil
}

func (db *DB) FetchByUserId(userId int64) ([]model.Stat, error) {
	var list []model.Stat
	bBucket := db.BytesBucket(userId)
	sBucket := db.SerialBucket(userId)
	if db.cache != nil {
		v, ok := db.cache.Get(sBucket)
		if ok {
			return v.([]model.Stat), nil
		}
	}

	if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bBucket)
		if b == nil {
			return NotFoundErr
		}

		if err := b.ForEach(func(k, v []byte) error {
			var metric model.Stat
			if err := json.Unmarshal(v, &metric); err != nil {
				return fmt.Errorf("json unmarshal error, %q", err)
			}
			list = append(list, metric)
			return nil
		}); err != nil {
			return fmt.Errorf("bucket for each: %v", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("view transaction error: %w", err)
	}

	if db.cache != nil {
		db.cache.Add(sBucket, list)
	}

	return list, nil
}

func (db *DB) Add(m model.Stat) error {
	tx, err := db.sDB.DB.Begin(true)
	if err != nil {
		return fmt.Errorf("starting transaction: %v", err)
	}

	defer tx.Rollback()

	bBucket := db.BytesBucket(m.UserId)
	sBucket := db.SerialBucket(m.UserId)

	b := tx.Bucket(bBucket)
	if b == nil {
		bs, err := tx.CreateBucket(db.BytesBucket(m.UserId))
		if err != nil {
			return fmt.Errorf("can not create bucket %d: %w", m.UserId, err)
		}
		b = bs
	}

	binaryId, err := m.Id.MarshalBinary()
	if err != nil {
		return fmt.Errorf("uuid binary: %v", err)
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}

	if err := b.Put(binaryId, bytes); err != nil {
		return fmt.Errorf("put to bucket error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %v", err)
	}

	if db.cache != nil {
		db.cache.Delete(sBucket)
	}

	return nil
}
