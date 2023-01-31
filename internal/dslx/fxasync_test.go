package dslx

import (
	"context"
	"sync"
	"testing"
)

func increaseByOne(wg *sync.WaitGroup) Func[int, *Maybe[int]] {
	return &increase{wg}
}

type increase struct {
	wg *sync.WaitGroup // set to n corresponding to the number of used goroutines
}

func (f *increase) Apply(ctx context.Context, i int) *Maybe[int] {
	f.wg.Done()
	f.wg.Wait() // we want to make sure that this function has been reached n times before we continue
	return &Maybe[int]{State: i + 1}
}

func TestMap(t *testing.T) {
	inputs := []int{0, 10, 20, 30}
	wg := sync.WaitGroup{}
	wg.Add(len(inputs))
	inputStream := StreamList(inputs...)

	res := make(map[int]bool)
	// we need 4 goroutines to decrease the waitgroup counter in Apply to 0
	for out := range Map(context.Background(), 4, increaseByOne(&wg), inputStream) {
		res[out.State] = true
	}
	if !(res[1] && res[11] && res[21] && res[31]) {
		t.Fatalf("TestMap: expected results 1,11,21,31, got %v", res)
	}
}

func TestMapNegativeParallelism(t *testing.T) {
	inputs := []int{0}
	wg := sync.WaitGroup{}
	wg.Add(len(inputs))
	inputStream := StreamList(inputs...)

	res := make(map[int]bool)
	// we expect parallelism to be set to 1 if it is < 0
	for out := range Map(context.Background(), -1, increaseByOne(&wg), inputStream) {
		res[out.State] = true
	}
	if !res[1] {
		t.Fatalf("TestMapNegativeParallelism: expected results 1, got %v", res)
	}
}

func TestApplyAsync(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	res := make(map[int]bool)
	for out := range ApplyAsync(context.Background(), increaseByOne(&wg), 0) {
		res[out.State] = true
	}
	if !res[1] {
		t.Fatalf("TestApplyAsync: expected results 1, got %v", res)
	}
}

func TestParallel(t *testing.T) {
	input := 2
	wg := sync.WaitGroup{}
	wg.Add(input)

	funcs := []Func[int, *Maybe[int]]{
		increaseByOne(&wg),
		increaseByOne(&wg),
	}
	res := []int{}
	for _, out := range Parallel(context.Background(), Parallelism(input), 0, funcs...) {
		res = append(res, out.State)
		if out.State != 1 {
			t.Fatalf("TestParallel: unexpected result, want 1, got %d", out.State)
		}
	}
	if len(res) != 2 {
		t.Fatalf("TestParallel: expected 3 results, got %d", len(res))
	}
}

func TestParallelNegativeParallelism(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	funcs := []Func[int, *Maybe[int]]{
		increaseByOne(&wg),
	}
	res := []int{}
	for _, out := range Parallel(context.Background(), -1, 0, funcs...) {
		res = append(res, out.State)
		if out.State != 1 {
			t.Fatalf("TestParallelNegativeParallelism: unexpected result, want 1, got %d", out.State)
		}
	}
	if len(res) != 1 {
		t.Fatalf("TestParallelNegativeParallelism: expected 3 results, got %d", len(res))
	}
}
