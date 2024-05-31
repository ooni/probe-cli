package oonimkall

//
// This file implements taskSession and derived types.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// taskKVStoreFSBuilderEngine creates a new KVStore
// using the ./internal/engine package.
type taskKVStoreFSBuilderEngine struct{}

var _ taskKVStoreFSBuilder = &taskKVStoreFSBuilderEngine{}

func (*taskKVStoreFSBuilderEngine) NewFS(path string) (model.KeyValueStore, error) {
	return kvstore.NewFS(path)
}

// taskSessionBuilderEngine builds a new session
// using the ./internal/engine package.
type taskSessionBuilderEngine struct{}

var _ taskSessionBuilder = &taskSessionBuilderEngine{}

// NewSession implements taskSessionBuilder.NewSession.
func (b *taskSessionBuilderEngine) NewSession(ctx context.Context,
	config engine.SessionConfig) (taskSession, error) {
	sess, err := engine.NewSession(ctx, config)
	// note: here we need to explicitly return nil because we're changing
	// the type and we would not otherwise get a nil session on error
	if err != nil {
		return nil, err
	}
	return sess, nil
}
