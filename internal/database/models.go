package database

import "time"

// Measurement model
type Measurement struct {
	ID             int       `db:"id"`
	Name           string    `db:"name"`
	StartTime      time.Time `db:"startTime"`
	EndTime        time.Time `db:"endTime"`
	Summary        string    `db:"summary"` // XXX this should be JSON
	ASN            int       `db:"asn"`
	IP             string    `db:"ip"`
	CountryCode    string    `db:"country"`
	State          string    `db:"state"`
	Failure        string    `db:"failure"`
	ReportFilePath string    `db:"reportFile"`
	ReportID       string    `db:"reportId"`
	Input          string    `db:"input"`
	MeasurementID  string    `db:"measurementId"`
	ResultID       string    `db:"resultId"`
}

// Result model
type Result struct {
	ID            int       `db:"id"`
	Name          int       `db:"name"`
	StartTime     time.Time `db:"startTime"`
	EndTime       time.Time `db:"endTime"`
	Summary       string    `db:"summary"` // XXX this should be JSON
	Done          bool      `db:"done"`
	DataUsageUp   int       `db:"dataUsageUp"`
	DataUsageDown int       `db:"dataUsageDown"`
}
