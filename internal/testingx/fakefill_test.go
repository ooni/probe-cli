package testingx

import (
	"testing"
	"time"
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
}

func TestFakeFillAllocatesIntoAPointerToPointer(t *testing.T) {
	var req *exampleStructure
	ff := &FakeFiller{}
	ff.Fill(&req)
	if req == nil {
		t.Fatal("we expected non nil here")
	}
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
	for _, value := range resp {
		if value == nil {
			t.Fatal("expected non-nil here")
		}
	}
}

func TestFakeFillAllocatesIntoAMapLikeWithNonStringKeys(t *testing.T) {
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
	for _, entry := range *resp {
		if entry == nil {
			t.Fatal("expected non-nil here")
		}
	}
}
