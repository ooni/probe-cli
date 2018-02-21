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

// Update the measurement in the database
func (r Measurement) Update(db *sqlx.DB) error {
	// XXX implement me
	return nil
}

// Result model
type Result struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	StartTime     time.Time `db:"start_time"`
	EndTime       time.Time `db:"end_time"`
	Summary       string    `db:"summary"` // XXX this should be JSON
	Done          bool      `db:"done"`
	DataUsageUp   int64     `db:"data_usage_up"`
	DataUsageDown int64     `db:"data_usage_down"`
}

// Update the Result in the database
func (r Result) Update(db *sqlx.DB) error {
	// XXX implement me
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
		return nil, errors.Wrap(err, "creating measurement")
	}
	r.ID = id
	return &r, nil
}
