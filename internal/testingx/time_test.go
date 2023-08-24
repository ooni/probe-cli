package testingx

import (
	"testing"
	"time"
)

func TestNewTimeDeterministic(t *testing.T) {
	zero := time.Date(2023, 8, 24, 11, 45, 00, 0, time.UTC)
	td := NewTimeDeterministic(zero)
	if !td.zeroTime.Equal(zero) {
		t.Fatal("unexpected zero time")
	}
	if td.counter != 0 {
		t.Fatal("unexpected counter")
	}
}

func TestTimeDeterministic(t *testing.T) {
	td := &TimeDeterministic{}
	t0 := td.Now()
	if !t0.Equal(td.zeroTime) {
		t.Fatal("invalid t0 value")
	}
	t1 := td.Now()
	if t1.Sub(t0) != time.Second {
		t.Fatal("invalid t1 value")
	}
	t2 := td.Now()
	if t2.Sub(t1) != time.Second {
		t.Fatal("invalid t2 value")
	}
}
