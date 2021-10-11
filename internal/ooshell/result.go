package ooshell

import "context"

//
// result.go
//
// Contains code to create a result (i.e., the result of
// running a group of experiments together)
//

// resultDB is a result the includes DB information.
type resultDB struct {
	// db is the DB to use.
	db DB

	// env is the underlying Environ.
	env *Environ

	// experiments is the list of experiments to run.
	experiments []string

	// group is the nettest group name.
	group string

	// id is the result ID.
	id int64

	// sess is the underlying measurement session.
	sess *sessionDB
}

// newResultDB creates a new resultDB instance.
func (env *Environ) newResultDB(
	sess *sessionDB, group string, experiments []string) (*resultDB, error) {
	networkID, err := env.DB.NewNetwork(sess.ID(),
		sess.ProbeIP(), sess.ProbeASN(), sess.ProbeNetworkName(), sess.ProbeCC())
	if err != nil {
		return nil, err
	}
	resultID, err := env.DB.NewResult(networkID, group)
	if err != nil {
		return nil, err
	}
	return &resultDB{
		db:          env.DB,
		env:         env,
		experiments: experiments,
		group:       group,
		id:          resultID,
		sess:        sess,
	}, nil
}

// Run runs the experiments contained in this result.
func (r *resultDB) Run(ctx context.Context) error {
	for idx, name := range r.experiments {
		exp, err := r.newExperimentDB(name, idx, len(r.experiments))
		if err != nil {
			return err
		}
		if err := exp.Run(ctx); err != nil {
			return err
		}
	}
	return r.db.SetResultFinished(r.id)
}
