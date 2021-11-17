package randx

import "testing"

func TestLetters(t *testing.T) {
	str := Letters(1024)
	for _, chr := range str {
		if (chr >= 'A' && chr <= 'Z') || (chr >= 'a' && chr <= 'z') {
			continue
		}
		t.Fatal("invalid input char")
	}
}

func TestLettersUppercase(t *testing.T) {
	str := LettersUppercase(1024)
	for _, chr := range str {
		if chr >= 'A' && chr <= 'Z' {
			continue
		}
		t.Fatal("invalid input char")
	}
}

func TestChangeCapitalization(t *testing.T) {
	str := Letters(2048)
	if ChangeCapitalization(str) == str {
		t.Fatal("capitalization not changed")
	}
}
