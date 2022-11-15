package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/pkg/errors"
	"github.com/upper/db/v4"
)

// ListMeasurements given a result ID
func ListMeasurements(sess db.Session, resultID int64) ([]MeasurementURLNetwork, error) {
	measurements := []MeasurementURLNetwork{}
	req := sess.SQL().Select(
		db.Raw("networks.*"),
		db.Raw("urls.*"),
		db.Raw("measurements.*"),
		db.Raw("results.*"),
	).From("results").
		Join("measurements").On("results.result_id = measurements.result_id").
		Join("networks").On("results.network_id = networks.network_id").
		LeftJoin("urls").On("urls.url_id = measurements.url_id").
		OrderBy("measurements.measurement_start_time").
		Where("results.result_id = ?", resultID)
	if err := req.All(&measurements); err != nil {
		log.Errorf("failed to run query %s: %v", req.String(), err)
		return measurements, err
	}
	return measurements, nil
}

// GetMeasurementJSON returns a map[string]interface{} given a database and a measurementID
func GetMeasurementJSON(sess db.Session, measurementID int64) (map[string]interface{}, error) {
	var (
		measurement MeasurementURLNetwork
		msmtJSON    map[string]interface{}
	)
	req := sess.SQL().Select(
		db.Raw("urls.*"),
		db.Raw("measurements.*"),
	).From("measurements").
		LeftJoin("urls").On("urls.url_id = measurements.url_id").
		Where("measurements.measurement_id= ?", measurementID)
	if err := req.One(&measurement); err != nil {
		log.Errorf("failed to run query %s: %v", req.String(), err)
		return nil, err
	}
	if measurement.Measurement.IsUploaded {
		// TODO(bassosimone): this should be a function exposed by probe-engine
		reportID := measurement.Measurement.ReportID.String
		measurementURL := &url.URL{
			Scheme: "https",
			Host:   "api.ooni.io",
			Path:   "/api/v1/raw_measurement",
		}
		query := url.Values{}
		query.Add("report_id", reportID)
		if measurement.URL.URL.Valid {
			query.Add("input", measurement.URL.URL.String)
		}
		measurementURL.RawQuery = query.Encode()
		log.Debugf("using %s", measurementURL.String())
		resp, err := http.Get(measurementURL.String())
		if err != nil {
			log.Errorf("failed to fetch the measurement %s %s", reportID, measurement.URL.URL.String)
			return nil, err
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&msmtJSON); err != nil {
			log.Error("failed to unmarshal the measurement_json")
			return nil, err
		}
		return msmtJSON, nil
	}
	// MeasurementFilePath might be NULL because the measurement from a
	// 3.0.0-beta install
	if !measurement.Measurement.MeasurementFilePath.Valid {
		log.Error("invalid measurement_file_path")
		log.Error("backup your OONI_HOME and run `ooniprobe reset`")
		return nil, errors.New("cannot access measurement file")
	}
	measurementFilePath := measurement.Measurement.MeasurementFilePath.String
	b, err := os.ReadFile(measurementFilePath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &msmtJSON); err != nil {
		log.Error("failed to unmarshal the measurement_json")
		log.Error("backup your OONI_HOME and run `ooniprobe reset`")
		return nil, err
	}
	return msmtJSON, nil
}

