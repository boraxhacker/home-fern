package route53

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"home-fern/internal/core"
	"home-fern/internal/datastore"
	"io"
	"math"
	"regexp"
	"strings"

	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"go.etcd.io/bbolt"
)

const (
	HostedZonePrefix = "/hostedzone/"
	ChangeInfoPrefix = "/change/"
	RecordSetPrefix  = "/recordset/"
)

type dataStore struct {
	ds *datastore.Datastore
}

type recordKey struct {
	rrname string
	rrkey  string
}

func newDataStore(ds *datastore.Datastore) *dataStore {
	return &dataStore{ds: ds}
}

func (ds *dataStore) deleteAll() error {
	err := ds.ds.DeleteBucket(datastore.Route53)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}
	return nil
}

func (ds *dataStore) logKeys(w io.Writer) error {
	return ds.ds.LogKeys(datastore.Route53, w)
}

func (ds *dataStore) deleteHostedZone(id string, ci *ChangeInfoData) error {
	err := ds.ds.Update(datastore.Route53, func(b *bbolt.Bucket) error {
		prefix := []byte(RecordSetPrefix + strings.TrimPrefix(id, HostedZonePrefix) + "/")
		c := b.Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			if err := b.Delete(k); err != nil {
				return err
			}
		}

		if !strings.HasPrefix(id, HostedZonePrefix) {
			id = HostedZonePrefix + id
		}
		if err := b.Delete([]byte(id)); err != nil {
			return err
		}

		// ChangeInfo is now stored in the same bucket.
		// The original had a TTL, which we are ignoring for now.
		jsonbytes, jerr := json.Marshal(ci)
		if jerr != nil {
			return jerr
		}
		return b.Put([]byte(ci.Id), jsonbytes)
	})

	if err != nil {
		return fmt.Errorf("failed to delete hosted zone %s: %w", id, err)
	}
	return nil
}

func (ds *dataStore) findHostedZones(nameFilter *regexp.Regexp) ([]HostedZoneData, error) {
	var result []HostedZoneData

	err := ds.ds.View(datastore.Route53, func(b *bbolt.Bucket) error {
		c := b.Cursor()
		prefix := []byte(HostedZonePrefix)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var hz HostedZoneData
			if err := json.Unmarshal(v, &hz); err != nil {
				return fmt.Errorf("failed to unmarshal hosted zone: %w", err)
			}

			if nameFilter != nil {
				if nameFilter.MatchString(hz.Name) {
					result = append(result, hz)
				}
			} else {
				result = append(result, hz)
			}
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, datastore.ErrBucketNotFound) {
			return []HostedZoneData{}, nil
		}
		return nil, fmt.Errorf("failed to find hosted zones: %w", err)
	}
	return result, nil
}

func (ds *dataStore) getChange(id string) (*ChangeInfoData, error) {
	var result ChangeInfoData
	if !strings.HasPrefix(id, ChangeInfoPrefix) {
		id = ChangeInfoPrefix + id
	}

	err := ds.ds.View(datastore.Route53, func(b *bbolt.Bucket) error {
		v := b.Get([]byte(id))
		if v == nil {
			return core.ErrNotFound
		}
		return json.Unmarshal(v, &result)
	})

	if err != nil {
		if errors.Is(err, datastore.ErrBucketNotFound) {
			return nil, core.ErrNotFound
		}
		if errors.Is(err, core.ErrNotFound) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get change %s: %w", id, err)
	}
	return &result, nil
}

func (ds *dataStore) getHostedZone(id string) (*HostedZoneData, error) {
	var result HostedZoneData
	if !strings.HasPrefix(id, HostedZonePrefix) {
		id = HostedZonePrefix + id
	}

	err := ds.ds.View(datastore.Route53, func(b *bbolt.Bucket) error {
		v := b.Get([]byte(id))
		if v == nil {
			return ErrNoSuchHostedZone
		}
		return json.Unmarshal(v, &result)
	})

	if err != nil {
		if errors.Is(err, datastore.ErrBucketNotFound) {
			return nil, ErrNoSuchHostedZone
		}
		if errors.Is(err, ErrNoSuchHostedZone) {
			return nil, ErrNoSuchHostedZone
		}
		return nil, fmt.Errorf("failed to get hosted zone %s: %w", id, err)
	}
	return &result, nil
}

func (ds *dataStore) getHostedZoneCount() (int, error) {
	count, err := ds.ds.GetPrefixCount(datastore.Route53, HostedZonePrefix)
	if err != nil {
		return -1, fmt.Errorf("failed to get hosted zone count: %w", err)
	}
	return count, nil
}

func (ds *dataStore) getRecordCount(id string) (int, error) {
	prefix := RecordSetPrefix + strings.TrimPrefix(id, HostedZonePrefix) + "/"
	count, err := ds.ds.GetPrefixCount(datastore.Route53, prefix)
	if err != nil {
		return -1, fmt.Errorf("failed to get record count for %s: %w", id, err)
	}
	return int(math.Max(2, float64(count))), nil
}

