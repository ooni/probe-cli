package mocks

import (
	"database/sql"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/upper/db/v4"
)

// Database allows mocking a database
type Database struct {
	MockSession              func() db.Session
	MockCreateNetwork        func(loc model.LocationProvider) (*model.DatabaseNetwork, error)
	MockCreateOrUpdateURL    func(urlStr string, categoryCode string, countryCode string) (int64, error)
	MockCreateResult         func(homePath string, testGroupName string, networkID int64) (*model.DatabaseResult, error)
	MockUpdateUploadedStatus func(result *model.DatabaseResult) error
	MockFinished             func(result *model.DatabaseResult) error
	MockDeleteResult         func(resultID int64) error
	MockCreateMeasurement    func(reportID sql.NullString, testName string, measurementDir string,
		idx int, resultID int64, urlID sql.NullInt64) (*model.DatabaseMeasurement, error)
	MockAddTestKeys        func(msmt *model.DatabaseMeasurement, tk interface{}) error
	MockDone               func(msmt *model.DatabaseMeasurement) error
	MockUploadFailed       func(msmt *model.DatabaseMeasurement, failure string) error
	MockUploadSucceeded    func(msmt *model.DatabaseMeasurement) error
	MockFailed             func(msmt *model.DatabaseMeasurement, failure string) error
	MockListResults        func() ([]model.DatabaseResultNetwork, []model.DatabaseResultNetwork, error)
	MockListMeasurements   func(resultID int64) ([]model.DatabaseMeasurementURLNetwork, error)
	MockGetMeasurementJSON func(msmtID int64) (map[string]interface{}, error)
	MockClose              func() error
}

var _ model.WritableDatabase = &Database{}

// Session calls MockSession
func (d *Database) Session() db.Session {
	return d.MockSession()
}

// CreateNetwork calls MockCreateNetwork
func (d *Database) CreateNetwork(loc model.LocationProvider) (*model.DatabaseNetwork, error) {
	return d.MockCreateNetwork(loc)
}

// CreateOrUpdateURL calls MockCreateOrUpdateURL
func (d *Database) CreateOrUpdateURL(urlStr string, categoryCode string, countryCode string) (int64, error) {
	return d.MockCreateOrUpdateURL(urlStr, categoryCode, countryCode)
}

// CreateResult calls MockCreateResult
func (d *Database) CreateResult(homePath string, testGroupName string, networkID int64) (*model.DatabaseResult, error) {
	return d.MockCreateResult(homePath, testGroupName, networkID)
}

// UpdateUploadedStatus calls MockUpdateUploadedStatus
func (d *Database) UpdateUploadedStatus(result *model.DatabaseResult) error {
	return d.MockUpdateUploadedStatus(result)
}

// Finished calls MockFinished
func (d *Database) Finished(result *model.DatabaseResult) error {
	return d.MockFinished(result)
}

// DeleteResult calls MockDeleteResult
func (d *Database) DeleteResult(resultID int64) error {
	return d.MockDeleteResult(resultID)
}

// CreateMeasurement calls MockCreateMeasurement
func (d *Database) CreateMeasurement(reportID sql.NullString, testName string, measurementDir string,
	idx int, resultID int64, urlID sql.NullInt64) (*model.DatabaseMeasurement, error) {
	return d.MockCreateMeasurement(reportID, testName, measurementDir, idx, resultID, urlID)
}

// AddTestKeys calls MockAddTestKeys
func (d *Database) AddTestKeys(msmt *model.DatabaseMeasurement, tk interface{}) error {
	return d.MockAddTestKeys(msmt, tk)
}

// Done calls MockDone
func (d *Database) Done(msmt *model.DatabaseMeasurement) error {
	return d.MockDone(msmt)
}

// UploadFailed calls MockUploadFailed
func (d *Database) UploadFailed(msmt *model.DatabaseMeasurement, failure string) error {
	return d.MockUploadFailed(msmt, failure)
}

// UploadSucceeded calls MockUploadSucceeded
func (d *Database) UploadSucceeded(msmt *model.DatabaseMeasurement) error {
	return d.MockUploadSucceeded(msmt)
}

// Failed calls MockFailed
func (d *Database) Failed(msmt *model.DatabaseMeasurement, failure string) error {
	return d.MockFailed(msmt, failure)
}

var _ model.ReadableDatabase = &Database{}

// ListResults calla MockListResults
func (d *Database) ListResults() ([]model.DatabaseResultNetwork, []model.DatabaseResultNetwork, error) {
	return d.MockListResults()
}

// ListMeasurements calls MockListMeasurements
func (d *Database) ListMeasurements(resultID int64) ([]model.DatabaseMeasurementURLNetwork, error) {
	return d.MockListMeasurements(resultID)
}

// GetMeasurementJSON calls MockGetMeasurementJSON
func (d *Database) GetMeasurementJSON(msmtID int64) (map[string]interface{}, error) {
	return d.MockGetMeasurementJSON(msmtID)
}

// Close calls MockClose
func (d *Database) Close() error {
	return d.MockClose()
}
