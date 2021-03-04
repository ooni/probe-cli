package sessionresolver

import (
	"errors"
	"testing"
)

func TestReadStateNothingInKVStore(t *testing.T) {
	reso := &Resolver{KVStore: &memkvstore{}}
	out, err := reso.readstate()
	if !errors.Is(err, errMemkvstoreNotFound) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestReadStateDecodeError(t *testing.T) {
	errMocked := errors.New("mocked error")
	reso := &Resolver{
		KVStore: &memkvstore{},
		codec:   &FakeCodec{DecodeErr: errMocked},
	}
	if err := reso.KVStore.Set(storekey, []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	out, err := reso.readstate()
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestReadStateAndPruneReadStateError(t *testing.T) {
	reso := &Resolver{KVStore: &memkvstore{}}
	out, err := reso.readstateandprune()
	if !errors.Is(err, errMemkvstoreNotFound) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestReadStateAndPruneWithUnsupportedEntries(t *testing.T) {
	reso := &Resolver{KVStore: &memkvstore{}}
	var in []*resolverinfo
	in = append(in, &resolverinfo{})
	if err := reso.writestate(in); err != nil {
		t.Fatal(err)
	}
	out, err := reso.readstateandprune()
	if !errors.Is(err, errNoEntries) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestReadStateDefaultWithMissingEntries(t *testing.T) {
	reso := &Resolver{KVStore: &memkvstore{}}
	// let us simulate that we have just one entry here
	existingURL := "https://dns.google/dns-query"
	existingScore := 0.88
	var in []*resolverinfo
	in = append(in, &resolverinfo{
		URL:   existingURL,
		Score: existingScore,
	})
	if err := reso.writestate(in); err != nil {
		t.Fatal(err)
	}
	// let us seee what we read
	out := reso.readstatedefault()
	if len(out) < 1 {
		t.Fatal("expected non-empty output")
	}
	keys := make(map[string]bool)
	var found bool
	for _, e := range out {
		keys[e.URL] = true
		if e.URL == existingURL {
			if e.Score != existingScore {
				t.Fatal("the score is not what we expected")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("did not found the pre-loaded URL")
	}
	for k := range allbyurl {
		if _, found := keys[k]; !found {
			t.Fatal("missing key", k)
		}
	}
}

func TestWriteStateCannotSerialize(t *testing.T) {
	errMocked := errors.New("mocked error")
	reso := &Resolver{
		codec: &FakeCodec{
			EncodeErr: errMocked,
		},
	}
	existingURL := "https://dns.google/dns-query"
	existingScore := 0.88
	var in []*resolverinfo
	in = append(in, &resolverinfo{
		URL:   existingURL,
		Score: existingScore,
	})
	if err := reso.writestate(in); !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}
