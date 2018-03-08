package database

import (
	"time"

	"github.com/apex/log"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Measurement model
type Measurement struct {
	ID             int64     `db:"id"`
	Name           string    `db:"name"`
	StartTime      time.Time `db:"start_time"`
	EndTime        time.Time `db:"end_time"`
	Summary        string    `db:"summary"` // XXX this should be JSON
	ASN            int64     `db:"asn"`
	IP             string    `db:"ip"`
	CountryCode    string    `db:"country"`
	State          string    `db:"state"`
	Failure        string    `db:"failure"`
	ReportFilePath string    `db:"report_file"`
	ReportID       string    `db:"report_id"`
	Input          string    `db:"input"`
	MeasurementID  string    `db:"measurement_id"`
	ResultID       string    `db:"result_id"`
}

// SetGeoIPInfo for the Measurement
func (m *Measurement) SetGeoIPInfo() error {
	return nil
}

// CreateMeasurement writes the measurement to the database a returns a pointer
// to the Measurement
func CreateMeasurement(db *sqlx.DB, m Measurement) (*Measurement, error) {
	res, err := db.NamedExec(`INSERT INTO measurements
		(name, start_time,
			summary, asn, ip, country,
			state, failure, report_file,
			report_id, input, measurement_id,
			result_id)
		VALUES (:name,:start_time,
			:asn,:ip,:country,
			:state,:failure,:report_file,
			:report_id,:input,
			:result_id)`,
		m)
	if err != nil {
		return nil, errors.Wrap(err, "creating measurement")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "creating measurement")
	}
	m.ID = id
	return &m, nil
}

// Result model
type Result struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	StartTime     time.Time `db:"start_time"`
	Runtime       float64   `db:"runtime"` // Runtime is expressed in Microseconds
	Summary       string    `db:"summary"` // XXX this should be JSON
	Done          bool      `db:"done"`
	DataUsageUp   int64     `db:"data_usage_up"`
	DataUsageDown int64     `db:"data_usage_down"`

	started time.Time
}

// Started marks the Result as having started
func (r *Result) Started(db *sqlx.DB) error {
	r.started = time.Now()
	return nil
}

// Finished marks the result as done and sets the runtime
func (r *Result) Finished(db *sqlx.DB) error {
	if r.Done == true || r.Runtime != 0 {
		return errors.New("Result is already finished")
	}
	r.Runtime = float64(time.Now().Sub(r.started)) / float64(time.Microsecond)
	r.Done = true

	res, err := db.NamedExec(`UPDATE results
		SET done = true, runtime = :runtime
		WHERE id = :id`, r)
	if err != nil {
		return errors.Wrap(err, "updating result")
	}
	count, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "updating result")
	}
	if count != 1 {
		return errors.New("inconsistent update count")
	}
	return nil
}

// CreateResult writes the Result to the database a returns a pointer
// to the Result
func CreateResult(db *sqlx.DB, r Result) (*Result, error) {
	log.Debugf("Creating result %s", r)
	res, err := db.NamedExec(`INSERT INTO results
		(name, start_time)
		VALUES (:name,:start_time)`,
		r)
	if err != nil {
		return nil, errors.Wrap(err, "creating result")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "creating result")
	}
	r.ID = id
	return &r, nil
}
