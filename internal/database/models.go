package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/jmoiron/sqlx"
	ooni "github.com/openobservatory/gooni"
	"github.com/pkg/errors"
)

// ResultSummaryFunc is the function used to generate result summaries
type ResultSummaryFunc func(SummaryMap) (string, error)

// SummaryMap contains a mapping from test name to serialized summary for it
type SummaryMap map[string]string

// UpdateOne will run the specified update query and check that it only affected one row
func UpdateOne(db *sqlx.DB, query string, arg interface{}) error {
	res, err := db.NamedExec(query, arg)

	if err != nil {
		return errors.Wrap(err, "updating table")
	}
	count, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "updating table")
	}
	if count != 1 {
		return errors.New("inconsistent update count")
	}
	return nil
}

// Measurement model
type Measurement struct {
	ID             int64     `db:"id"`
	Name           string    `db:"name"`
	StartTime      time.Time `db:"start_time"`
	Runtime        float64   `db:"runtime"` // Fractional number of seconds
	Summary        string    `db:"summary"` // XXX this should be JSON
	ASN            string    `db:"asn"`
	IP             string    `db:"ip"`
	CountryCode    string    `db:"country"`
	State          string    `db:"state"`
	Failure        string    `db:"failure"`
	UploadFailure  string    `db:"upload_failure"`
	Uploaded       bool      `db:"uploaded"`
	ReportFilePath string    `db:"report_file"`
	ReportID       string    `db:"report_id"`
	Input          string    `db:"input"`
	ResultID       int64     `db:"result_id"`
}

// SetGeoIPInfo for the Measurement
func (m *Measurement) SetGeoIPInfo() error {
	return nil
}

// Failed writes the error string to the measurement
func (m *Measurement) Failed(db *sqlx.DB, failure string) error {
	m.Failure = failure

	err := UpdateOne(db, `UPDATE measurements
		SET failure = :failure, state = :state
		WHERE id = :id`, m)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// Done marks the measurement as completed
func (m *Measurement) Done(db *sqlx.DB) error {
	runtime := time.Now().UTC().Sub(m.StartTime)
	m.Runtime = runtime.Seconds()
	m.State = "done"

	err := UpdateOne(db, `UPDATE measurements
		SET state = :state, runtime = :runtime
		WHERE id = :id`, m)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// UploadFailed writes the error string for the upload failure to the measurement
func (m *Measurement) UploadFailed(db *sqlx.DB, failure string) error {
	m.UploadFailure = failure
	m.Uploaded = false

	err := UpdateOne(db, `UPDATE measurements
		SET upload_failure = :upload_failure
		WHERE id = :id`, m)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// UploadSucceeded writes the error string for the upload failure to the measurement
func (m *Measurement) UploadSucceeded(db *sqlx.DB) error {
	m.Uploaded = true

	err := UpdateOne(db, `UPDATE measurements
		SET uploaded = :uploaded
		WHERE id = :id`, m)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// WriteSummary writes the summary to the measurement
func (m *Measurement) WriteSummary(db *sqlx.DB, summary string) error {
	m.Summary = summary

	err := UpdateOne(db, `UPDATE measurements
		SET summary = :summary
		WHERE id = :id`, m)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

//AddToResult adds a measurement to a result
func (m *Measurement) AddToResult(db *sqlx.DB, result *Result) error {
	m.ResultID = result.ID
	finalPath := filepath.Join(result.MeasurementDir,
		filepath.Base(m.ReportFilePath))

	err := os.Rename(m.ReportFilePath, finalPath)
	if err != nil {
		return errors.Wrap(err, "moving report file")
	}
	m.ReportFilePath = finalPath

	err = UpdateOne(db, `UPDATE measurements
		SET result_id = :result_id, report_file = :report_file
		WHERE id = :id`, m)
	if err != nil {
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}

// CreateMeasurement writes the measurement to the database a returns a pointer
// to the Measurement
func CreateMeasurement(db *sqlx.DB, m Measurement, i string) (*Measurement, error) {
	// XXX Do we want to have this be part of something else?
	m.StartTime = time.Now().UTC()
	m.Input = i
	m.State = "active"

	res, err := db.NamedExec(`INSERT INTO measurements
		(name, start_time,
			asn, ip, country,
			state, failure, report_file,
			report_id, input,
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
	ID             int64     `db:"id"`
	Name           string    `db:"name"`
	StartTime      time.Time `db:"start_time"`
	Runtime        float64   `db:"runtime"` // Runtime is expressed in fractional seconds
	Summary        string    `db:"summary"` // XXX this should be JSON
	Done           bool      `db:"done"`
	DataUsageUp    int64     `db:"data_usage_up"`
	DataUsageDown  int64     `db:"data_usage_down"`
	MeasurementDir string    `db:"measurement_dir"`
}

// MakeSummaryMap return a mapping of test names to summaries for the given
// result
func MakeSummaryMap(db *sqlx.DB, r *Result) (SummaryMap, error) {
	summaryMap := SummaryMap{}

	msmts := []Measurement{}
	// XXX maybe we only want to select some of the columns
	err := db.Select(&msmts, "SELECT name, summary FROM measurements WHERE result_id = $1", r.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get measurements")
	}
	for _, msmt := range msmts {
		summaryMap[msmt.Name] = msmt.Summary
	}
	return summaryMap, nil
}

// Finished marks the result as done and sets the runtime
func (r *Result) Finished(db *sqlx.DB, makeSummary ResultSummaryFunc) error {
	if r.Done == true || r.Runtime != 0 {
		return errors.New("Result is already finished")
	}
	r.Runtime = time.Now().UTC().Sub(r.StartTime).Seconds()
	r.Done = true
	// XXX add in here functionality to compute the summary
	summaryMap, err := MakeSummaryMap(db, r)
	if err != nil {
		return err
	}

	r.Summary, err = makeSummary(summaryMap)
	if err != nil {
		return err
	}

	err = UpdateOne(db, `UPDATE results
		SET done = :done, runtime = :runtime, summary = :summary
		WHERE id = :id`, r)
	if err != nil {
		return errors.Wrap(err, "updating finished result")
	}
	return nil
}

// MakeResultsPath creates and returns a directory for the result
func MakeResultsPath(r *Result) (string, error) {
	home, err := ooni.GetOONIHome()
	if err != nil {
		return "", errors.Wrap(err, "default measurements path")
	}
	p := filepath.Join(home, "msmts",
		fmt.Sprintf("%s-%s", r.Name, r.StartTime.Format(time.RFC3339Nano)))

	// If the path already exists, this is a problem. It should not clash, because
	// we are using nanosecond precision for the starttime.
	if _, e := os.Stat(p); e == nil {
		return "", errors.New("results path already exists")
	}
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return "", err
	}
	return p, nil
}

// CreateResult writes the Result to the database a returns a pointer
// to the Result
func CreateResult(db *sqlx.DB, r Result) (*Result, error) {
	log.Debugf("Creating result %v", r)

	p, err := MakeResultsPath(&r)
	if err != nil {
		return nil, err
	}
	r.MeasurementDir = p
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
