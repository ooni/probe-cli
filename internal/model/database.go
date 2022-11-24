package model

//
// Results database
//

import (
	"database/sql"
	"time"
)

// WritableDatabase supports writing and updating data.
type WritableDatabase interface {
	// CreateNetwork will create a new network in the network table
	//
	// Arguments:
	//
	// - loc: loc is the location provider used to instantiate the network
	//
	// Returns either a database network instance or an error
	CreateNetwork(loc LocationProvider) (*DatabaseNetwork, error)

	// CreateOrUpdateURL will create a new URL entry to the urls table if it doesn't
	// exists, otherwise it will update the category code of the one already in
	// there.
	//
	// Arguments:
	//
	// - urlStr is the URL string to create or update
	//
	// - categoryCode is the category code to update
	//
	// - countryCode is the country code to update
	//
	// Returns either the new URL id or an error
	CreateOrUpdateURL(urlStr string, categoryCode string, countryCode string) (int64, error)

	// CreateResult writes the Result to the database a returns a pointer
	// to the Result
	//
	// Arguments:
	//
	// - homePath is the home directory path to make the results directory
	//
	// - testGroupName is used to annotate the database the result
	//
	// - networkID is the id of the underlying network
	//
	// Returns either a database result instance or an error
	CreateResult(homePath string, testGroupName string, networkID int64) (*DatabaseResult, error)

	// UpdateUploadedStatus will check if all the measurements inside of a given result set have been
	// uploaded and if so will set the is_uploaded flag to true
	//
	// Arguments:
	//
	// - result is the database result to update
	//
	// Returns a non-nil error if update failed
	UpdateUploadedStatus(result *DatabaseResult) error

	// Finished marks the result as done and sets the runtime
	//
	// Arguments:
	//
	// - result is the database result to mark as done
	//
	// Returns a non-nil error if result could not be marked as done
	Finished(result *DatabaseResult) error

	// DeleteResult will delete a particular result and the relative measurement on
	// disk.
	//
	// Arguments:
	//
	// - resultID is the id of the database result to be deleted
	//
	// Returns a non-nil error if result could not be deleted
	DeleteResult(resultID int64) error

	// CreateMeasurement writes the measurement to the database a returns a pointer
	// to the Measurement
	//
	// Arguments:
	//
	// - reportID is the report id to annotate
	//
	// - testName is the experiment name to use
	//
	// - measurementDir is the measurement directory path
	//
	// - resultID is the result id to annotate
	//
	// - urlID is the id of the URL input
	//
	// Returns either a database measurement or an error
	CreateMeasurement(reportID sql.NullString, testName string, measurementDir string, idx int,
		resultID int64, urlID sql.NullInt64) (*DatabaseMeasurement, error)

	// AddTestKeys writes the summary to the measurement
	//
	// Arguments:
	//
	// - msmt is the database measurement to update
	//
	// - tk is the testkeys
	//
	// Returns a non-nil error if measurement update failed
	AddTestKeys(msmt *DatabaseMeasurement, tk any) error

	// Done marks the measurement as completed
	//
	// Arguments:
	//
	// - msmt is the database measurement to update
	//
	// Returns a non-nil error if the measurement could not be marked as done
	Done(msmt *DatabaseMeasurement) error

	// UploadFailed writes the error string for the upload failure to the measurement
	//
	// Arguments:
	//
	// - msmt is the database measurement to update
	//
	// - failure is the error string to use
	//
	// Returns a non-nil error if the measurement update failed
	UploadFailed(msmt *DatabaseMeasurement, failure string) error

	// UploadSucceeded writes the error string for the upload failure to the measurement
	//
	// Arguments:
	//
	// - msmt is the database measurement to update
	//
	// Returns a non-nil error is measurement update failed
	UploadSucceeded(msmt *DatabaseMeasurement) error

	// Failed writes the error string to the measurement
	//
	// Arguments:
	//
	// - msmt is the database measurement to update
	//
	// - failure is the error string to use
	//
	// Returns a non-nil error if the measurement update failed
	Failed(msmt *DatabaseMeasurement, failure string) error
}

// ReadableDatabase only supports reading data.
type ReadableDatabase interface {
	// ListResults return the list of results
	//
	// Arguments:
	//
	// Returns either the complete and incomplete database results or an error
	ListResults() ([]DatabaseResultNetwork, []DatabaseResultNetwork, error)

	// ListMeasurements given a result ID
	//
	// Arguments:
	//
	// - resultID is the id of the result to search measurements from
	//
	// Returns the measurements under the given result or an error
	ListMeasurements(resultID int64) ([]DatabaseMeasurementURLNetwork, error)

	// GetMeasurementJSON returns a map[string]interface{} given a database and a measurementID
	//
	// Arguments:
	//
	// - msmtID is the measurement id to generate JSON
	//
	// Returns the measurement JSON or an error
	GetMeasurementJSON(msmtID int64) (map[string]interface{}, error)
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
