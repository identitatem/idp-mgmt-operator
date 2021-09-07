// Copyright Red Hat

package helpers

import (
	"regexp"
	"testing"
)

func TestRandomAlpha(t *testing.T) {
	size := 64
	regexString := "^[ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz]+$"
	random := RandomString(size, RandomTypeAlpha)
	RunRandomCheck(t, random, regexString, size)
}

func TestRandomAlphaNum(t *testing.T) {
	size := 128
	regexString := "^[0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz]+$"
	random := RandomString(size, RandomTypeAlphaNum)
	RunRandomCheck(t, random, regexString, size)
}

func TestRandomNumber(t *testing.T) {
	size := 48
	regexString := "^[0123456789]+$"
	random := RandomString(size, RandomTypeNumber)
	RunRandomCheck(t, random, regexString, size)
}

func TestRandomPassword(t *testing.T) {
	size := 32
	regexString := "^[abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_?!&@]+$"
	random := RandomString(size, RandomTypePassword)
	RunRandomCheck(t, random, regexString, size)
}

func RunRandomCheck(t *testing.T, random string, regex string, size int) {
	if len(random) != size {
		t.Fatalf(`Random string is not the correct size.  Expected %d, actual %d.  Random string is %s`, size, len(random), random)
	}
	isValid := regexp.MustCompile(regex).MatchString
	if !isValid(random) {
		t.Fatalf(`Random string contains invalid characters.  Expected %s, actual %s`, regex, random)
	}

}
