package tunnel

import "context"

// MockableSession is a mockable session.
type MockableSession struct {
	// Result contains the bytes of the psiphon config.
	Result []byte

	// Err is the error, if any.
	Err error
}

// FetchPsiphonConfig implements ExperimentSession.FetchPsiphonConfig
func (sess *MockableSession) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return sess.Result, sess.Err
}
