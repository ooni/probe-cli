package ooshell

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

// DB is the database that keeps tracks of events.
type DB interface {
	// NewSession creates a new session for running measurements.
	//
	// On success, this function returns a row ID and a nil error,
	// while on failure it returns a non-nil error.
	NewSession() (int64, error)

	// SetSessionBootstrapResult sets the results of the bootstrap.
	//
	// Arguments:
	//
	// - id is the result of a previous NewSession;
	//
	// - err is the bootstrap result (which may be nil).
	//
	// Returns nil on success, an error on failure.
	SetSessionBootstrapResult(id int64, err error) error

	// NewNetwork creates a new network in which we run a measurement.
	//
	// Arguments:
	//
	// - sessionID is the result of a previous NewSession;
	//
	// - IP is the probe's IP address;
	//
	// - ASN is the IP's ASN;
	//
	// - networkName is the ASN's name;
	//
	// - countryCode is the IP's country code.
	//
	// On success, this function returns a row ID and a nil error,
	// while on failure it returns a non-nil error.
	NewNetwork(sessionID int64,
		IP string, ASN uint, networkName, countryCode string) (int64, error)

	// NewResult creates a new result of a run.
	//
	// Arguments:
	//
	// - networkID is the result of a previous NewNetwork.
	//
	// - groupName is the name of the experiment group (e.g., "circumvention");
	//
	// On success, this function returns a row ID and a nil error,
	// while on failure it returns a non-nil error.
	NewResult(networkID int64, groupName string) (int64, error)

	// SetResultFinished marks a given result as finished.
	SetResultFinished(resultID int64) error

	// NewExperiment creates a new experiment run.
	//
	// Arguments:
	//
	// - experimentName is the name of the experiment (e.g., "ndt");
	//
	// - resultID is a result of a previous NewResult.
	//
	// On success, this function returns a row ID and a nil error,
	// while on failure it returns a non-nil error.
	NewExperiment(resultID int64, experimentName string) (int64, error)

	// UpdateExperimentProgress updates on the experiment progress.
	UpdateExperimentProgress(experimentID int64, percentage float64, message string)

	// SetExperimentFinished sets an experiment as finished.
	SetExperimentFinished(experimentID int64, err error) error
}

// NewLoggerDB creates a new DB that only logs.
func NewLoggerDB(logger Logger) DB {
	return &loggerDB{
		experimentID: &atomicx.Int64{},
		logger:       logger,
		networkID:    &atomicx.Int64{},
		sessionID:    &atomicx.Int64{},
		resultID:     &atomicx.Int64{},
	}
}

// loggerDB is a DB that only logs.
type loggerDB struct {
	experimentID *atomicx.Int64
	logger       Logger
	networkID    *atomicx.Int64
	sessionID    *atomicx.Int64
	resultID     *atomicx.Int64
}

func (db *loggerDB) NewSession() (int64, error) {
	sessionID := db.sessionID.Add(1)
	db.logger.Infof("current time: %s", time.Now().Format("2006-01-02 15:04:05 MST"))
	db.logger.Infof("boostrapping session#%d... in progress", sessionID)
	return sessionID, nil
}

func (db *loggerDB) SetSessionBootstrapResult(id int64, err error) error {
	db.logger.Infof("boostrapping session#%d... %+v", id, db.asFailureString(err))
	return nil
}

func (db *loggerDB) NewNetwork(sessionID int64,
	IP string, ASN uint, networkName, countryCode string) (int64, error) {
	networkID := db.networkID.Add(1)
	db.logger.Infof("network#%d sess=%d ip=%s asn=%d name=%s cc=%s",
		networkID, sessionID, IP, ASN, networkName, countryCode)
	return networkID, nil
}

func (db *loggerDB) NewResult(networkID int64, group string) (int64, error) {
	resultID := db.resultID.Add(1)
	db.logger.Infof("result#%d group=%s network=%d", resultID, group, networkID)
	return resultID, nil
}

func (db *loggerDB) SetResultFinished(resultID int64) error {
	db.logger.Infof("marking result#%d as complete", resultID)
	return nil
}

func (db *loggerDB) NewExperiment(resultID int64, experimentName string) (int64, error) {
	experimentID := db.experimentID.Add(1)
	db.logger.Infof(
		"experiment#%d name=%s result=%d", experimentID, experimentName, resultID)
	return experimentID, nil
}

func (db *loggerDB) UpdateExperimentProgress(
	experimentID int64, percentage float64, message string) {
	db.logger.Infof("[%5.1f%%] %s", percentage*100, message)
}

func (db *loggerDB) SetExperimentFinished(experimentID int64, err error) error {
	db.logger.Infof(
		"experiment#%d finished failure=%s", experimentID, db.asFailureString(err))
	return nil
}

func (db *loggerDB) asFailureString(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
