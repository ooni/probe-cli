package oonimkall

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/version"
)

func TestTaskKVSToreFSBuilderEngine(t *testing.T) {
	b := &taskKVStoreFSBuilderEngine{}
	store, err := b.NewFS("testdata/state")
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("expected non-nil store here")
	}
}

func TestTaskSessionBuilderEngine(t *testing.T) {
	t.Run("NewSession", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			builder := &taskSessionBuilderEngine{}
			ctx := context.Background()
			config := engine.SessionConfig{
				Logger:          log.Log,
				SoftwareName:    "ooniprobe-cli",
				SoftwareVersion: version.Version,
			}
			sess, err := builder.NewSession(ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			sess.Close()
		})

		t.Run("on failure", func(t *testing.T) {
			builder := &taskSessionBuilderEngine{}
			ctx := context.Background()
			config := engine.SessionConfig{}
			sess, err := builder.NewSession(ctx, config)
			if err == nil {
				t.Fatal("expected an error here")
			}
			if sess != nil {
				t.Fatal("expected nil session here")
			}
		})
	})
}

func TestTaskSessionEngine(t *testing.T) {

	// newSession is a helper function for creating a new session.
	newSession := func(t *testing.T) taskSession {
		builder := &taskSessionBuilderEngine{}
		ctx := context.Background()
		config := engine.SessionConfig{
			Logger:          log.Log,
			SoftwareName:    "ooniprobe-cli",
			SoftwareVersion: version.Version,
		}
		sess, err := builder.NewSession(ctx, config)
		if err != nil {
			t.Fatal(err)
		}
		return sess
	}

	t.Run("NewExperimentBuilderByName", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			sess := newSession(t)
			builder, err := sess.NewExperimentBuilderByName("ndt")
			if err != nil {
				t.Fatal(err)
			}
			if builder == nil {
				t.Fatal("expected non-nil builder")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			sess := newSession(t)
			builder, err := sess.NewExperimentBuilderByName("antani")
			if err == nil {
				t.Fatal("expected an error here")
			}
			if builder != nil {
				t.Fatal("expected nil builder")
			}
		})
	})
}

func TestTaskExperimentBuilderEngine(t *testing.T) {

	// newBuilder is a helper function for creating a new session
	// as well as a new experiment builder
	newBuilder := func(t *testing.T) (taskSession, taskExperimentBuilder) {
		builder := &taskSessionBuilderEngine{}
		ctx := context.Background()
		config := engine.SessionConfig{
			Logger:          log.Log,
			SoftwareName:    "ooniprobe-cli",
			SoftwareVersion: version.Version,
		}
		sess, err := builder.NewSession(ctx, config)
		if err != nil {
			t.Fatal(err)
		}
		expBuilder, err := sess.NewExperimentBuilderByName("ndt")
		if err != nil {
			t.Fatal(err)
		}
		return sess, expBuilder
	}

	t.Run("NewExperiment", func(t *testing.T) {
		_, builder := newBuilder(t)
		exp := builder.NewExperimentInstance()
		if exp == nil {
			t.Fatal("expected non-nil experiment here")
		}
	})
}