// ListResults return the list of results
func ListResults(sess db.Session) ([]ResultNetwork, []ResultNetwork, error) {
	doneResults := []ResultNetwork{}
	incompleteResults := []ResultNetwork{}
	req := sess.SQL().Select(
		db.Raw("networks.network_name"),
		db.Raw("networks.network_type"),
		db.Raw("networks.ip"),
		db.Raw("networks.asn"),
		db.Raw("networks.network_country_code"),

		db.Raw("results.result_id"),
		db.Raw("results.test_group_name"),
		db.Raw("results.result_start_time"),
		db.Raw("results.network_id"),
		db.Raw("results.result_is_viewed"),
		db.Raw("results.result_runtime"),
		db.Raw("results.result_is_done"),
		db.Raw("results.result_is_uploaded"),
		db.Raw("results.result_data_usage_up"),
		db.Raw("results.result_data_usage_down"),
		db.Raw("results.measurement_dir"),

		db.Raw("COUNT(CASE WHEN measurements.is_anomaly = TRUE THEN 1 END) as anomaly_count"),
		db.Raw("COUNT() as total_count"),
		// The test_keys column are concanetated with the "|" character as a separator.
		// We consider this to be safe since we only really care about values of the
		// performance test_keys where the values are all numbers and none of the keys
		// contain the "|" character.
		db.Raw("group_concat(test_keys, '|') as test_keys"),
	).From("results").
		Join("networks").On("results.network_id = networks.network_id").
		Join("measurements").On("measurements.result_id = results.result_id").
		OrderBy("results.result_start_time").
		GroupBy(
			db.Raw("networks.network_name"),
			db.Raw("networks.network_type"),
			db.Raw("networks.ip"),
			db.Raw("networks.asn"),
			db.Raw("networks.network_country_code"),

			db.Raw("results.result_id"),
			db.Raw("results.test_group_name"),
			db.Raw("results.result_start_time"),
			db.Raw("results.network_id"),
			db.Raw("results.result_is_viewed"),
			db.Raw("results.result_runtime"),
			db.Raw("results.result_is_done"),
			db.Raw("results.result_is_uploaded"),
			db.Raw("results.result_data_usage_up"),
			db.Raw("results.result_data_usage_down"),
			db.Raw("results.measurement_dir"),
		)
	if err := req.Where("result_is_done = true").All(&doneResults); err != nil {
		return doneResults, incompleteResults, errors.Wrap(err, "failed to get result done list")
	}
	if err := req.Where("result_is_done = false").All(&incompleteResults); err != nil {
		return doneResults, incompleteResults, errors.Wrap(err, "failed to get result done list")
	}
	return doneResults, incompleteResults, nil
}

// DeleteResult will delete a particular result and the relative measurement on
// disk.
func DeleteResult(sess db.Session, resultID int64) error {
	var result Result
	res := sess.Collection("results").Find("result_id", resultID)
	if err := res.One(&result); err != nil {
		if err == db.ErrNoMoreRows {
			return err
		}
		log.WithError(err).Error("error in obtaining the result")
		return err
	}
	if err := res.Delete(); err != nil {
		log.WithError(err).Error("failed to delete the result directory")
		return err
	}

	os.RemoveAll(result.MeasurementDir)
	return nil
}

