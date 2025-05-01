package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

type FernCredentials struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Username  string `yaml:"username"`
}

type FernConfig struct {
	Region      string            `yaml:"region"`
	Credentials []FernCredentials `yaml:"credentials"`
	Keys        []KmsKey          `yaml:"kms"`
	DnsDefaults DnsDefaults       `yaml:"dns"`
}

type DnsDefaults struct {
	Soa         string   `yaml:"soa"`
	NameServers []string `yaml:"nameServers"`
}

type KmsKey struct {
	KeyId string `yaml:"id"`
	Alias string `yaml:"alias"`
	Key   string `yaml:"key"`
}

func (key *KmsKey) EncryptString(stringToEncrypt string) (string, error) {

	// Since the key is in string format, convert it to bytes
	bytes, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(bytes)
	if err != nil {
		return "", err
	}

	// Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	// https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce. Nonce should never be reused with the same key.
	// Since we use GCM, we recommend using 12 bytes.
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data using aesGCM.Seal. Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(stringToEncrypt), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (key *KmsKey) DecryptString(encryptedString string) (string, error) {

	enc, err := base64.StdEncoding.DecodeString(encryptedString)
	if err != nil {
		return "", err
	}

	// Since the key is in string format, convert it to bytes
	bytes, err := base64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(bytes)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()

	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

type ResourceTag struct {
	Key   string
	Value string
}

type DatabaseDumper interface {
	LogKeys(writer io.Writer) error
}

const (
	ZeroAccountId string = "000000000000"
)
