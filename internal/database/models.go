package database

import (
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/jmoiron/sqlx"
	"github.com/ooni/probe-cli/nettests/summary"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
)

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

// ListMeasurements given a result ID
func ListMeasurements(db *sqlx.DB, resultID int64) ([]*Measurement, error) {
	measurements := []*Measurement{}

	rows, err := db.Query(`SELECT id, name,
		start_time, runtime,
		country,
		asn,
		summary,
		input
		FROM measurements
		WHERE result_id = ?
		ORDER BY start_time;`, resultID)
	if err != nil {
		return measurements, errors.Wrap(err, "failed to get measurement list")
	}

	for rows.Next() {
		msmt := Measurement{}
		err = rows.Scan(&msmt.ID, &msmt.Name,
			&msmt.StartTime, &msmt.Runtime,
			&msmt.CountryCode,
			&msmt.ASN,
			&msmt.Summary, &msmt.Input,
			//&result.DataUsageUp, &result.DataUsageDown)
		)
		if err != nil {
			log.WithError(err).Error("failed to fetch a row")
			continue
		}
		measurements = append(measurements, &msmt)
	}

	return measurements, nil
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

// AddToResult adds a measurement to a result
func (m *Measurement) AddToResult(db *sqlx.DB, result *Result) error {
	var err error

	m.ResultID = result.ID
	finalPath := filepath.Join(result.MeasurementDir,
		filepath.Base(m.ReportFilePath))

	// If the finalPath already exists, it means it has already been moved there.
	// This happens in multi input reports
	if _, err = os.Stat(finalPath); os.IsNotExist(err) {
		err = os.Rename(m.ReportFilePath, finalPath)
		if err != nil {
			return errors.Wrap(err, "moving report file")
		}
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
	Country        string    `db:"country"`
	ASN            string    `db:"asn"`
	NetworkName    string    `db:"network_name"`
	Runtime        float64   `db:"runtime"` // Runtime is expressed in fractional seconds
	Summary        string    `db:"summary"` // XXX this should be JSON
	Done           bool      `db:"done"`
	DataUsageUp    int64     `db:"data_usage_up"`
	DataUsageDown  int64     `db:"data_usage_down"`
	MeasurementDir string    `db:"measurement_dir"`
}

// ListResults return the list of results
func ListResults(db *sqlx.DB) ([]*Result, []*Result, error) {
	doneResults := []*Result{}
	incompleteResults := []*Result{}

	rows, err := db.Query(`SELECT id, name,
		start_time, runtime,
		network_name, country,
		asn,
		summary, done
		FROM results
		WHERE done = 1
		ORDER BY start_time;`)
	if err != nil {
		return doneResults, incompleteResults, errors.Wrap(err, "failed to get result done list")
	}

	for rows.Next() {
		result := Result{}
		err = rows.Scan(&result.ID, &result.Name,
			&result.StartTime, &result.Runtime,
			&result.NetworkName, &result.Country,
			&result.ASN,
			&result.Summary, &result.Done,
			//&result.DataUsageUp, &result.DataUsageDown)
		)
		if err != nil {
			log.WithError(err).Error("failed to fetch a row")
			continue
		}
		doneResults = append(doneResults, &result)
	}

	rows, err = db.Query(`SELECT
		id, name,
		start_time,
		network_name, country,
		asn
		FROM results
		WHERE done != 1
		ORDER BY start_time;`)
	if err != nil {
		return doneResults, incompleteResults, errors.Wrap(err, "failed to get result done list")
	}

	for rows.Next() {
		result := Result{Done: false}
		err = rows.Scan(&result.ID, &result.Name, &result.StartTime,
			&result.NetworkName, &result.Country,
			&result.ASN)
		if err != nil {
			log.WithError(err).Error("failed to fetch a row")
			continue
		}
		incompleteResults = append(incompleteResults, &result)
	}
	return doneResults, incompleteResults, nil
}

// MakeSummaryMap return a mapping of test names to summaries for the given
// result
func MakeSummaryMap(db *sqlx.DB, r *Result) (summary.SummaryMap, error) {
	summaryMap := summary.SummaryMap{}

	msmts := []Measurement{}
	// XXX maybe we only want to select some of the columns
	err := db.Select(&msmts, "SELECT name, summary FROM measurements WHERE result_id = $1", r.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get measurements")
	}
	for _, msmt := range msmts {
		val, ok := summaryMap[msmt.Name]
		if ok {
			summaryMap[msmt.Name] = append(val, msmt.Summary)
		} else {
			summaryMap[msmt.Name] = []string{msmt.Summary}
		}
	}
	return summaryMap, nil
}

// Finished marks the result as done and sets the runtime
func (r *Result) Finished(db *sqlx.DB, makeSummary summary.ResultSummaryFunc) error {
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

// CreateResult writes the Result to the database a returns a pointer
// to the Result
func CreateResult(db *sqlx.DB, homePath string, r Result) (*Result, error) {
	log.Debugf("Creating result %v", r)

	p, err := utils.MakeResultsDir(homePath, r.Name, r.StartTime)
	if err != nil {
		return nil, err
	}
	r.MeasurementDir = p
	res, err := db.NamedExec(`INSERT INTO results
		(name, start_time, country, network_name, asn)
		VALUES (:name,:start_time,:country,:network_name,:asn)`,
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