// UpdateUploadedStatus will check if all the measurements inside of a given result set have been uploaded and if so will set the is_uploaded flag to true
func UpdateUploadedStatus(sess db.Session, result *Result) error {
	err := sess.Tx(func(tx db.Session) error {
		uploadedTotal := UploadedTotalCount{}
		req := tx.SQL().Select(
			db.Raw("SUM(measurements.measurement_is_uploaded)"),
			db.Raw("COUNT(*)"),
		).From("results").
			Join("measurements").On("measurements.result_id = results.result_id").
			Where("results.result_id = ?", result.ID)

		err := req.One(&uploadedTotal)
		if err != nil {
			log.WithError(err).Error("failed to retrieve total vs uploaded counts")
			return err
		}
		if uploadedTotal.UploadedCount == uploadedTotal.TotalCount {
			result.IsUploaded = true
		} else {
			result.IsUploaded = false
		}
		err = tx.Collection("results").Find("result_id", result.ID).Update(result)
		if err != nil {
			log.WithError(err).Error("failed to update result")
			return errors.Wrap(err, "updating result")
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Error("Failed to write to the results table")
		return err
	}

	return nil
}

// CreateMeasurement writes the measurement to the database a returns a pointer
// to the Measurement
func CreateMeasurement(sess db.Session, reportID sql.NullString, testName string, measurementDir string, idx int, resultID int64, urlID sql.NullInt64) (*Measurement, error) {
	// TODO we should look into generating this file path in a more robust way.
	// If there are two identical test_names in the same test group there is
	// going to be a clash of test_name
	msmtFilePath := filepath.Join(measurementDir, fmt.Sprintf("msmt-%s-%d.json", testName, idx))
	msmt := Measurement{
		ReportID:            reportID,
		TestName:            testName,
		ResultID:            resultID,
		MeasurementFilePath: sql.NullString{String: msmtFilePath, Valid: true},
		URLID:               urlID,
		IsFailed:            false,
		IsDone:              false,
		// XXX Do we want to have this be part of something else?
		StartTime: time.Now().UTC(),
		TestKeys:  "",
	}

	newID, err := sess.Collection("measurements").Insert(msmt)
	if err != nil {
		return nil, errors.Wrap(err, "creating measurement")
	}
	msmt.ID = newID.ID().(int64)
	return &msmt, nil
}

// CreateResult writes the Result to the database a returns a pointer
// to the Result
func CreateResult(sess db.Session, homePath string, testGroupName string, networkID int64) (*Result, error) {
	startTime := time.Now().UTC()

	p, err := makeResultsDir(homePath, testGroupName, startTime)
	if err != nil {
		return nil, err
	}

	result := Result{
		TestGroupName: testGroupName,
		StartTime:     startTime,
		NetworkID:     networkID,
	}
	result.MeasurementDir = p
	log.Debugf("Creating result %v", result)

	newID, err := sess.Collection("results").Insert(result)
	if err != nil {
		return nil, errors.Wrap(err, "creating result")
	}
	result.ID = newID.ID().(int64)
	return &result, nil
}

// CreateNetwork will create a new network in the network table
func CreateNetwork(sess db.Session, loc engine.LocationProvider) (*Network, error) {
	network := Network{
		ASN:         loc.ProbeASN(),
		CountryCode: loc.ProbeCC(),
		NetworkName: loc.ProbeNetworkName(),
		// On desktop we consider it to always be wifi
		NetworkType: "wifi",
		IP:          loc.ProbeIP(),
	}
	newID, err := sess.Collection("networks").Insert(network)
	if err != nil {
		return nil, err
	}

	network.ID = newID.ID().(int64)
	return &network, nil
}

// CreateOrUpdateURL will create a new URL entry to the urls table if it doesn't
// exists, otherwise it will update the category code of the one already in
// there.
func CreateOrUpdateURL(sess db.Session, urlStr string, categoryCode string, countryCode string) (int64, error) {
	var url URL

	err := sess.Tx(func(tx db.Session) error {
		res := tx.Collection("urls").Find(
			db.Cond{"url": urlStr, "url_country_code": countryCode},
		)
		err := res.One(&url)

		if err == db.ErrNoMoreRows {
			url = URL{
				URL:          sql.NullString{String: urlStr, Valid: true},
				CategoryCode: sql.NullString{String: categoryCode, Valid: true},
				CountryCode:  sql.NullString{String: countryCode, Valid: true},
			}
			newID, insErr := tx.Collection("urls").Insert(url)
			if insErr != nil {
				log.Error("Failed to insert into the URLs table")
				return insErr
			}
			url.ID = sql.NullInt64{Int64: newID.ID().(int64), Valid: true}
		} else if err != nil {
			log.WithError(err).Error("Failed to get single result")
			return err
		} else {
			url.CategoryCode = sql.NullString{String: categoryCode, Valid: true}
			res.Update(url)
		}

		return nil
	})

	if err != nil {
		log.WithError(err).Error("Failed to write to the URL table")
		return 0, err
	}
	return url.ID.Int64, nil
}

// AddTestKeys writes the summary to the measurement
func AddTestKeys(sess db.Session, msmt *Measurement, tk interface{}) error {
	var (
		isAnomaly      bool
		isAnomalyValid bool
	)
	tkBytes, err := json.Marshal(tk)
	if err != nil {
		log.WithError(err).Error("failed to serialize summary")
	}

	// This is necessary so that we can extract from the the opaque testKeys just
	// the IsAnomaly field of bool type.
	// Maybe generics are not so bad after-all, heh golang?
	isAnomalyValue := reflect.ValueOf(tk).FieldByName("IsAnomaly")
	if isAnomalyValue.IsValid() && isAnomalyValue.Kind() == reflect.Bool {
		isAnomaly = isAnomalyValue.Bool()
		isAnomalyValid = true
	}
	msmt.TestKeys = string(tkBytes)
	msmt.IsAnomaly = sql.NullBool{Bool: isAnomaly, Valid: isAnomalyValid}

	err = sess.Collection("measurements").Find("measurement_id", msmt.ID).Update(msmt)
	if err != nil {
		log.WithError(err).Error("failed to update measurement")
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}
