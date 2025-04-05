package route53

import (
	"encoding/json"
	"github.com/dgraph-io/badger/v4"
	"home-fern/internal/core"
	"log"
)

type dataStore struct {
	db *badger.DB
}

func newDataStore(databasePath string) *dataStore {

	opts := badger.DefaultOptions(databasePath).WithLoggingLevel(badger.ERROR)
	db, err := badger.Open(opts)
	if err != nil {
		log.Panicln("Error opening badger db:", err)
	}

	return &dataStore{db: db}
}

func (ds *dataStore) Close() {

	if ds.db != nil {
		err := ds.db.Close()
		if err != nil {

			log.Println("Failed to close database.", err)
		}
	}
}

func (ds *dataStore) deleteKeys(keys []string) core.ErrorCode {

	err := core.DeleteKeys(ds.db, keys)

	if err != nil {

		return translateBadgerError(err)
	}

	return core.ErrNone
}

func (ds *dataStore) findHostedZones() ([]HostedZoneData, core.ErrorCode) {

	var result []HostedZoneData

	err := ds.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			var hz HostedZoneData

			verr := item.Value(func(val []byte) error {

				return json.Unmarshal(val, &hz)
			})

			if verr != nil {
				return verr
			}

			result = append(result, hz)
		}

		return nil
	})

	if err != nil {

		return nil, translateBadgerError(err)
	}

	return result, core.ErrNone
}

func (ds *dataStore) putHostedZone(hz *HostedZoneData, cd *ChangeInfoData) core.ErrorCode {

	data := []core.PutData{{
		Key:  "/hostedzone/" + hz.Id,
		Data: hz,
	}, {
		Key:  hz.Id + "/change/" + cd.Id,
		Data: cd,
	}}

	err := core.PutKeys(ds.db, data)
	if err != nil {
		return translateBadgerError(err)
	}

	return core.ErrNone
}

func translateBadgerError(err error) core.ErrorCode {

	log.Println("An error occurred.", err)
	return core.ErrInternalError
}
