package netxlite

//
// I/O extensions
//

import (
	"context"
	"errors"
	"io"
)

// TODO(bassosimone): consider integrating StreamAllContext from
// internal/experiment/webconnectivity/iox.go

// ReadAllContext is like io.ReadAll but reads r in a
// background goroutine. This function will return
// earlier if the context is cancelled. In which case
// we will continue reading from the reader in the background
// goroutine, and we will discard the result. To stop
// the long-running goroutine, close the connection
// bound to the reader. Until such a connection is closed,
// you're leaking the backround goroutine and doing I/O.
//
// As of Go 1.17.6, ReadAllContext additionally deals
// with wrapped io.EOF correctly, while io.ReadAll does
// not. See https://github.com/ooni/probe/issues/1965.
func ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error) {
	datach, errch := make(chan []byte, 1), make(chan error, 1) // buffers
	go func() {
		data, err := io.ReadAll(r)
		if errors.Is(err, io.EOF) {
			// See https://github.com/ooni/probe/issues/1965
			err = nil
		}
		if err != nil {
			errch <- err
			return
		}
		datach <- data
	}()
	select {
	case data := <-datach:
		return data, nil
	case <-ctx.Done():
		return nil, NewTopLevelGenericErrWrapper(ctx.Err())
	case err := <-errch:
		return nil, NewTopLevelGenericErrWrapper(err)
	}
}

// CopyContext is like io.Copy but may terminate earlier
// when the context expires. This function has the same
// caveats of ReadAllContext regarding the temporary leaking
// of the background I/O goroutine.
//
// As of Go 1.17.6, CopyContext additionally deals
// with wrapped io.EOF correctly, while io.Copy does
// not. See https://github.com/ooni/probe/issues/1965.
func CopyContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	countch, errch := make(chan int64, 1), make(chan error, 1) // buffers
	go func() {
		count, err := io.Copy(dst, src)
		if errors.Is(err, io.EOF) {
			// See https://github.com/ooni/probe/issues/1965
			err = nil
		}
		if err != nil {
			errch <- err
			return
		}
		countch <- count
	}()
	select {
	case count := <-countch:
		return count, nil
	case <-ctx.Done():
		return 0, NewTopLevelGenericErrWrapper(ctx.Err())
	case err := <-errch:
		return 0, NewTopLevelGenericErrWrapper(err)
	}
}
