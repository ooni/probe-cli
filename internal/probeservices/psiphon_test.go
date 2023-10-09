package probeservices

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestFetchPsiphonConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	clnt := newclient()
	if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
		t.Fatal(err)
	}
	if err := clnt.MaybeLogin(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := clnt.FetchPsiphonConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var config interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
}

func TestFetchPsiphonConfigNotRegistered(t *testing.T) {
	clnt := newclient()
	state := State{
		// Explicitly empty so the test is more clear
	}
	if err := clnt.StateFile.Set(state); err != nil {
		t.Fatal(err)
	}
	data, err := clnt.FetchPsiphonConfig(context.Background())
	if !errors.Is(err, ErrNotRegistered) {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}
