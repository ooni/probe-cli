package tracex

import (
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSaver(t *testing.T) {
	t.Run("concurrent writes followed by read", func(t *testing.T) {
		saver := Saver{}
		var wg sync.WaitGroup
		const parallel = 10
		wg.Add(parallel)
		for idx := 0; idx < parallel; idx++ {
			go func() {
				saver.Write(&EventReadFromOperation{&EventValue{}})
				wg.Done()
			}()
		}
		wg.Wait()
		ev := saver.Read()
		if len(ev) != parallel {
			t.Fatal("unexpected number of events read")
		}
	})

	t.Run("NewConnectObserver", func(t *testing.T) {
		t.Run("nil Saver", func(t *testing.T) {
			var saver *Saver
			obs := saver.NewConnectObserver()
			if obs != nil {
				t.Fatal("expected nil observer")
			}
		})

		t.Run("nonnnil Saver", func(t *testing.T) {
			saver := &Saver{}
			obs := saver.NewConnectObserver()
			underlying := obs.(*dialerConnectObserver)
			if underlying.saver != saver {
				t.Fatal("invalid saver")
			}
		})
	})

	t.Run("NewReadWriteObserver", func(t *testing.T) {
		t.Run("nil Saver", func(t *testing.T) {
			var saver *Saver
			obs := saver.NewReadWriteObserver()
			if obs != nil {
				t.Fatal("expected nil observer")
			}
		})

		t.Run("nonnnil Saver", func(t *testing.T) {
			saver := &Saver{}
			obs := saver.NewReadWriteObserver()
			underlying := obs.(*dialerReadWriteObserver)
			if underlying.saver != saver {
				t.Fatal("invalid saver")
			}
		})
	})

	t.Run("WrapQUICDialer", func(t *testing.T) {
		t.Run("nil Saver", func(t *testing.T) {
			var saver *Saver
			base := &mocks.QUICDialer{}
			qd := saver.WrapQUICDialer(base)
			if qd != base {
				t.Fatal("unexpected returned QUICDialer")
			}
		})

		t.Run("nonnnil Saver", func(t *testing.T) {
			saver := &Saver{}
			base := &mocks.QUICDialer{}
			qd := saver.WrapQUICDialer(base)
			underlying := qd.(*QUICDialerSaver)
			if underlying.Saver != saver {
				t.Fatal("invalid Saver")
			}
			if underlying.QUICDialer != base {
				t.Fatal("invalid QUICDialer")
			}
		})
	})
}
