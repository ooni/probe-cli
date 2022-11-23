package model

//
// Database
//

import (
	"database/sql"
	"time"

	"github.com/upper/db/v4"
)

// WritabeDatabase supports writing and updating data.
type WritableDatabase interface {
	// Session returns the database session
	Session() db.Session

	// CreateNetwork will create a new network in the network table
	CreateNetwork(loc LocationProvider) (*DatabaseNetwork, error)

	// CreateOrUpdateURL will create a new URL entry to the urls table if it doesn't
	// exists, otherwise it will update the category code of the one already in
	// there.
	CreateOrUpdateURL(urlStr string, categoryCode string, countryCode string) (int64, error)

	// CreateResult writes the Result to the database a returns a pointer
	// to the Result
	CreateResult(homePath string, testGroupName string, networkID int64) (*DatabaseResult, error)

	// UpdateUploadedStatus will check if all the measurements inside of a given result set have been uploaded and if so will set the is_uploaded flag to true
	UpdateUploadedStatus(result *DatabaseResult) error

	// Finished marks the result as done and sets the runtime
	Finished(result *DatabaseResult) error

	// DeleteResult will delete a particular result and the relative measurement on
	// disk.
	DeleteResult(resultID int64) error

	// CreateMeasurement writes the measurement to the database a returns a pointer
	// to the Measurement
	CreateMeasurement(reportID sql.NullString, testName string, measurementDir string, idx int,
		resultID int64, urlID sql.NullInt64) (*DatabaseMeasurement, error)

	// AddTestKeys writes the summary to the measurement
	AddTestKeys(msmt *DatabaseMeasurement, tk interface{}) error

	// Done marks the measurement as completed
	Done(msmt *DatabaseMeasurement) error

	// UploadFailed writes the error string for the upload failure to the measurement
	UploadFailed(msmt *DatabaseMeasurement, failure string) error

	// UploadSucceeded writes the error string for the upload failure to the measurement
	UploadSucceeded(msmt *DatabaseMeasurement) error

	// Failed writes the error string to the measurement
	Failed(msmt *DatabaseMeasurement, failure string) error

	// Close closes the database session
	Close() error
}

// ReadableDatabase only supports reading data.
type ReadableDatabase interface {
	// Session returns the database session
	Session() db.Session

	// ListResults return the list of results
	ListResults() ([]DatabaseResultNetwork, []DatabaseResultNetwork, error)

	// ListMeasurements given a result ID
	ListMeasurements(resultID int64) ([]DatabaseMeasurementURLNetwork, error)

	// GetMeasurementJSON returns a map[string]interface{} given a database and a measurementID
	GetMeasurementJSON(msmtID int64) (map[string]interface{}, error)

	// Close closes the database session
	Close() error
}

// ResultNetwork is used to represent the structure made from the JOIN
// between the results and networks tables.
type DatabaseResultNetwork struct {
	DatabaseResult  `db:",inline"`
	DatabaseNetwork `db:",inline"`
	AnomalyCount    uint64 `db:"anomaly_count"`
	TotalCount      uint64 `db:"total_count"`
	TestKeys        string `db:"test_keys"`
}

// UploadedTotalCount is the count of the measurements which have been uploaded vs the total measurements in a given result set
type UploadedTotalCount struct {
	UploadedCount int64 `db:",inline"`
	TotalCount    int64 `db:",inline"`
}

// MeasurementURLNetwork is used for the JOIN between Measurement and URL
type DatabaseMeasurementURLNetwork struct {
	DatabaseMeasurement `db:",inline"`
	DatabaseNetwork     `db:",inline"`
	DatabaseResult      `db:",inline"`
	DatabaseURL         `db:",inline"`
}

// DatabaseNetwork represents a network tested by the user
type DatabaseNetwork struct {
	ID          int64  `db:"network_id,omitempty"`
	NetworkName string `db:"network_name"`
	NetworkType string `db:"network_type"`
	IP          string `db:"ip"`
	ASN         uint   `db:"asn"`
	CountryCode string `db:"network_country_code"`
}

// DatabaseURL represents URLs from the testing lists
type DatabaseURL struct {
	ID           sql.NullInt64  `db:"url_id,omitempty"`
	URL          sql.NullString `db:"url"`
	CategoryCode sql.NullString `db:"category_code"`
	CountryCode  sql.NullString `db:"url_country_code"`
}

// Database Measurement model
type DatabaseMeasurement struct {
	ID               int64          `db:"measurement_id,omitempty"`
	TestName         string         `db:"test_name"`
	StartTime        time.Time      `db:"measurement_start_time"`
	Runtime          float64        `db:"measurement_runtime"` // Fractional number of seconds
	IsDone           bool           `db:"measurement_is_done"`
	IsUploaded       bool           `db:"measurement_is_uploaded"`
	IsFailed         bool           `db:"measurement_is_failed"`
	FailureMsg       sql.NullString `db:"measurement_failure_msg,omitempty"`
	IsUploadFailed   bool           `db:"measurement_is_upload_failed"`
	UploadFailureMsg sql.NullString `db:"measurement_upload_failure_msg,omitempty"`
	IsRerun          bool           `db:"measurement_is_rerun"`
	ReportID         sql.NullString `db:"report_id,omitempty"`
	URLID            sql.NullInt64  `db:"url_id,omitempty"` // Used to reference URL
	MeasurementID    sql.NullInt64  `db:"collector_measurement_id,omitempty"`
	IsAnomaly        sql.NullBool   `db:"is_anomaly,omitempty"`
	// FIXME we likely want to support JSON. See: https://github.com/upper/db/issues/462
	TestKeys            string         `db:"test_keys"`
	ResultID            int64          `db:"result_id"`
	ReportFilePath      sql.NullString `db:"report_file_path,omitempty"`
	MeasurementFilePath sql.NullString `db:"measurement_file_path,omitempty"`
}

// Database Result model
type DatabaseResult struct {
	ID             int64     `db:"result_id,omitempty"`
	TestGroupName  string    `db:"test_group_name"`
	StartTime      time.Time `db:"result_start_time"`
	NetworkID      int64     `db:"network_id"`     // Used to include a Network
	Runtime        float64   `db:"result_runtime"` // Runtime is expressed in fractional seconds
	IsViewed       bool      `db:"result_is_viewed"`
	IsDone         bool      `db:"result_is_done"`
	IsUploaded     bool      `db:"result_is_uploaded"`
	DataUsageUp    float64   `db:"result_data_usage_up"`
	DataUsageDown  float64   `db:"result_data_usage_down"`
	MeasurementDir string    `db:"measurement_dir"`
}

// PerformanceTestKeys is the result summary for a performance test
type PerformanceTestKeys struct {
	Upload   float64 `json:"upload"`
	Download float64 `json:"download"`
	Ping     float64 `json:"ping"`
	Bitrate  float64 `json:"median_bitrate"`
}
