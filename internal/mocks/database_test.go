package mocks

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDatabase(t *testing.T) {
	t.Run("CreateNetwork", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockCreateNetwork: func(loc model.LocationProvider) (*model.DatabaseNetwork, error) {
				return nil, expected
			},
		}
		sess := &LocationProvider{}
		network, err := db.CreateNetwork(sess)
		if network != nil {
			t.Fatal("expected nil network")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("CreateOrUpdateURL", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockCreateOrUpdateURL: func(urlStr, categoryCode, countryCode string) (int64, error) {
				return int64(0), expected
			},
		}
		urlID, err := db.CreateOrUpdateURL("https://google.com", "", "")
		if urlID != int64(0) {
			t.Fatal("expected urlID 0")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("CreateResult", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockCreateResult: func(homePath, testGroupName string, networkID int64) (*model.DatabaseResult, error) {
				return nil, expected
			},
		}
		result, err := db.CreateResult("home", "circumvention", 0)
		if result != nil {
			t.Fatal("expected nil result")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("UpdateUploadedStatus", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockUpdateUploadedStatus: func(result *model.DatabaseResult) error {
				return expected
			},
		}
		result := &model.DatabaseResult{}
		err := db.UpdateUploadedStatus(result)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("Finished", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockFinished: func(result *model.DatabaseResult) error {
				return expected
			},
		}
		result := &model.DatabaseResult{}
		err := db.Finished(result)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("DeleteResult", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockDeleteResult: func(resultID int64) error {
				return expected
			},
		}
		err := db.DeleteResult(0)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("CreateMeasurement", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockCreateMeasurement: func(reportID sql.NullString, testName, measurementDir string,
				idx int, resultID int64, urlID sql.NullInt64) (*model.DatabaseMeasurement, error) {
				return nil, expected
			},
		}
		msmt, err := db.CreateMeasurement(sql.NullString{}, "web_connectivity", "/", 0, 0, sql.NullInt64{})
		if msmt != nil {
			t.Fatal("expected nil measurement")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("AddTestKeys", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockAddTestKeys: func(msmt *model.DatabaseMeasurement, tk interface{}) error {
				return expected
			},
		}
		tk := make(map[string]string) // use a random type to pass as any in test keys
		err := db.AddTestKeys(&model.DatabaseMeasurement{}, tk)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("Done", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockDone: func(msmt *model.DatabaseMeasurement) error {
				return expected
			},
		}
		err := db.Done(&model.DatabaseMeasurement{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})
	t.Run("Done", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockDone: func(msmt *model.DatabaseMeasurement) error {
				return expected
			},
		}
		err := db.Done(&model.DatabaseMeasurement{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("UploadFailed", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockUploadFailed: func(msmt *model.DatabaseMeasurement, failure string) error {
				return expected
			},
		}
		err := db.UploadFailed(&model.DatabaseMeasurement{}, "measurement upload failed")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("UploadSucceeded", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockUploadSucceeded: func(msmt *model.DatabaseMeasurement) error {
				return expected
			},
		}
		err := db.UploadSucceeded(&model.DatabaseMeasurement{})
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("Failed", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockFailed: func(msmt *model.DatabaseMeasurement, failure string) error {
				return expected
			},
		}
		err := db.Failed(&model.DatabaseMeasurement{}, "failed")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("ListResults", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockListResults: func() ([]model.DatabaseResultNetwork, []model.DatabaseResultNetwork, error) {
				return nil, nil, expected
			},
		}
		doneResults, incompleteResults, err := db.ListResults()
		if doneResults != nil {
			t.Fatal("expected nil done results")
		}
		if incompleteResults != nil {
			t.Fatal("expected nil incomplete results")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("ListMeasurements", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockListMeasurements: func(resultID int64) ([]model.DatabaseMeasurementURLNetwork, error) {
				return nil, expected
			},
		}
		msmts, err := db.ListMeasurements(0)
		if msmts != nil {
			t.Fatal("expected nil measurements")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("GetMeasurementJSON", func(t *testing.T) {
		expected := errors.New("mocked")
		db := &Database{
			MockGetMeasurementJSON: func(msmtID int64) (map[string]interface{}, error) {
				return nil, expected
			},
		}
		msmtJSON, err := db.GetMeasurementJSON(0)
		if msmtJSON != nil {
			t.Fatal("expected nil measurement JSON")
		}
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
	})
}
