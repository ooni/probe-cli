package webconnectivitylte

import "testing"

func TestBitmask(t *testing.T) {
	bitmask := &Bitmask{}

	bitmask.Set(55)
	t.Logf("%v", bitmask.V)

	bitmask.Set(17)
	t.Logf("%v", bitmask.V)

	bitmask.Set(4)
	t.Logf("%v", bitmask.V)

	v1 := bitmask.Get(55)
	t.Logf("%v", v1)

	v2 := bitmask.Get(50)
	t.Logf("%v", v2)

	v3 := bitmask.Get(17)
	t.Logf("%v", v3)

	v4 := bitmask.Get(14)
	t.Logf("%v", v4)

	v5 := bitmask.Get(4)
	t.Logf("%v", v5)

	v6 := bitmask.Get(0)
	t.Logf("%v", v6)

	bitmask.Clear(55)
	bitmask.Clear(4)
	t.Logf("%v", bitmask.V)

	v7 := bitmask.Get(55)
	t.Logf("%v", v7)

	v8 := bitmask.Get(17)
	t.Logf("%v", v8)

	v9 := bitmask.Get(4)
	t.Logf("%v", v9)
}
