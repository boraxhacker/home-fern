package route53

import (
	"encoding/json"
	"errors"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/dgraph-io/badger/v4"
	"home-fern/internal/core"
	"log"
	"math"
	"regexp"
	"strings"
	"time"
)

const (
	// /hostedzone/{id}
	// /recordset/{id}/{name}

	HostedZonePrefix = "/hostedzone/"
	ChangeInfoPrefix = "/change/"
	RecordSetPrefix  = "/recordset/"
)

type dataStore struct {
	db *badger.DB
}

type listOptions struct {
	hostedZone  *HostedZoneData
	startRecord string
	startType   awstypes.RRType
	count       int
}

type recordKey struct {
	rrname string
	rrkey  string
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

func (ds *dataStore) deleteHostedZone(id string, ci *ChangeInfoData) core.ErrorCode {

	err := ds.db.Update(func(txn *badger.Txn) error {

		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = []byte(RecordSetPrefix + strings.TrimPrefix(HostedZonePrefix, id) + "/")

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			rerr := txn.Delete(it.Item().Key())
			if rerr != nil {
				return rerr
			}
		}

		if !strings.HasPrefix(id, HostedZonePrefix) {
			id = HostedZonePrefix + id
		}

		derr := txn.Delete([]byte(id))
		if derr != nil {
			return derr
		}

		bytes, jerr := json.Marshal(ci)
		if jerr != nil {
			return jerr
		}

		entry := badger.NewEntry([]byte(ChangeInfoPrefix+ci.Id), bytes).WithTTL(time.Hour * 24 * 30)

		serr := txn.SetEntry(entry)
		if serr != nil {
			return serr
		}

		return nil
	})

	if err != nil {

		return translateBadgerError(err)
	}

	return core.ErrNone
}

func (ds *dataStore) findHostedZones(nameFilter string) ([]HostedZoneData, core.ErrorCode) {

	var result []HostedZoneData

	err := ds.db.View(func(txn *badger.Txn) error {

		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Prefix = []byte(HostedZonePrefix)

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

			match, _ := regexp.MatchString(nameFilter, hz.Name)
			if match {
				result = append(result, hz)
			}
		}

		return nil
	})

	if err != nil {

		return nil, translateBadgerError(err)
	}

	return result, core.ErrNone
}

func (ds *dataStore) getChange(id string) (*ChangeInfoData, core.ErrorCode) {

	var result ChangeInfoData

	if !strings.HasPrefix(id, ChangeInfoPrefix) {
		id = ChangeInfoPrefix + id
	}

	err := ds.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {

			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &result)
		})
	})

	if err != nil {

		return nil, translateBadgerError(err)
	}

	return &result, core.ErrNone
}

func (ds *dataStore) getHostedZone(id string) (*HostedZoneData, core.ErrorCode) {

	var result HostedZoneData

	if !strings.HasPrefix(id, HostedZonePrefix) {
		id = HostedZonePrefix + id
	}

	err := ds.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &result)
		})
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrNoSuchHostedZone
		}

		return nil, translateBadgerError(err)
	}

	return &result, core.ErrNone
}

func (ds *dataStore) getHostedZoneCount() (int, core.ErrorCode) {

	result, err := core.GetPrefixCount(ds.db, HostedZonePrefix)
	if err != nil {

		return -1, translateBadgerError(err)
	}

	return result, core.ErrNone

}

func (ds *dataStore) getRecordCount(id string) (int, core.ErrorCode) {

	result, err := core.GetPrefixCount(ds.db, RecordSetPrefix+strings.TrimPrefix(HostedZonePrefix, id)+"/")
	if err != nil {

		return -1, translateBadgerError(err)
	}

	return int(math.Max(2, float64(result))), core.ErrNone
}

func (ds *dataStore) getResourceRecordSets(options *listOptions) (*ListRecordSetsOutput, core.ErrorCode) {

	result := ListRecordSetsOutput{
		Records:    make([]ResourceRecordSetData, 0),
		NextRecord: "",
		NexType:    "",
	}

	start := ""
	count := 0

	prefix := RecordSetPrefix + strings.TrimPrefix(HostedZonePrefix, options.hostedZone.Id) + "/"
	if options.startRecord != "" && options.startType != "" {

		header, err := convertToKey(options.hostedZone.Name, options.startRecord, options.startType)
		if err != core.ErrNone {
			return nil, err
		}

		start = strings.TrimSuffix(prefix, "/") + header.rrkey
	}

	err := ds.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {

			item := it.Item()

			add := start == ""
			if !add && string(item.Key()) == start {
				add = true
				start = ""
			}

			if add {
				var rr ResourceRecordSetData

				verr := item.Value(func(val []byte) error {

					return json.Unmarshal(val, &rr)
				})

				if verr != nil {
					return verr
				}

				if options.count > 0 && count == options.count {
					result.NextRecord = rr.Name
					result.NexType = rr.Type
					break
				}

				result.Records = append(result.Records, rr)
				count = count + 1
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrNoSuchHostedZone
		}

		return nil, translateBadgerError(err)
	}

	return &result, core.ErrNone

}

