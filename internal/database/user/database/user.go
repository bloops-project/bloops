package database

import (
	"encoding/json"
	"fmt"
	"github.com/bloops-games/bloops/internal/byteutil"
	"github.com/bloops-games/bloops/internal/cache"
	"github.com/bloops-games/bloops/internal/database"
	"github.com/bloops-games/bloops/internal/database/user/model"
	bolt "go.etcd.io/bbolt"
)

var NotFoundErr = fmt.Errorf("not found")

const bucket = "users"

func New(db *database.DB, cache cache.Cache) *DB {
	return &DB{sDB: db, cache: cache}
}

type DB struct {
	sDB *database.DB

	cache cache.Cache
}

type fetchFn func(key int64) ([]byte, error)

func (db *DB) cachedValue(key int64, fn fetchFn) (model.User, error) {
	if db.cache != nil {
		v, ok := db.cache.Get(key)
		if ok {
			return v.(model.User), nil
		}
	}

	var u model.User
	bytes, err := fn(key)
	if err != nil {
		return u, fmt.Errorf("fetch: %w", err)
	}

	if len(bytes) == 0 {
		return u, NotFoundErr
	}

	if err := json.Unmarshal(bytes, &u); err != nil {
		return u, fmt.Errorf("unmarshal: %v", err)
	}

	if db.cache != nil {
		db.cache.Add(key, u)
	}

	return u, nil
}

func (db *DB) FetchByUsername(username string) (model.User, error) {
	var user model.User
	if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return NotFoundErr
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var u model.User
			if err := json.Unmarshal(v, &u); err != nil {
				return fmt.Errorf("json unmarshal error, %q", err)
			}
			if u.Username == username {
				user = u

				return nil
			}
		}
		if user.Id == 0 {
			return NotFoundErr
		}
		return nil
	}); err != nil {
		return user, fmt.Errorf("view transaction error: %w", err)
	}
	return user, nil
}

func (db *DB) Fetch(userId int64) (model.User, error) {
	var u model.User
	pk := byteutil.EncodeInt64ToBytes(userId)
	u, err := db.cachedValue(userId, func(key int64) ([]byte, error) {
		var bytes []byte

		if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucket))
			if b == nil {
				return NotFoundErr
			}
			bytes = b.Get(pk)
			return nil
		}); err != nil {
			return nil, fmt.Errorf("view transaction error: %w", err)
		}

		return bytes, nil
	})

	if err != nil {
		return u, fmt.Errorf("cached value: %w", err)
	}

	return u, nil
}

func (db *DB) Store(m model.User) error {
	var b *bolt.Bucket
	bytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	pk := byteutil.EncodeInt64ToBytes(m.Id)
	if err := db.sDB.DB.Update(func(tx *bolt.Tx) error {
		b = tx.Bucket([]byte(bucket))
		if b == nil {
			bs, err := tx.CreateBucket([]byte(bucket))
			if err != nil {
				return fmt.Errorf("create bucket: %w", err)
			}

			b = bs
		}

		if err := b.Put(pk, bytes); err != nil {
			return fmt.Errorf("put to bucket error: %w", err)
		}

		if db.cache != nil {
			db.cache.Add(m.Id, m)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}

	return nil
}
