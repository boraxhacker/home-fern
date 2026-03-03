package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type KmsKey struct {
	KeyId string `yaml:"id"`
	Alias string `yaml:"alias"`
	Key   string `yaml:"key"`
}

func FindKeyId(keys []KmsKey, keyId string) (*KmsKey, error) {

	chk := keyId
	if strings.HasPrefix(keyId, "arn:aws:kms:") {

		// (obviously) ignores region and account id
		pieces := strings.Split(keyId, ":")
		if len(pieces) != 6 {
			return nil, ErrInvalidKeyId
		}

		chk = strings.TrimPrefix(pieces[5], "key/")
	}

	for _, key := range keys {

		if "alias/"+key.Alias == chk || key.KeyId == chk {

			return &key, nil
		}
	}

	return nil, ErrInvalidKeyId
}

func (key *KmsKey) EncryptString(stringToEncrypt string, aad []byte) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return "", fmt.Errorf("invalid base64 key: %w", err)
	}

	block, err := aes.NewCipher(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(stringToEncrypt), aad)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (key *KmsKey) DecryptString(encryptedString string, aad []byte) (string, error) {
	enc, err := base64.StdEncoding.DecodeString(encryptedString)
	if err != nil {
		return "", fmt.Errorf("invalid base64 ciphertext: %w", err)
	}

	bytes, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return "", fmt.Errorf("invalid base64 key: %w", err)
	}

	block, err := aes.NewCipher(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(enc) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}
