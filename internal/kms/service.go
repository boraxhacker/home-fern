package kms

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type Service struct {
	accountId string
	region    string
	keys      []KmsKey
}

func NewService(keys []KmsKey, region string, accountId string) *Service {

	result := Service{
		region:    region,
		accountId: accountId,
		keys:      keys,
	}

	return &result
}

func (s *Service) Encrypt(request *awskms.EncryptInput) (*awskms.EncryptOutput, error) {

	key, err := FindKeyId(s.keys, aws.ToString(request.KeyId))
	if err != nil {
		return nil, err
	}

	var aad []byte

	if len(request.EncryptionContext) > 0 {

		aad, err = json.Marshal(request.EncryptionContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal encryption context: %w", ErrKMSInternalException)
		}
	}

	encstr, err := key.EncryptString(string(request.Plaintext), aad)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", ErrKMSInternalException)
	}

	result := awskms.EncryptOutput{
		KeyId:               request.KeyId,
		CiphertextBlob:      []byte(encstr),
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
	}

	return &result, nil
}

func (s *Service) Decrypt(request *awskms.DecryptInput) (*awskms.DecryptOutput, error) {

	key, err := FindKeyId(s.keys, aws.ToString(request.KeyId))
	if err != nil {
		return nil, err
	}

	var aad []byte

	if request.EncryptionContext != nil && len(request.EncryptionContext) > 0 {

		aad, err = json.Marshal(request.EncryptionContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal encryption context: %w", ErrKMSInternalException)
		}
	}

	decstr, err := key.DecryptString(string(request.CiphertextBlob), aad)
	if err != nil {
		return nil, ErrInvalidCiphertextException
	}

	result := awskms.DecryptOutput{
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
		KeyId:               request.KeyId,
		Plaintext:           []byte(decstr),
	}

	return &result, nil
}
