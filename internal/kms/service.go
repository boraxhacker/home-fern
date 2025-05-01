package kms

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"home-fern/internal/core"
	"strings"
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

	stringToEncrypt := string(request.Plaintext) + s.createContextSuffix(request.EncryptionContext)

	encstr, err := key.EncryptString(stringToEncrypt)
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

	decstr, err := key.DecryptString(string(request.CiphertextBlob))
	if err != nil {
		return nil, ErrKMSInternalException
	}

	suffix := s.createContextSuffix(request.EncryptionContext)
	if !strings.HasSuffix(decstr, suffix) {
		return nil, ErrInvalidCiphertextException
	}

	result := awskms.DecryptOutput{
		EncryptionAlgorithm: types.EncryptionAlgorithmSpecSymmetricDefault,
		KeyId:               request.KeyId,
		Plaintext:           []byte(strings.TrimSuffix(decstr, suffix)),
	}

	return &result, core.ErrNone
}

func (s *Service) createContextSuffix(ctx map[string]string) string {

	result := ""
	for ctxkey, ctxvalue := range ctx {

		result = result + ":::" + ctxkey + "=" + ctxvalue
	}

	return result
}
