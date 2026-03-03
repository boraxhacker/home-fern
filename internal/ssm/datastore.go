package ssm

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"home-fern/internal/core"
	"home-fern/internal/datastore"
	"home-fern/internal/kms"
	"io"
	"regexp"

	"go.etcd.io/bbolt"
)

type dataStore struct {
	ds   *datastore.Datastore
	keys []kms.KmsKey
}

func newDataStore(ds *datastore.Datastore, keys []kms.KmsKey) *dataStore {
	return &dataStore{ds: ds, keys: keys}
}

func (ds *dataStore) delete(key string) error {
	err := ds.ds.DeleteKeys(datastore.Ssm, []string{key})
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

func (ds *dataStore) deleteAll() error {
	err := ds.ds.DeleteBucket(datastore.Ssm)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}
	return nil
}

func (ds *dataStore) logKeys(w io.Writer) error {
	return ds.ds.LogKeys(datastore.Ssm, w)
}

func (ds *dataStore) findParametersByKey(
	filters []string, maxResults int, nextToken string) ([]ParameterData, string, error) {

	var result []ParameterData
	count := 0
	nextTokenResp := ""

	err := ds.ds.View(datastore.Ssm, func(b *bbolt.Bucket) error {
		c := b.Cursor()

		startKey := []byte{}
		if nextToken != "" {
			decodedToken, derr := base64.StdEncoding.DecodeString(nextToken)
			if derr != nil {
				return fmt.Errorf("invalid next token: %w", derr)
			}
			startKey = decodedToken
		}

		for k, v := c.Seek(startKey); k != nil; k, v = c.Next() {
			key := string(k)
			for _, filter := range filters {
				match, _ := regexp.MatchString(filter, key)
				if match {
					if count == maxResults {
						nextTokenResp = key
						return nil // Stop iteration
					}

					var param ParameterData
					if err := json.Unmarshal(v, &param); err != nil {
						return fmt.Errorf("failed to unmarshal parameter %s: %w", key, err)
					}

					result = append(result, param)
					count++
					break // Found a match for this key, move to next key
				}
			}
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, datastore.ErrBucketNotFound) {
			return []ParameterData{}, "", nil
		}
		return nil, "", fmt.Errorf("failed to find parameters: %w", err)
	}

	nextToken64 := ""
	if nextTokenResp != "" {
		nextToken64 = base64.StdEncoding.EncodeToString([]byte(nextTokenResp))
	}

	return result, nextToken64, nil
}

func (ds *dataStore) getParameter(key string) (*ParameterData, error) {
	var param ParameterData

	err := ds.ds.View(datastore.Ssm, func(b *bbolt.Bucket) error {
		v := b.Get([]byte(key))
		if v == nil {
			return core.ErrNotFound
		}
		return json.Unmarshal(v, &param)
	})

	if err != nil {
		if errors.Is(err, datastore.ErrBucketNotFound) {
			return nil, core.ErrNotFound
		}
		if errors.Is(err, core.ErrNotFound) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get parameter %s: %w", key, err)
	}

	return &param, nil
}

func (ds *dataStore) putParameter(
	key string, value *ParameterData, overwrite bool, skipTagCopy bool) (int64, error) {

	var newVersion int64 = 1

	err := ds.ds.Update(datastore.Ssm, func(b *bbolt.Bucket) error {
		existingBytes := b.Get([]byte(key))

		if existingBytes != nil {
			var existingParam ParameterData
			if err := json.Unmarshal(existingBytes, &existingParam); err != nil {
				return fmt.Errorf("failed to unmarshal existing parameter %s: %w", key, err)
			}

			if !overwrite {
				return core.ErrAlreadyExists
			}

			if !skipTagCopy {
				value.Tags = existingParam.Tags
			}

			newVersion = existingParam.Version + 1
		}

		value.Version = newVersion
		paramBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal parameter: %w", err)
		}

		return b.Put([]byte(key), paramBytes)
	})

	if err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return -1, core.ErrAlreadyExists
		}
		return -1, fmt.Errorf("failed to put parameter %s: %w", key, err)
	}

	return newVersion, nil
}

func (ds *dataStore) encrypt(stringToEncrypt string, keyId string) (string, error) {
	key, err := kms.FindKeyId(ds.keys, keyId)
	if err != nil {
		return "", err
	}

	result, err := key.EncryptString(stringToEncrypt, nil)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	return result, nil
}

func (ds *dataStore) decrypt(encryptedString string, keyId string) (string, error) {
	key, err := kms.FindKeyId(ds.keys, keyId)
	if err != nil {
		return "", err
	}

	result, err := key.DecryptString(encryptedString, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return result, nil
}
