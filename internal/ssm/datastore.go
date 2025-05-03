package ssm

import (
	"encoding/json"
	"errors"
	"github.com/dgraph-io/badger/v4"
	"home-fern/internal/core"
	"log"
	"regexp"
)

type dataStore struct {
	db   *badger.DB
	keys []core.KmsKey
}

func newDataStore(databasePath string, keys []core.KmsKey) *dataStore {

	opts := badger.DefaultOptions(databasePath).WithLoggingLevel(badger.ERROR)
	db, err := badger.Open(opts)
	if err != nil {
		log.Panicln("Error opening badger db:", err)
	}

	return &dataStore{db: db, keys: keys}
}

func (ds *dataStore) Close() {

	if ds.db != nil {
		err := ds.db.Close()
		if err != nil {

			log.Println("Failed to close database.", err)
		}
	}
}

func (ds *dataStore) delete(key string) core.ErrorCode {

	keys := []string{key}
	err := core.DeleteKeys(ds.db, keys)

	if err != nil {

		return translateBadgerError(err)
	}

	return core.ErrNone
}

func (ds *dataStore) findParametersByKey(filters []string) ([]ParameterData, core.ErrorCode) {

	var result []ParameterData

	err := ds.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			for _, filter := range filters {

				match, _ := regexp.MatchString(filter, key)
				if match {

					var param ParameterData
					umerr := item.Value(func(val []byte) error {
						return json.Unmarshal(val, &param)
					})

					if umerr == nil {

						result = append(result, param)

					} else {

						return umerr
					}

					break
				}
			}
		}

		return nil
	})

	if err != nil {

		return nil, translateBadgerError(err)
	}

	return result, core.ErrNone
}

func (ds *dataStore) getParameter(key string) (*ParameterData, core.ErrorCode) {

	var param ParameterData

	err := ds.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {

			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &param)
		})
	})

	if err != nil {

		return nil, translateBadgerError(err)
	}

	return &param, core.ErrNone
}

func (ds *dataStore) putParameter(
	key string, value *ParameterData, overwrite bool, skipTagCopy bool) (int64, core.ErrorCode) {

	var newVersion int64 = 1
	var existingParam ParameterData

	err := ds.db.Update(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(key))

		if err == nil {

			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &existingParam)
			}); err != nil {
				return err
			}

			if !overwrite {
				return badger.ErrRejected
			}

			if !skipTagCopy {
				// we need to copy the old tags, update version
				value.Tags = existingParam.Tags
				newVersion = existingParam.Version + 1
			}

		} else if !errors.Is(err, badger.ErrKeyNotFound) {

			return err
		}

		value.Version = newVersion
		paramBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		return txn.Set([]byte(key), paramBytes)
	})

	if err != nil {

		return -1, translateBadgerError(err)
	}

	return newVersion, core.ErrNone
}

func (ds *dataStore) encrypt(stringToEncrypt string, keyId string) (string, core.ErrorCode) {

	key, ec := core.FindKeyId(ds.keys, keyId)
	if ec != core.ErrNone {
		return "", ec
	}

	result, err := key.EncryptString(stringToEncrypt, nil)
	if err != nil {
		return "", translateBadgerError(err)
	}

	return result, core.ErrNone
}

func (ds *dataStore) decrypt(encryptedString string, keyId string) (string, core.ErrorCode) {

	key, ec := core.FindKeyId(ds.keys, keyId)
	if ec != core.ErrNone {
		return "", ec
	}

	result, err := key.DecryptString(encryptedString, nil)
	if err != nil {
		return "", translateBadgerError(err)
	}

	return result, core.ErrNone
}

func translateBadgerError(err error) core.ErrorCode {

	if errors.Is(err, badger.ErrKeyNotFound) {
		return ErrParameterNotFound
	} else if errors.Is(err, badger.ErrRejected) {
		return ErrParameterAlreadyExists
	}

	log.Println("An error occurred.", err)
	return core.ErrInternalError
}
