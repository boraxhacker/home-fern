package datastore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.etcd.io/bbolt"
)

type BucketName string

const (
	Ssm     BucketName = "Ssm"
	Route53 BucketName = "Route53"
)

var (
	ErrKeyExists      = errors.New("key already exists")
	ErrBucketNotFound = errors.New("bucket not found")
)

type Datastore struct {
	db *bbolt.DB
}

type PutData struct {
	Key       string
	Data      interface{}
	Delete    bool
	Overwrite bool
}

func New(path string) (*Datastore, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt db: %w", err)
	}

	return &Datastore{db: db}, nil
}

func (ds *Datastore) Close() error {
	return ds.db.Close()
}

func (ds *Datastore) View(bucket BucketName, fn func(b *bbolt.Bucket) error) error {
	return ds.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return fn(b)
	})
}

func (ds *Datastore) Update(bucket BucketName, fn func(b *bbolt.Bucket) error) error {
	return ds.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return fn(b)
	})
}

func (ds *Datastore) DeleteBucket(bucket BucketName) error {
	return ds.db.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if errors.Is(err, bbolt.ErrBucketNotFound) {
			return nil
		}
		return err
	})
}

func (ds *Datastore) DeleteKeys(bucket BucketName, keys []string) error {
	return ds.Update(bucket, func(b *bbolt.Bucket) error {
		for _, key := range keys {
			if err := b.Delete([]byte(key)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (ds *Datastore) LogKeys(bucket BucketName, w io.Writer) error {
	return ds.View(bucket, func(b *bbolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			if _, err := w.Write(k); err != nil {
				return err
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				return err
			}
			return nil
		})
	})
}

func (ds *Datastore) PutKeys(bucket BucketName, data []PutData) error {
	return ds.Update(bucket, func(b *bbolt.Bucket) error {
		for _, item := range data {
			if item.Delete {
				if err := b.Delete([]byte(item.Key)); err != nil {
					return err
				}
			} else {
				if !item.Overwrite {
					if b.Get([]byte(item.Key)) != nil {
						return ErrKeyExists
					}
				}

				dataBytes, err := json.Marshal(item.Data)
				if err != nil {
					return err
				}

				if err := b.Put([]byte(item.Key), dataBytes); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (ds *Datastore) GetPrefixCount(bucket BucketName, prefix string) (int, error) {
	count := 0
	err := ds.View(bucket, func(b *bbolt.Bucket) error {
		c := b.Cursor()
		prefixBytes := []byte(prefix)
		for k, _ := c.Seek(prefixBytes); k != nil && bytes.HasPrefix(k, prefixBytes); k, _ = c.Next() {
			count++
		}
		return nil
	})

	if err != nil && errors.Is(err, ErrBucketNotFound) {
		return 0, nil
	}

	return count, err
}
