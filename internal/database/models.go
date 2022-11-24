package database

import (
	"database/sql"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/pkg/errors"
)

// Finished implements WritableDatabase.Finished
func (d *Database) Finished(result *model.DatabaseResult) error {
	if result.IsDone || result.Runtime != 0 {
		return errors.New("Result is already finished")
	}
	result.Runtime = time.Now().UTC().Sub(result.StartTime).Seconds()
	result.IsDone = true

	err := d.sess.Collection("results").Find("result_id", result.ID).Update(result)
	if err != nil {
		return errors.Wrap(err, "updating finished result")
	}
	return nil
}

// Failed implements WritableDatabase.Failed
func (d *Database) Failed(msmt *model.DatabaseMeasurement, failure string) error {
	msmt.FailureMsg = sql.NullString{String: failure, Valid: true}
	msmt.IsFailed = true
	err := d.sess.Collection("measurements").Find("measurement_id", msmt.ID).Update(msmt)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// Done implements WritableDatabase.Done
func (d *Database) Done(msmt *model.DatabaseMeasurement) error {
	runtime := time.Now().UTC().Sub(msmt.StartTime)
	msmt.Runtime = runtime.Seconds()
	msmt.IsDone = true
	err := d.sess.Collection("measurements").Find("measurement_id", msmt.ID).Update(msmt)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// UploadFailed implements WritableDatabase.UploadFailed
func (d *Database) UploadFailed(msmt *model.DatabaseMeasurement, failure string) error {
	msmt.UploadFailureMsg = sql.NullString{String: failure, Valid: true}
	msmt.IsUploaded = false
	err := d.sess.Collection("measurements").Find("measurement_id", msmt.ID).Update(msmt)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// UploadSucceeded implements WritableDatabase.UploadSucceeded
func (d *Database) UploadSucceeded(msmt *model.DatabaseMeasurement) error {
	msmt.IsUploaded = true
	err := d.sess.Collection("measurements").Find("measurement_id", msmt.ID).Update(msmt)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}
