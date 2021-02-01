package oonimkall

import "github.com/google/uuid"

// NewUUID4 generates a new UUID4 string. This functionality is typically
// used by mobile apps to generate random unique identifiers.
func NewUUID4() string {
	return uuid.Must(uuid.NewRandom()).String()
}
