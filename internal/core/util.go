package core

import (
	"math/rand"
	"strings"
	"time"
)

func StringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// generateRandomString generates a random string of a specified length using alphanumeric characters,
// ensuring that the string does not start with a number.
func GenerateRandomString(length int) string {
	// Define the set of characters to choose from.
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Seed the random number generator.
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create a byte slice to store the random characters.
	b := make([]byte, length)

	// Ensure the first character is an alphabet.
	b[0] = charset[seededRand.Intn(len(charset)-10)]

	// Fill the rest of the byte slice with random characters from the charset.
	for i := 1; i < length; i++ {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	// Convert the byte slice to a string and return it.
	return string(b)
}

func FindKeyId(keys []KmsKey, keyId string) (*KmsKey, ErrorCode) {

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

			return &key, ErrNone
		}
	}

	return nil, ErrInvalidKeyId
}