func (ds *dataStore) putHostedZone(
	hz *HostedZoneData, changes []ChangeData, ci *ChangeInfoData) core.ErrorCode {

	curzones, cerr := ds.findHostedZones(strings.Replace(hz.Name, ".", "\\.", -1))
	if cerr != core.ErrNone || len(curzones) > 0 {
		if len(curzones) > 0 {
			return ErrHostedZoneAlreadyExists
		}

		return cerr
	}

	data := []core.PutData{{
		Key:       hz.Id,
		Data:      hz,
		Overwrite: false,
	}, {
		Key:       ci.Id,
		Data:      ci,
		Overwrite: false,
		TTL:       time.Until(time.Now().Add(90 * 24 * time.Hour)),
	}}

	pddata, pderr := convertToPutData(hz, changes)
	if pderr != core.ErrNone {
		return pderr
	}

	data = append(data, pddata...)

	err := core.PutKeys(ds.db, data)
	if err != nil {
		return translateBadgerError(err)
	}

	return core.ErrNone
}

func (ds *dataStore) putRecordSets(hz *HostedZoneData, changes []ChangeData, ci *ChangeInfoData) core.ErrorCode {

	data, pderr := convertToPutData(hz, changes)
	if pderr != core.ErrNone {
		return pderr
	}

	data = append(data, core.PutData{
		Key:       ci.Id,
		Data:      ci,
		Overwrite: false,
		TTL:       time.Until(time.Now().Add(90 * 24 * time.Hour)),
	})

	err := core.PutKeys(ds.db, data)
	if err != nil {
		if errors.Is(err, badger.ErrRejected) {
			return ErrInvalidInput
		}
		return translateBadgerError(err)
	}

	return core.ErrNone
}

func (ds *dataStore) updateHostedZone(hz *HostedZoneData) core.ErrorCode {

	data := []core.PutData{{
		Key:       hz.Id,
		Data:      hz,
		Overwrite: true,
	}}

	err := core.PutKeys(ds.db, data)
	if err != nil {
		return translateBadgerError(err)
	}

	return core.ErrNone
}

func translateBadgerError(err error) core.ErrorCode {

	if errors.Is(err, badger.ErrRejected) {
		return ErrHostedZoneAlreadyExists
	} else if errors.Is(err, badger.ErrKeyNotFound) {
		return core.ErrNotFound
	}

	log.Println("An error occurred.", err)
	return core.ErrInternalError
}

func convertToPutData(hz *HostedZoneData, changes []ChangeData) ([]core.PutData, core.ErrorCode) {

	hzid := strings.TrimPrefix(HostedZonePrefix, hz.Id)

	result := make([]core.PutData, 0)

	for _, change := range changes {

		header, err := convertToKey(hz.Name, change.ResourceRecordSet.Name, change.ResourceRecordSet.Type)
		if err != core.ErrNone {

			return nil, err
		}

		change.ResourceRecordSet.Name = header.rrname

		result = append(result, core.PutData{
			Key:       RecordSetPrefix + hzid + header.rrkey,
			Data:      change.ResourceRecordSet,
			Delete:    change.Action == awstypes.ChangeActionDelete,
			Overwrite: change.Action == awstypes.ChangeActionUpsert,
		})
	}

	return result, core.ErrNone
}

func convertToKey(domain string, rrname string, rrtype awstypes.RRType) (*recordKey, core.ErrorCode) {

	lwrname := strings.ToLower(rrname)
	if !strings.HasSuffix(lwrname, ".") {
		lwrname = lwrname + "."
	}

	if lwrname != domain && !strings.HasSuffix(lwrname, "."+domain) {
		return nil, ErrInvalidChangeBatch
	}

	rrkey := strings.Replace(lwrname, domain, "", -1)
	if rrkey == "" {
		rrkey = "@"
	}

	result := recordKey{
		rrname: lwrname,
		rrkey:  "/" + strings.TrimSuffix(lwrname, ".") + "/" + strings.ToLower(string(rrtype)),
	}

	return &result, core.ErrNone
}
