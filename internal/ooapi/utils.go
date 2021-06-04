package ooapi

import "fmt"

func newErrEmptyField(field string) error {
	return fmt.Errorf("%w: %s", ErrEmptyField, field)
}

func newHTTPFailure(status int) error {
	return fmt.Errorf("%w: %d", ErrHTTPFailure, status)
}

func newQueryFieldInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}

func newQueryFieldBool(v bool) string {
	return fmt.Sprintf("%v", v)
}

func newAuthorizationHeader(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}
