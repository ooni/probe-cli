package netxlite

//
// I/O extensions
//

import (
	"context"
	"errors"
	"io"
)

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

// StreamAllContext streams from the given reader [r] until
// interrupted by [ctx] or when [r] hits the EOF.
//
// This function runs a background goroutine that should exit as soon
// as [ctx] is done or when [reader] is closed, if applicable.
//
// This function transforms an errors.Is(err, io.EOF) to a nil error
// such as the standard library's ReadAll does.
//
// The caller of this function MUST check for the context being
// expired when there's an error and decide what it is proper to
// do in such a case. Typically, when streaming the HTTP response
// body for Web Connectivity v0.5, the right thing to do is to
// report the body as being truncated.
//
// This function might return a non-zero-length buffer along with
// an non-nil error in the case in which we could only read a portion
// of the body and then we were interrupted by the error.
func StreamAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	data, err := streamAllContext(ctx, reader)
	if err != nil {
		err = NewTopLevelGenericErrWrapper(err)
	}
	return data, err
}

// streamAllContext is the inner function invoked by [StreamAllContext].
func streamAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	// create channels for communicating with inner goroutine.
	datach, errch := make(chan []byte), make(chan error)

	// ensure we're able to notify the inner goroutine about cancellation.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for {
			// create buffer holding the next chunk of data.
			//
			// Implementation note: the buffer MUST be created at each
			// loop, otherwise we're data-racing with the reader.
			buffer := make([]byte, 1<<13)

			// issue the read proper
			count, err := reader.Read(buffer)

			// handle the case where we have some data
			//
			// Note: it's legit to read some data _and_ receive an error
			if count > 0 {
				data := buffer[:count]
				select {
				case datach <- data:
					// fallthrough to check error
				case <-ctx.Done():
					return
				}
			}

			// handle the case of error
			//
			// Note: it's legit to read some data _and_ receive an error
			if err != nil {
				select {
				case errch <- err:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// preallocate the result buffer to avoid frequent reallocations
	resultbuf := make([]byte, 0, 1<<17)

	for {
		select {
		// case #1 - we received some data from the inner reader
		case data := <-datach:
			resultbuf = append(resultbuf, data...)

		// case #2 - error occurred, remember to deal w/ EOF
		case err := <-errch:
			if errors.Is(err, io.EOF) {
				// see https://github.com/ooni/probe/issues/1965
				return resultbuf, nil
			}
			return resultbuf, err

		// case #3 - the context is done, goodbye!
		case <-ctx.Done():
			// Historical implementation note: the original StreamAllContext
			// implemented inside Web Connectivity v0.5 transformed the
			// context.DeadlineExceeded error to nil. However, this is wrong
			// because it prevents distinguishing real EOF and the case in
			// which we timed out receiving from the socket.
			return resultbuf, ctx.Err()
		}
	}
}
