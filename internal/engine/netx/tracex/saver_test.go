package tracex

import (
	"sync"
	"testing"
)

func TestSaver(t *testing.T) {
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
}
