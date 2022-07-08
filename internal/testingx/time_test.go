package testingx

import (
	"testing"
	"time"
)

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
