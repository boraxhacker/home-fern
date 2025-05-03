package kms

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"home-fern/internal/core"
)

type Service struct {
	accountId string
	region    string
	keys      []core.KmsKey
}

func NewService(fernConfig *core.FernConfig, accountId string) *Service {

	result := Service{
		region:    fernConfig.Region,
		accountId: accountId,
		keys:      fernConfig.Keys,
	}

	return &result
}

func (s *Service) Encrypt(request *awskms.EncryptInput) (*awskms.EncryptOutput, core.ErrorCode) {

	key, ec := core.FindKeyId(s.keys, aws.ToString(request.KeyId))
	if ec != core.ErrNone {
		return nil, ec
	}

	var aad []byte
	var err error

	if len(request.EncryptionContext) > 0 {

		aad, err = json.Marshal(request.EncryptionContext)
		if err != nil {
			return nil, ErrKMSInternalException
		}
	}

	encstr, err := key.EncryptString(string(request.Plaintext), aad)
	if err != nil {
		return nil, ErrKMSInternalException
	}

	result := awskms.EncryptOutput{
		KeyId:               request.KeyId,
		CiphertextBlob:      []byte(encstr),
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
	}

	return &result, core.ErrNone
}

func (s *Service) Decrypt(request *awskms.DecryptInput) (*awskms.DecryptOutput, core.ErrorCode) {

	key, ec := core.FindKeyId(s.keys, aws.ToString(request.KeyId))
	if ec != core.ErrNone {
		return nil, ec
	}

	var aad []byte
	var err error

	if request.EncryptionContext != nil && len(request.EncryptionContext) > 0 {

		aad, err = json.Marshal(request.EncryptionContext)
		if err != nil {
			return nil, ErrKMSInternalException
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

	return &result, core.ErrNone
}
