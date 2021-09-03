// Copyright Red Hat

package helpers

import (
	"crypto/rand"
)

type RandomType string

const (
	RandomTypeNumber   RandomType = "number"
	RandomTypeAlpha    RandomType = "alpha"
	RandomTypeAlphaNum RandomType = "alphanum"
	RandomTypePassword RandomType = "password"
)

func RandomString(strSize int, randType RandomType) string {

	var dictionary string

	switch randType {
	case RandomTypePassword:
		dictionary = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_?!&@"
	case RandomTypeAlphaNum:
		dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	case RandomTypeAlpha:
		dictionary = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	case RandomTypeNumber:
		dictionary = "0123456789"
	}

	var bytes = make([]byte, strSize)
	_, err := rand.Read(bytes)
	if err == nil {
		for k, v := range bytes {
			bytes[k] = dictionary[v%byte(len(dictionary))]
		}
	}
	return string(bytes)
}
