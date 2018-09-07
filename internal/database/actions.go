package database

import (
	"database/sql"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
	"upper.io/db.v3/lib/sqlbuilder"
)

// ListMeasurements given a result ID
func ListMeasurements(db sqlbuilder.Database, resultID int64) ([]*Measurement, error) {
	measurements := []*Measurement{}

	/*
		FIXME
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
	*/

	return measurements, nil
}

// ListResults return the list of results
func ListResults(db sqlbuilder.Database) ([]*Result, []*Result, error) {
	doneResults := []*Result{}
	incompleteResults := []*Result{}

	/*
		FIXME
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
	*/

	/*
			FIXME
		rows, err := db.Query(`SELECT
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
	*/

	/*
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
	*/

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
