package database

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloops-games/bloops/internal/byteutil"
	"github.com/bloops-games/bloops/internal/database"
	"github.com/bloops-games/bloops/internal/database/matchstate/model"
	bolt "go.etcd.io/bbolt"
)

const prefix = "states"

var (
	ErrEntryNotFound  = fmt.Errorf("not found")
	ErrBucketNotFound = fmt.Errorf("bucket not found")
)

func New(db *database.DB) *DB {
	return &DB{sDB: db}
}

type DB struct {
	sDB *database.DB
}

func (db *DB) FetchAll() ([]model.State, error) {
	var list []model.State

	if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(prefix))
		if b == nil {
			return ErrEntryNotFound
		}

		if err := b.ForEach(func(k, v []byte) error {
			var metric model.State
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

	return list, nil
}

func (db *DB) Clean() error {
	tx, err := db.sDB.DB.Begin(true)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	defer tx.Rollback() // nolint

	if err := tx.DeleteBucket([]byte(prefix)); err != nil {
		if errors.Is(err, bolt.ErrBucketNotFound) {
			return ErrBucketNotFound
		}
		return fmt.Errorf("delete bucket: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func (db *DB) Add(m model.State) error {
	tx, err := db.sDB.DB.Begin(true)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	defer tx.Rollback() // nolint

	b := tx.Bucket([]byte(prefix))
	if b == nil {
		bs, err := tx.CreateBucket([]byte(prefix))
		if err != nil {
			return fmt.Errorf("can not create bucket: %w", err)
		}
		b = bs
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := b.Put(byteutil.EncodeInt64ToBytes(m.Code), bytes); err != nil {
		return fmt.Errorf("put to bucket error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
