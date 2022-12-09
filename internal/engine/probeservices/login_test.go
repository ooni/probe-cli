package probeservices

import (
	"context"
	"testing"
	"time"
)

func TestMaybeLogin(t *testing.T) {
	t.Run("when we already have a token", func(t *testing.T) {
		clnt := newclient()
		state := State{
			Expire: time.Now().Add(time.Hour),
			Token:  "xx-xxx-x-xxxx",
		}
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		if err := clnt.MaybeLogin(ctx); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("when we have already registered", func(t *testing.T) {
		clnt := newclient()
		state := State{
			// Explicitly empty to clarify what this test does
		}
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		if err := clnt.MaybeLogin(ctx); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the API call fails", func(t *testing.T) {
		clnt := newclient()
		clnt.BaseURL = "\t\t\t" // causes the code to fail
		state := State{
			ClientID: "xx-xxx-x-xxxx",
			Password: "xx",
		}
		if err := clnt.StateFile.Set(state); err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		if err := clnt.MaybeLogin(ctx); err == nil {
			t.Fatal("expected an error here")
		}
	})
}

func TestMaybeLoginIdempotent(t *testing.T) {
	clnt := newclient()
	ctx := context.Background()
	metadata := MetadataFixture()
	if err := clnt.MaybeRegister(ctx, metadata); err != nil {
		t.Fatal(err)
	}
	if err := clnt.MaybeLogin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := clnt.MaybeLogin(ctx); err != nil {
		t.Fatal(err)
	}
	if clnt.LoginCalls.Load() != 1 {
		t.Fatal("called login API too many times")
	}
}
