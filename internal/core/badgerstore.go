package core

import (
	"encoding/json"
	"github.com/dgraph-io/badger/v4"
)

type PutData struct {
	Key  string
	Data interface{}
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

func PutKeys(db *badger.DB, data []PutData) error {

	err := db.Update(func(txn *badger.Txn) error {

		for _, item := range data {

			bytes, jerr := json.Marshal(item.Data)
			if jerr != nil {
				return jerr
			}

			return txn.Set([]byte(item.Key), bytes)
		}

		return nil
	})

	if err != nil {

		return err
	}

	return nil
}
