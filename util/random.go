package util

import (
	"crypto/rand"
	"errors"
)

// RandBytes generates a buffer of random bytes of the specified length
func RandBytes(length int) ([]byte, error) {
	buff := make([]byte, length)
	n, err := rand.Read(buff)
	if err != nil {
		return nil, err
	}
	if n != length {
		return nil, errors.New("randomness buffer was not filled")
	}
	return buff, nil
}
