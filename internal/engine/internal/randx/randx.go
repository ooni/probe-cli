// Package randx contains math/rand extensions
package randx

import (
	"math/rand"
	"time"
	"unicode"
)

const (
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	letters   = uppercase + lowercase
)

func lettersWithString(n int, letterBytes string) string {
	// See https://stackoverflow.com/questions/22892120
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rnd.Intn(len(letterBytes))]
	}
	return string(b)
}

// Letters return a string composed of random letters
func Letters(n int) string {
	return lettersWithString(n, letters)
}

// LettersUppercase return a string composed of random uppercase letters
func LettersUppercase(n int) string {
	return lettersWithString(n, uppercase)
}

// ChangeCapitalization returns a new string where the capitalization
// of each character is changed at random.
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