func (ds *dataStore) getResourceRecordSets(hzId string) ([]ResourceRecordSetData, error) {
	var result []ResourceRecordSetData
	prefix := RecordSetPrefix + strings.TrimPrefix(hzId, HostedZonePrefix) + "/"

	err := ds.ds.View(datastore.Route53, func(b *bbolt.Bucket) error {
		c := b.Cursor()
		prefixBytes := []byte(prefix)
		for k, v := c.Seek(prefixBytes); k != nil && bytes.HasPrefix(k, prefixBytes); k, v = c.Next() {
			var rr ResourceRecordSetData
			if err := json.Unmarshal(v, &rr); err != nil {
				return fmt.Errorf("failed to unmarshal record set: %w", err)
			}
			result = append(result, rr)
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, datastore.ErrBucketNotFound) {
			return []ResourceRecordSetData{}, nil
		}
		return nil, fmt.Errorf("failed to get resource record sets for %s: %w", hzId, err)
	}
	return result, nil
}

func (ds *dataStore) putHostedZone(hz *HostedZoneData, changes []ChangeData, ci *ChangeInfoData) error {
	filter, rerr := regexp.Compile(strings.Replace(hz.Name, ".", "\\.", -1))
	if rerr != nil {
		return fmt.Errorf("failed to compile regex: %w", rerr)
	}

	curzones, cerr := ds.findHostedZones(filter)
	if cerr != nil {
		return cerr
	}
	if len(curzones) > 0 {
		return ErrHostedZoneAlreadyExists
	}

	data := []datastore.PutData{{
		Key:       hz.Id,
		Data:      hz,
		Overwrite: false,
	}, {
		Key:       ci.Id,
		Data:      ci,
		Overwrite: false,
	}}

	pddata, pderr := convertToPutData(hz, changes)
	if pderr != nil {
		return pderr
	}
	data = append(data, pddata...)

	err := ds.ds.PutKeys(datastore.Route53, data)
	if err != nil {
		return fmt.Errorf("failed to put hosted zone: %w", err)
	}
	return nil
}

func (ds *dataStore) putRecordSets(hz *HostedZoneData, changes []ChangeData, ci *ChangeInfoData) error {
	data, pderr := convertToPutData(hz, changes)
	if pderr != nil {
		return pderr
	}
	data = append(data, datastore.PutData{
		Key:       ci.Id,
		Data:      ci,
		Overwrite: false,
	})

	err := ds.ds.PutKeys(datastore.Route53, data)
	if err != nil {
		if errors.Is(err, datastore.ErrKeyExists) {
			return ErrInvalidInput
		}
		return fmt.Errorf("failed to put record sets: %w", err)
	}
	return nil
}

func (ds *dataStore) updateHostedZone(hz *HostedZoneData) error {
	data := []datastore.PutData{{
		Key:       hz.Id,
		Data:      hz,
		Overwrite: true,
	}}
	err := ds.ds.PutKeys(datastore.Route53, data)
	if err != nil {
		return fmt.Errorf("failed to update hosted zone: %w", err)
	}
	return nil
}

func convertToPutData(hz *HostedZoneData, changes []ChangeData) ([]datastore.PutData, error) {
	hzid := strings.TrimPrefix(hz.Id, HostedZonePrefix)
	result := make([]datastore.PutData, 0)

	for _, change := range changes {
		header, err := convertToKey(hz.Name, change.ResourceRecordSet.Name, change.ResourceRecordSet.Type)
		if err != nil {
			return nil, err
		}
		change.ResourceRecordSet.Name = header.rrname

		result = append(result, datastore.PutData{
			Key:       RecordSetPrefix + hzid + header.rrkey,
			Data:      change.ResourceRecordSet,
			Delete:    change.Action == awstypes.ChangeActionDelete,
			Overwrite: change.Action == awstypes.ChangeActionUpsert,
		})
	}
	return result, nil
}

func convertToKey(domainp string, rrname string, rrtype awstypes.RRType) (*recordKey, error) {
	lwrname := strings.ToLower(rrname)
	if !strings.HasSuffix(lwrname, ".") {
		lwrname = lwrname + "."
	}

	if lwrname != domainp && !strings.HasSuffix(lwrname, "."+domainp) {
		return nil, ErrInvalidChangeBatch
	}

	rrkey := strings.Replace(lwrname, domainp, "", -1)
	if rrkey == "" {
		rrkey = "@"
	}

	result := recordKey{
		rrname: lwrname,
		rrkey:  "/" + strings.TrimSuffix(rrkey, ".") + "/" + strings.ToLower(string(rrtype)),
	}
	return &result, nil
}
