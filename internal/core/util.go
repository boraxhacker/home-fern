package core

import (
	"math/rand"
	"time"
)

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
