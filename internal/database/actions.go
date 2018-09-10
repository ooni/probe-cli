package database

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
	db "upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"
)

// ListMeasurements given a result ID
func ListMeasurements(sess sqlbuilder.Database, resultID int64) ([]MeasurementURLNetwork, error) {
	measurements := []MeasurementURLNetwork{}

	req := sess.Select(
		"networks.id as network_id",
		"results.id as result_id",
		"urls.id as url_id",
		db.Raw("networks.*"),
		db.Raw("urls.*"),
		db.Raw("measurements.*"),
	).From("results").
		Join("measurements").On("results.id = measurements.result_id").
		Join("networks").On("results.network_id = networks.id").
		LeftJoin("urls").On("urls.id = measurements.url_id").
		OrderBy("measurements.start_time").
		Where("results.id = ?", resultID)

	if err := req.All(&measurements); err != nil {
		log.Errorf("failed to run query %s: %v", req.String(), err)
		return measurements, err
	}
	return measurements, nil
}

// GetResultTestKeys returns a list of TestKeys for a given measurements
func GetResultTestKeys(sess sqlbuilder.Database, resultID int64) (string, error) {
	res := sess.Collection("measurements").Find("result_id", resultID)
	defer res.Close()

	var msmt Measurement
	for res.Next(&msmt) {
		if msmt.TestName == "web_connectivity" {
			break
		}
		// We only really care about the NDT TestKeys
		if msmt.TestName == "ndt" {
			return msmt.TestKeys, nil
		}
	}
	return "{}", nil
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
		"networks.id AS network_id",
		"results.id AS result_id",
		db.Raw("networks.*"),
		db.Raw("results.*"),
	).From("results").
		Join("networks").On("results.network_id = networks.id").
		OrderBy("results.start_time")

	if err := req.Where("is_done = true").All(&doneResults); err != nil {
		return doneResults, incompleteResults, errors.Wrap(err, "failed to get result done list")
	}
	if err := req.Where("is_done = false").All(&incompleteResults); err != nil {
		return doneResults, incompleteResults, errors.Wrap(err, "failed to get result done list")
	}

	return doneResults, incompleteResults, nil
}

// CreateMeasurement writes the measurement to the database a returns a pointer
// to the Measurement
func CreateMeasurement(sess sqlbuilder.Database, reportID sql.NullString, testName string, resultID int64, reportFilePath string, urlID sql.NullInt64) (*Measurement, error) {
	msmt := Measurement{
		ReportID:       reportID,
		TestName:       testName,
		ResultID:       resultID,
		ReportFilePath: reportFilePath,
		URLID:          urlID,
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
func CreateNetwork(sess sqlbuilder.Database, location *utils.LocationInfo) (*Network, error) {
	network := Network{
		ASN:         location.ASN,
		CountryCode: location.CountryCode,
		NetworkName: location.NetworkName,
		// On desktop we consider it to always be wifi
		NetworkType: "wifi",
		IP:          location.IP,
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
func CreateOrUpdateURL(sess sqlbuilder.Database, url string, categoryCode string, countryCode string) (int64, error) {
	var urlID int64

	res, err := sess.Update("urls").Set(
		"url", url,
		"category_code", categoryCode,
		"country_code", countryCode,
	).Where("url = ? AND country_code = ?", url, countryCode).Exec()

	if err != nil {
		log.Error("Failed to write to the URL table")
		return 0, err
	}
	affected, err := res.RowsAffected()

	if err != nil {
		log.Error("Failed to get affected row count")
		return 0, err
	}
	if affected == 0 {
		newID, err := sess.Collection("urls").Insert(
			URL{
				URL:          sql.NullString{String: url, Valid: true},
				CategoryCode: sql.NullString{String: categoryCode, Valid: true},
				CountryCode:  sql.NullString{String: countryCode, Valid: true},
			})
		if err != nil {
			log.Error("Failed to insert into the URLs table")
			return 0, err
		}
		urlID = newID.(int64)
	} else {
		lastID, err := res.LastInsertId()
		if err != nil {
			log.Error("failed to get URL ID")
			return 0, err
		}
		urlID = lastID
	}
	log.Debugf("returning url %d", urlID)

	return urlID, nil
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

	err = sess.Collection("measurements").Find("id", msmt.ID).Update(msmt)
	if err != nil {
		log.WithError(err).Error("failed to update measurement")
		return errors.Wrap(err, "updating measurement")
	}
	return nil
}
