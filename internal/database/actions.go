package database

import (
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
func CreateMeasurement(sess sqlbuilder.Database, m Measurement, i string) (*Measurement, error) {
	col := sess.Collection("measurements")

	// XXX Do we want to have this be part of something else?
	m.StartTime = time.Now().UTC()

	// XXX insert also the URL and stuff
	//m.Input = i
	//m.State = "active"

	newID, err := col.Insert(m)
	if err != nil {
		return nil, errors.Wrap(err, "creating measurement")
	}
	m.ID = newID.(int64)
	return &m, nil
}

// CreateResult writes the Result to the database a returns a pointer
// to the Result
func CreateResult(sess sqlbuilder.Database, homePath string, r Result) (*Result, error) {
	log.Debugf("Creating result %v", r)

	col := sess.Collection("results")

	p, err := utils.MakeResultsDir(homePath, r.TestGroupName, r.StartTime)
	if err != nil {
		return nil, err
	}
	r.MeasurementDir = p
	newID, err := col.Insert(r)
	if err != nil {
		return nil, errors.Wrap(err, "creating result")
	}
	r.ID = newID.(int64)
	return &r, nil
}
