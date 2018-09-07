package database

import (
	"database/sql"
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

// ListResults return the list of results
func ListResults(sess sqlbuilder.Database) ([]ResultNetwork, []ResultNetwork, error) {
	doneResults := []ResultNetwork{}
	incompleteResults := []ResultNetwork{}

	req := sess.Select(
		"networks.id AS network_id",
		db.Raw("results.*"),
		db.Raw("networks.*"),
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
