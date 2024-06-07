package testingx

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// exampleStructure is an example structure we fill.
type exampleStructure struct {
	CategoryCodes string
	CountryCode   string
	Enabled       bool
	MaxResults    int64
	Now           time.Time
}

func TestFakeFillWorksWithCustomTime(t *testing.T) {
	var req *exampleStructure
	ff := &FakeFiller{
		Now: func() time.Time {
			return time.Date(1992, time.January, 24, 17, 53, 0, 0, time.UTC)
		},
	}
	ff.Fill(&req)
	if req == nil {
		t.Fatal("we expected non nil here")
	}
	t.Log(req)
}

func TestFakeFillAllocatesIntoAPointerToPointer(t *testing.T) {
	var req *exampleStructure
	ff := &FakeFiller{}
	ff.Fill(&req)
	if req == nil {
		t.Fatal("we expected non nil here")
	}
	t.Log(req)
}

func TestFakeFillAllocatesIntoAMapLikeWithStringKeys(t *testing.T) {
	var resp map[string]*exampleStructure
	ff := &FakeFiller{}
	ff.Fill(&resp)
	if resp == nil {
		t.Fatal("we expected non nil here")
	}
	if len(resp) < 1 {
		t.Fatal("we expected some data here")
	}
	t.Log(resp)
	for _, value := range resp {
		if value == nil {
			t.Fatal("expected non-nil here")
		}
	}
}

func TestFakeFillPanicsWithMapsWithNonStringKeys(t *testing.T) {
	var panicmsg string
	func() {
		defer func() {
			if v := recover(); v != nil {
				panicmsg = v.(string)
			}
		}()
		var resp map[int64]*exampleStructure
		ff := &FakeFiller{}
		ff.Fill(&resp)
		if resp != nil {
			t.Fatal("we expected nil here")
		}
	}()
	if panicmsg != "fakefill: we only support string key types" {
		t.Fatal("unexpected panic message", panicmsg)
	}
}

func TestFakeFillAllocatesIntoASlice(t *testing.T) {
	var resp *[]*exampleStructure
	ff := &FakeFiller{}
	ff.Fill(&resp)
	if resp == nil {
		t.Fatal("we expected non nil here")
	}
	if len(*resp) < 1 {
		t.Fatal("we expected some data here")
	}
	t.Log(resp)
	for _, entry := range *resp {
		if entry == nil {
			t.Fatal("expected non-nil here")
		}
	}
}

func TestFakeFillSkipsPrivateTypes(t *testing.T) {
	t.Run("with private struct fields", func(t *testing.T) {
		// define structure with mixed private and public fields
		type employee struct {
			ID   int64
			age  int64
			name string
		}

		// create empty employee
		var person employee

		// fake-fill the employee
		ff := &FakeFiller{}
		ff.Fill(&person)

		// define what we expect to see
		expect := employee{
			ID:   person.ID,
			age:  0,
			name: "",
		}

		// make sure we've got what we expected
		//
		// Note: we cannot use cmp.Diff directly because it cannot
		// access private fields, so we need to write manual comparison
		if person != expect {
			t.Fatal("expected", expect, "got", person)
		}
	})

	t.Run("make sure we cannot initialize a non-addressable type", func(t *testing.T) {
		// create a zero struct
		shouldRemainZero := exampleStructure{}

		// attempt to fake fill w/o taking the address
		ff := &FakeFiller{}
		ff.Fill(shouldRemainZero)

		// make sure it's still zero
		if diff := cmp.Diff(exampleStructure{}, shouldRemainZero); diff != "" {
			t.Fatal(diff)
		}
	})
}
