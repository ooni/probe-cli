package dslx

import (
	"testing"
)

func TestCollect(t *testing.T) {
	ch := make(chan int)
	resCh := make(chan []int)
	defer close(resCh)
	go func() {
		resCh <- Collect(ch)
	}()

	ch <- 1
	ch <- 1
	close(ch)

	res := <-resCh
	if len(res) != 2 {
		t.Fatalf("TestCollect: Unexpected number of collected items, want 2, got %d", len(res))
	}
	for _, i := range res {
		if i != 1 {
			t.Fatalf("TestCollect: Unexpected value of collected item, want 1, got %d", i)
		}
	}
}

func TestStreamList(t *testing.T) {
	input := []int{0, 1, 2, 3}
	i := 0
	for o := range StreamList(input...) { // this will block if the returned channel is not closed
		if o != input[i] {
			t.Fatalf("TestStreamList: unexpected item in stream, expected %d, got %d", input[i], o)
		}
		i += 1
	}
}

func TestZip(t *testing.T) {
	srcCh1 := make(chan int)
	srcCh2 := make(chan int)
	resCh := make(chan []int)
	defer close(resCh)

	go func() {
		resCh <- ZipAndCollect(srcCh1, srcCh2)
	}()

	srcCh1 <- 1
	close(srcCh1)
	srcCh2 <- 2
	close(srcCh2)

	out := <-resCh
	if len(out) != 2 {
		t.Fatalf("TestZip: unexpected number of items from channel, expected 2, got %d", len(out))
	}
}
