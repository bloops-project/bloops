package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bloops-games/bloops/internal/byteutil"
	"github.com/bloops-games/bloops/internal/cache"
	"github.com/bloops-games/bloops/internal/database"
	"github.com/bloops-games/bloops/internal/database/stat/model"
	bolt "go.etcd.io/bbolt"
)

const prefix = "stat"

var (
	pLen        = len(prefix)
	ErrNotFound = fmt.Errorf("not found")
)

func New(db *database.DB, cache cache.Cache) *DB {
	return &DB{sDB: db, cache: cache}
}

type DB struct {
	sDB *database.DB

	cache cache.Cache
}

func (db *DB) BytesBucket(userID int64) []byte {
	b := make([]byte, pLen+2<<5) // prefix + uint64
	copy(b, prefix[:])
	copy(b[pLen:], byteutil.EncodeInt64ToBytes(userID))
	return b
}

func (db *DB) SerialBucket(userID int64) string {
	return fmt.Sprintf("%s%d", prefix, userID)
}

func (db *DB) FetchRateStat(userID int64) (model.RateStat, error) {
	var rates model.RateStat
	stats, err := db.FetchByuserID(userID)
	if err != nil {
		return rates, fmt.Errorf("fetch by userID: %w", err)
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

func (db *DB) FetchProfileStat(userID int64) (model.AggregationStat, error) {
	var aggregationStat model.AggregationStat
	var sumPoints, pointsNum int
	var sumDuration time.Duration

	stats, err := db.FetchByuserID(userID)
	if err != nil {
		return aggregationStat, fmt.Errorf("fetch by userID: %w", err)
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

func (db *DB) FetchByuserID(userID int64) ([]model.Stat, error) {
	var list []model.Stat
	bBucket := db.BytesBucket(userID)
	sBucket := db.SerialBucket(userID)
	if db.cache != nil {
		v, ok := db.cache.Get(sBucket)
		if ok {
			return v.([]model.Stat), nil
		}
	}

	if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bBucket)
		if b == nil {
			return ErrNotFound
		}

		if err := b.ForEach(func(k, v []byte) error {
			var metric model.Stat
			if err := json.Unmarshal(v, &metric); err != nil {
				return fmt.Errorf("json unmarshal error, %w", err)
			}
			list = append(list, metric)
			return nil
		}); err != nil {
			return fmt.Errorf("bucket for each: %w", err)
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
		return fmt.Errorf("starting transaction: %w", err)
	}

	defer tx.Rollback() //nolint

	bBucket := db.BytesBucket(m.UserID)
	sBucket := db.SerialBucket(m.UserID)

	b := tx.Bucket(bBucket)
	if b == nil {
		bs, err := tx.CreateBucket(db.BytesBucket(m.UserID))
		if err != nil {
			return fmt.Errorf("can not create bucket %d: %w", m.UserID, err)
		}
		b = bs
	}

	binaryID, err := m.ID.MarshalBinary()
	if err != nil {
		return fmt.Errorf("uuid binary: %w", err)
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := b.Put(binaryID, bytes); err != nil {
		return fmt.Errorf("put to bucket error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	if db.cache != nil {
		db.cache.Delete(sBucket)
	}

	return nil
}
