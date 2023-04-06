package miniengine

//
// The "close" task
//

import "github.com/ooni/probe-cli/v3/internal/optional"

// TODO(bassosimone): we should refactor this code to return a Task[Void], which
// allows us to print logs while closing the session.

// Close closes the [Session]. This function is safe to call multiple
// times. We'll close underlying resources on the first invocation and
// otherwise do nothing for subsequent invocations.
func (s *Session) Close() (err error) {
	s.closeJustOnce.Do(func() {
		// make sure the cleanup is synchronized.
		defer s.mu.Unlock()
		s.mu.Lock()

		// handle the case where there is no state.
		if s.state.IsNone() {
			return
		}

		// obtain the underlying state
		state := s.state.Unwrap()

		// replace with empty state
		s.state = optional.None[*engineSessionState]()

		// close the underlying session
		err = state.sess.Close()
	})
	return err
}
