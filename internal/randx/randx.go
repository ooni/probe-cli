// Package randx contains math/rand extensions. The functions
// exported by this package do not use a CSRNG so you SHOULD NOT
// use these strings for, e.g., generating passwords.
package randx

import (
	"math/rand"
	"time"
	"unicode"
)

// These constants are used by lettersWithString.
const (
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	letters   = uppercase + lowercase
)

// lettersWithString is a method for generating a random string
// described at https://stackoverflow.com/questions/22892120.
func lettersWithString(n int, letterBytes string) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rnd.Intn(len(letterBytes))]
	}
	return string(b)
}

// Letters return a string composed of random letters. Note that
// this function uses a non-cryptographically-secure generator.
func Letters(n int) string {
	return lettersWithString(n, letters)
}

// LettersUppercase return a string composed of random uppercase
// letters. Note that this function uses a non-cryptographically-secure
// generator. So, we SHOULD NOT use it for generating passwords.
func LettersUppercase(n int) string {
	return lettersWithString(n, uppercase)
}

// ChangeCapitalization returns a new string where the capitalization
// of each character is changed at random. Note that this function
// uses a non-cryptographically-secure generator. So, we SHOULD NOT use
// it for generating passwords.
func ChangeCapitalization(source string) (dest string) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, chr := range source {
		if unicode.IsLower(chr) && rnd.Float64() <= 0.5 {
			dest += string(unicode.ToUpper(chr))
		} else if unicode.IsUpper(chr) && rnd.Float64() <= 0.5 {
			dest += string(unicode.ToLower(chr))
		} else {
			dest += string(chr)
		}
	}
	return
}
