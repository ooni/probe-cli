package dslx

import (
	"testing"
)

func TestCollect(t *testing.T) {
	t.Run("Collect results from channel", func(t *testing.T) {
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
			t.Fatalf("unexpected number of collected items")
		}
		for _, i := range res {
			if i != 1 {
				t.Fatalf("unexpected value of collected item")
			}
		}
	})
}

func TestZip(t *testing.T) {
	t.Run("Merge results from 2 channels into one channel and collect", func(t *testing.T) {
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
			t.Fatalf("unexpected number of items from channel")
		}
	})
}

func TestStreamList(t *testing.T) {
	t.Run("Create a channel from list of numbers with StreamList", func(t *testing.T) {
		input := []int{0, 1, 2, 3}
		i := 0
		for o := range StreamList(input...) { // this will block if the returned channel is not closed
			if o != input[i] {
				t.Fatalf("unexpected item in stream")
			}
			i += 1
		}
	})
}
