package core

import (
	"encoding/json"
	"errors"
	"github.com/dgraph-io/badger/v4"
	"time"
)

type PutData struct {
	Key       string
	Data      interface{}
	Overwrite bool
	TTL       time.Duration
}

func DeleteKeys(db *badger.DB, keys []string) error {

	err := db.Update(
		func(txn *badger.Txn) error {

			for _, key := range keys {
				err := txn.Delete([]byte(key))
				if err != nil {
					return err
				}
			}

			return nil
		})

	if err != nil {
		return err
	}

	return nil
}

func GetPrefixCount(db *badger.DB, prefix string) (int, error) {

	result := 0
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			result += 1
		}

		return nil
	})

	if err != nil {

		return -1, err
	}

	return result, nil
}

func PutKeys(db *badger.DB, data []PutData) error {

	err := db.Update(func(txn *badger.Txn) error {

		for _, item := range data {

			var currerr error
			_, err := txn.Get([]byte(item.Key))

			if err == nil {
				if !item.Overwrite {
					currerr = badger.ErrRejected
				}
			} else if !errors.Is(err, badger.ErrKeyNotFound) {

				currerr = err
			}

			if currerr != nil {
				return currerr
			}

			bytes, jerr := json.Marshal(item.Data)
			if jerr != nil {
				return jerr
			}

			entry := badger.NewEntry([]byte(item.Key), bytes)
			if item.TTL > time.Nanosecond {
				entry = entry.WithTTL(item.TTL)
			}

			serr := txn.SetEntry(entry)
			if serr != nil {
				return serr
			}
		}

		return nil
	})

	if err != nil {

		return err
	}

	return nil
}
