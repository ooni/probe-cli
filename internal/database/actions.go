package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/enginex"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
	db "upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"
)

// ListMeasurements given a result ID
func ListMeasurements(sess sqlbuilder.Database, resultID int64) ([]MeasurementURLNetwork, error) {
	measurements := []MeasurementURLNetwork{}
	req := sess.Select(
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
func GetMeasurementJSON(sess sqlbuilder.Database, measurementID int64) (map[string]interface{}, error) {
	var (
		measurement MeasurementURLNetwork
		msmtJSON    map[string]interface{}
	)
	req := sess.Select(
		db.Raw("urls.*"),
		db.Raw("measurements.*"),
	).From("measurements").
		LeftJoin("urls").On("urls.url_id = measurements.url_id").
		Where("measurements.measurement_id= ?", measurementID)
	if err := req.One(&measurement); err != nil {
		log.Errorf("failed to run query %s: %v", req.String(), err)
		return nil, err
	}
	measurementFilePath := measurement.Measurement.MeasurementFilePath.String
	// TODO handle the case in which we have MeasurementFilePath == NULL because
	// it's a beta measurement
	b, err := ioutil.ReadFile(measurementFilePath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &msmtJSON); err != nil {
		return nil, err
	}
	return msmtJSON, nil
}

// GetResultTestKeys returns a list of TestKeys for a given result
func GetResultTestKeys(sess sqlbuilder.Database, resultID int64) (string, error) {
	res := sess.Collection("measurements").Find("result_id", resultID)
	defer res.Close()

	var (
		msmt Measurement
		tk   PerformanceTestKeys
	)
	for res.Next(&msmt) {
		// We only really care about performance keys.
		// Note: since even in case of failure we still initialise an empty struct,
		// it could be that these keys come out as initializes with the default
		// values.
		// XXX we may want to change this behaviour by adding `omitempty` to the
		// struct definition.
		if msmt.TestName != "ndt" && msmt.TestName != "dash" {
			return "{}", nil
		}
		if err := json.Unmarshal([]byte(msmt.TestKeys), &tk); err != nil {
			log.WithError(err).Error("failed to parse testKeys")
			return "{}", err
		}
	}
	b, err := json.Marshal(tk)
	if err != nil {
		log.WithError(err).Error("failed to serialize testKeys")
		return "{}", err
	}
	return string(b), nil
}

// GetMeasurementCounts returns the number of anomalous and total measurement for a given result
func GetMeasurementCounts(sess sqlbuilder.Database, resultID int64) (uint64, uint64, error) {
	var (
		totalCount uint64
		anmlyCount uint64
		err        error
	)
	col := sess.Collection("measurements")

	// XXX these two queries can be done with a single query
	totalCount, err = col.Find("result_id", resultID).
		Count()
	if err != nil {
		log.WithError(err).Error("failed to get total count")
		return totalCount, anmlyCount, err
	}

	anmlyCount, err = col.Find("result_id", resultID).
		And(db.Cond{"is_anomaly": true}).Count()
	if err != nil {
		log.WithError(err).Error("failed to get anmly count")
		return totalCount, anmlyCount, err
	}

	log.Debugf("counts: %d, %d, %d", resultID, totalCount, anmlyCount)
	return totalCount, anmlyCount, err
}

// ListResults return the list of results
func ListResults(sess sqlbuilder.Database) ([]ResultNetwork, []ResultNetwork, error) {
	doneResults := []ResultNetwork{}
	incompleteResults := []ResultNetwork{}
	req := sess.Select(
		db.Raw("networks.*"),
		db.Raw("results.*"),
	).From("results").
		Join("networks").On("results.network_id = networks.network_id").
		OrderBy("results.result_start_time")
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
func DeleteResult(sess sqlbuilder.Database, resultID int64) error {
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

// CreateMeasurement writes the measurement to the database a returns a pointer
// to the Measurement
func CreateMeasurement(sess sqlbuilder.Database, reportID sql.NullString, testName string, measurementDir string, idx int, resultID int64, urlID sql.NullInt64) (*Measurement, error) {
	msmtFilePath := filepath.Join(measurementDir, fmt.Sprintf("msmt-%d.json", idx))
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
	msmt.ID = newID.(int64)
	return &msmt, nil
}

// CreateResult writes the Result to the database a returns a pointer
// to the Result
func CreateResult(sess sqlbuilder.Database, homePath string, testGroupName string, networkID int64) (*Result, error) {
	startTime := time.Now().UTC()

	p, err := utils.MakeResultsDir(homePath, testGroupName, startTime)
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
	result.ID = newID.(int64)
	return &result, nil
}

// CreateNetwork will create a new network in the network table
func CreateNetwork(sess sqlbuilder.Database, loc enginex.LocationProvider) (*Network, error) {
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

	network.ID = newID.(int64)
	return &network, nil
}

// CreateOrUpdateURL will create a new URL entry to the urls table if it doesn't
// exists, otherwise it will update the category code of the one already in
// there.
func CreateOrUpdateURL(sess sqlbuilder.Database, urlStr string, categoryCode string, countryCode string) (int64, error) {
	var url URL

	tx, err := sess.NewTx(nil)
	if err != nil {
		log.WithError(err).Error("failed to create transaction")
		return 0, err
	}
	res := tx.Collection("urls").Find(
		db.Cond{"url": urlStr, "url_country_code": countryCode},
	)
	err = res.One(&url)

	if err == db.ErrNoMoreRows {
		url = URL{
			URL:          sql.NullString{String: urlStr, Valid: true},
			CategoryCode: sql.NullString{String: categoryCode, Valid: true},
			CountryCode:  sql.NullString{String: countryCode, Valid: true},
		}
		newID, insErr := tx.Collection("urls").Insert(url)
		if insErr != nil {
			log.Error("Failed to insert into the URLs table")
			return 0, insErr
		}
		url.ID = sql.NullInt64{Int64: newID.(int64), Valid: true}
	} else if err != nil {
		log.WithError(err).Error("Failed to get single result")
		return 0, err
	} else {
		url.CategoryCode = sql.NullString{String: categoryCode, Valid: true}
		res.Update(url)
	}

	err = tx.Commit()
	if err != nil {
		log.WithError(err).Error("Failed to write to the URL table")
		return 0, err
	}

	log.Debugf("returning url %d", url.ID.Int64)

	return url.ID.Int64, nil
}

// AddTestKeys writes the summary to the measurement
func AddTestKeys(sess sqlbuilder.Database, msmt *Measurement, tk interface{}) error {
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
	if isAnomalyValue.IsValid() == true && isAnomalyValue.Kind() == reflect.Bool {
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
