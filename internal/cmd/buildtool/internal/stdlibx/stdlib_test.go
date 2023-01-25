package stdlibx

import "testing"

func TestNewStdlib(t *testing.T) {
	lib := NewStdlib()
	realStdlib, ok := lib.(*stdlib)
	if !ok {
		t.Fatal("not an instance of *stdlib")
	}
	_, ok = realStdlib.exiter.(*realExiter)
	if !ok {
		t.Fatal("not an instance of *realExiter")
	}
}
