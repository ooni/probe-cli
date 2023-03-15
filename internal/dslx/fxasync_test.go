package dslx

import (
	"context"
	"sync"
	"testing"
)

func getFnWait(wg *sync.WaitGroup) Func[int, *Maybe[int]] {
	return &fnWait{wg}
}

type fnWait struct {
	wg *sync.WaitGroup // set to n corresponding to the number of used goroutines
}

func (f *fnWait) Apply(ctx context.Context, i int) *Maybe[int] {
	f.wg.Done()
	f.wg.Wait() // continue when n goroutines have reached this point
	return &Maybe[int]{State: i + 1}
}

/*
Test cases:
- Map multiple inputs to multiple goroutines:
  - with 4 goroutines
  - expect parallelism set to 1 if < 0
*/
func TestMap(t *testing.T) {
	t.Run("Map multiple inputs to multiple goroutines", func(t *testing.T) {
		tests := map[string]struct {
			input       []int
			parallelism int
		}{
			"with 4 goroutines":                   {input: []int{0, 10, 20, 30}, parallelism: 4},
			"expect parallelism set to 1 if <= 0": {input: []int{0}, parallelism: 0},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				wg := sync.WaitGroup{}
				wg.Add(len(tt.input))
				inputStream := StreamList(tt.input...)

				res := make(map[int]bool)
				// we need tt.parallelism goroutines to decrease the waitgroup counter to 0
				for out := range Map(context.Background(), Parallelism(tt.parallelism), getFnWait(&wg), inputStream) {
					res[out.State] = true
				}
				for _, i := range tt.input {
					if !res[i+1] {
						t.Fatalf("unexpected result")
					}
				}
			})
		}
	})
}

func TestApplyAsync(t *testing.T) {
	t.Run("ApplyAsync: ApplyAsync returns a channel", func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(1)
		out := <-ApplyAsync(context.Background(), getFnWait(&wg), 0)
		if out.State != 1 {
			t.Fatalf("unexpected result")
		}
	})
}

/*
Test cases:
- Parallel: Map multiple funcs working on the same input to multiple goroutines:
  - with 2 goroutines and 2 processing funcs
  - expect parallelism set to 1 if < 0
*/
func TestParallel(t *testing.T) {
	t.Run("Parallel: Map multiple funcs working on the same input to multiple goroutines", func(t *testing.T) {
		tests := map[string]struct {
			funcs       int
			parallelism int
		}{
			"with 2 goroutines and 2 funcs":       {funcs: 2, parallelism: 2},
			"expect parallelism set to 1 if <= 0": {funcs: 1, parallelism: 0},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				wg := sync.WaitGroup{}
				wg.Add(tt.funcs)
				funcs := []Func[int, *Maybe[int]]{}
				for i := 0; i < tt.funcs; i++ {
					funcs = append(funcs, getFnWait(&wg))
				}
				out := Parallel(context.Background(), Parallelism(tt.parallelism), 0, funcs...)
				if len(out) != tt.funcs {
					t.Fatalf("unexpected number of results")
				}
			})
		}
	})
}
