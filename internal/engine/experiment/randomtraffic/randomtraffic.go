// Package randomtraffic contains the randomtraffic experiment.

package randomtraffic

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// testVersion is the experiment version.
const testVersion = "0.1.0"

// Config contains the experiment config.
type Config struct {
	// Target IP of the experiment
	Target string `ooni:"Target IP"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// True if no errors occurred 
	Success bool `json:"success"`

	// Number of successful connections after control test
	ConnectionCount int `json:"connection_count"`

	// The popcount of the triggering packet
	FinalPopcount float64 `json:"final_popcount"`

	// True if first six bytes of the final payload are printable
	FirstSix bool `json:"first_six"`

	// True if there exist twenty contiguous bytes of printable ASCII in the final payload
	TwentyContig bool `json:"twenty_contig"`

	// True if at least half of the final payload is made up of printable ASCII
	HalfPrintable bool `json:"half_printable"`

	// True if final popcount is less than 3.4 or greater than 4.6
	PopcountRange bool `json:"popcount_range"`

	// True if fingerprinted as HTTP
	MatchesHTTP bool `json:"matches_http"`

	// True if fingerprinted as TLS
	MatchesTLS bool `json:"matches_tls"`

	// Payload of final packet
	Payload []byte `json:"payload"`

	// False if all 20 connections succeeded
	Censorship bool `json:"censorship"`

	// String of error
	Error *string `json:"error"`
}

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return "randomtraffic"
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// Prediction stores the boolean value of each exemption rule
// (as specified in Algorithm 1) of the final packet sent
type Prediction struct {
	// True if first six bytes of the payload are printable
	FirstSix bool

	// True if there exist twenty contiguous bytes of printable ASCII in the payload
	TwentyContig bool

	// True if at least half of the payload is made up of printable ASCII
	HalfPrintable bool

	// True if popcount is less than 3.4 or greater than 4.6
	PopcountRange bool

	// True if fingerprinted as HTTP
	MatchesHTTP bool

	// True if fingerprinted as TLS
	MatchesTLS bool
}

// Sets defaults of prediction in the event of no payloads being sent
func defaultPrediction() Prediction {
	return Prediction{
		FirstSix:      false,
		TwentyContig:  false,
		HalfPrintable: false,
		PopcountRange: false,
		MatchesHTTP:   false,
		MatchesTLS:    false}
}

// Checks if a byte represents a printable ASCII character
func isPrintable(b byte) bool {
	return (b >= 0x20 && b <= 0x7e)
}

// Checks if all bytes in a byte array are printable ASCII characters
func allPrintable(bytes []byte) bool {
	for _, b := range bytes {
		if !isPrintable(b) {
			return false
		}
	}
	return true
}

// Counts the number of one bits there are in any given byte
func bitsInByte(b byte) int {
	count := 0
	var mask byte = 0x01
	for p := 1; p <= 8; p++ {
		// Isolate bit and see if zero
		if b&mask != 0x00 {
			count++
		}
		// Move mask
		mask = mask << 1
	}
	return count
}

// Returns the popcount of the given byte stream
func popcount(bytes []byte) float64 {
	bitCount := 0
	// Sum all one bits
	for _, b := range bytes {
		bitCount += bitsInByte(b)
	}
	// Calculate average bits per byte
	return (float64(bitCount) / float64(len(bytes)))
}

// Compares two byte array slices
func slicesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Returns true if bytes match HTTP fingerprint
func fingerprintHTTP(bytes []byte) bool {

	// Specific HTTP headers that are exempt from blocking
	GET := [4]byte{0x47, 0x45, 0x54, 0x20}
	PUT := [4]byte{0x50, 0x55, 0x54, 0x20}
	POST := [5]byte{0x50, 0x4F, 0x53, 0x54, 0x20}
	HEAD := [5]byte{0x48, 0x45, 0x41, 0x44, 0x20}

	// Check if byte array begins with one of the above and return
	return (slicesEqual(bytes[:4], GET[:]) ||
		slicesEqual(bytes[:4], PUT[:]) ||
		slicesEqual(bytes[:5], POST[:]) ||
		slicesEqual(bytes[:5], HEAD[:]))
}

// Returns true if bytes match TLS fingerprint
func fingerprintTLS(bytes []byte) bool {
	if bytes[0] == 0x16 || bytes[0] == 0x17 {
		return (bytes[1] == 0x03 && bytes[2] >= 0x00 && bytes[2] <= 0x09)
	} else {
		return false
	}
}

// Checks if popcount is allowed
func popcountTest(bytes []byte) bool {
	popcount := popcount(bytes)
	return (popcount > 4.6 && popcount < 3.4)
}

// Checks all three ASCII criteria and popcount that the censorship looks for
func getPrediction(bytes []byte) Prediction {

	// Initializing values
	firstSix := false
	twentyContig := false
	halfPrintable := false

	// Checks the first six bytes are printable ASCII characters
	if allPrintable(bytes[:6]) {
		firstSix = true
	}

	contig := 0
	total := 0
	for _, b := range bytes {
		if isPrintable(b) {
			// Keeping count of contiguous printable ASCII bytes
			contig = contig + 1
			// If there are twenty contiguous printable ASCII bytes the packet is not censored
			if contig >= 20 {
				twentyContig = true
			}
			// Count total printable ASCII bytes
			total = total + 1
		} else {
			// Reset contiguous count if not printable ASCII
			contig = 0
		}
	}
	// If more than 50% of the packet is printable ASCII the packet is not censored
	halfPrintable = (float64(total) / float64(len(bytes))) > .5

	// Remaining exemptions
	popcountRange := popcountTest(bytes)
	matchesHTTP := fingerprintHTTP(bytes)
	matchesTLS := fingerprintTLS(bytes)

	// Returning prediction
	return Prediction{
		FirstSix:      firstSix,
		TwentyContig:  twentyContig,
		HalfPrintable: halfPrintable,
		PopcountRange: popcountRange,
		MatchesHTTP:   matchesHTTP,
		MatchesTLS:    matchesTLS}
}

// maxRuntime is the maximum runtime for this experiment
const maxRuntime = 300 * time.Second

// Converts domain name to IP address
func domainToIP(hostname string, ctx context.Context, sess model.ExperimentSession) {
	reso := netxlite.NewStdlibResolver(sess.Logger())

	addrs, err := reso.LookupHost(ctx, "dns.google")
	if err != nil {
		handle(err, false, sess)
	}
	sess.Logger().Infof("resolver addrs: %+v", addrs)
}

const maxTests = 20

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	// Creating timeout context
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()

	// Running control
	var err error = nil
	var timeout bool = false
	var residual bool = false
	var payload []byte = nil

	var controlAttempts int = 3

	// We count in 22th's because 22 messages will be logged
	// One for the control, one each of the 20 tests, and one after all is cleaned up
	callbacks.OnProgress(1.0/(float64(maxTests)+2.0), "Initializing IP")

	if m.config.Target != "" {
		// If IP provided, run control on that IP
		sess.Logger().Infof("Running control on %s", m.config.Target)
		timeout, err = testIP(m.config.Target, true, &payload, ctx, sess)
		if err != nil {
			// If control test fails inform and carry error
			sess.Logger().Warnf("%s", "Control test failed. This may be due to residual censorship. Try again in 3-5 minutes.")
		} else {
			// If control test successful continue with test
			sess.Logger().Infof("Control test successful")

			// Logging IP and port
			sess.Logger().Infof("Running tests on %s", m.config.Target)
		}
	} else {
		// If no IP given, continue testing while attempts remain until we find a valid IP
		checkForIP := true
		for checkForIP == true && controlAttempts > 0 {
			// Fetching IP address
			m.config.Target = GetIP()

			// Beginning control test
			sess.Logger().Infof("Running control on %s", m.config.Target)
			timeout, err = testIP(m.config.Target, true, &payload, ctx, sess)

			// Decrementing attempts left
			controlAttempts = controlAttempts - 1
			if err != nil {
				if controlAttempts > 0 {
					// If failed with more attempts, inform user
					sess.Logger().Infof("Control test failed. Fetching new IP")
				} else {
					// If failed with no attempts, inform user and end search
					sess.Logger().Infof("Failed to connect at this time")
					checkForIP = false
				}
			} else {
				// If control test successful log and end search
				sess.Logger().Infof("Control test successful")

				// Logging IP and port
				sess.Logger().Infof("Running tests on %s", m.config.Target)
				checkForIP = false
			}
		}
	}

	// Running tests
	testCount := 1

	// If there is censorship detected or an experiment error, end the experiment
	// Note: This will not run if the control test fails
	for testCount <= maxTests && err == nil && residual == false {

		// Informing client, running test, and looping to next test
		callbacks.OnProgress(float64(testCount+1)/(float64(maxTests)+2.0), "Running test "+strconv.Itoa(testCount))
		timeout, err = testIP(m.config.Target, false, &payload, ctx, sess)

		// If timeout, it should timeout again due to residual censorship. This tests that
		if timeout == true {

			// Logging and running test
			sess.Logger().Infof("Checking residual")
			residual, err = testIP(m.config.Target, false, &payload, ctx, sess)

			// If not residual, log and continue testing
			if residual == false {
				sess.Logger().Infof("Connection successful")
				timeout = false
			}

		}

		// Incrementing test count
		testCount = testCount + 1
	}

	testCount = testCount - 1
	callbacks.OnProgress(1, "Done.")

	// Configuring testkeys
	measurement.TestKeys = configureTestKeys(testCount, timeout, residual, payload, err, sess.Logger())

	// Return error (nil if no error)
	return err
}

// This function establishes and returns a TCP connection between the client and a target IP address
func dialTCP(ctx context.Context, address string, logger model.DebugLogger) (net.Conn, error) {
	d := netxlite.NewDialerWithoutResolver(logger)
	return d.DialContext(ctx, "tcp", address)
}

// This function handles errors as a result of establishing a connection
// If handling a control test error, any error including a timeouts will be returned
func handle(err error, control bool, sess model.ExperimentSession) (bool, error) {
	var ew *netxlite.ErrWrapper
	if !errors.As(err, &ew) {
		// Experiment error, no timeout
		return false, errors.New("Error not wrapped")
	}

	// If timeout, log and return
	if err.Error() == netxlite.FailureGenericTimeoutError {
		sess.Logger().Infof("Connection timed out")
		if control {
			return true, err
		} else {
			return true, nil
		}
	}

	// Otherwise, log error and return the error
	sess.Logger().Warnf("error string    : %s", err.Error())
	sess.Logger().Warnf("OONI failure    : %s", ew.Failure)
	sess.Logger().Warnf("failed operation: %s", ew.Operation)
	sess.Logger().Warnf("underlying error: %+v", ew.WrappedErr)

	// Experiment error and no timeout returned
	return false, err
}

// This function runs a single test on an IP by establishing a connection and if successful
// then sending random data through the connection as an attempt to trigger the censorship

// Note: The control test does NOT expect timeout errors and will return an error
func testIP(
	IP string, control bool, payload *[]byte,
	ctx context.Context, sess model.ExperimentSession,
) (bool, error) {

	// Connecting to target IP address
	conn, err := dialTCP(ctx, IP, sess.Logger())

	// Handling error
	if err != nil {
		sess.Logger().Warnf("%s", "Failed to connect")
		return handle(err, control, sess)
	}
	defer conn.Close()

	// The average payload length is between 500 and 1000. This creates
	// an array of bytes in that range, all of which are zero
	bytes := make([]byte, rand.Intn(500)+500)

	// If this is not the control test then make the payload random
	if !control {
		// Filling byte array with random bytes
		rand.Read(bytes)
	}

	// Record payload
	*payload = bytes

	// Bytes are sent through the connection
	conn.Write(bytes)

	// No timeout and no experiment error
	return false, nil
}

func configureTestKeys(
	testCount int, timeout bool, residual bool,
	payload []byte, err error, logger model.InfoLogger,
) *TestKeys {
	testkeys := &TestKeys{}

	// If there is no error the test was successfully conducted
	testkeys.Success = (err == nil)

	// If the connection timed out and residual censorship was observed
	// then censorship is logged
	testkeys.Censorship = (timeout == true && residual == true)

	// If there is an error, save its string in the test keys
	if err != nil {
		errString := err.Error()
		testkeys.Error = &errString
	} else {
		testkeys.Error = nil
	}

	// Amount of successful connections is the test count
	// one is subtracted due to the one failed connection
	testkeys.ConnectionCount = testCount - 1

	// Using payload to set both Payload and FinalPopcount testkeys
	testkeys.Payload = payload
	testkeys.FinalPopcount = popcount(payload)

	// Setting prediction using payload
	prediction := defaultPrediction()
	if err == nil {
		prediction = getPrediction(payload)
	}

	// Setting remaining testkeys
	testkeys.FirstSix = prediction.FirstSix
	testkeys.TwentyContig = prediction.TwentyContig
	testkeys.HalfPrintable = prediction.HalfPrintable
	testkeys.PopcountRange = prediction.PopcountRange
	testkeys.MatchesHTTP = prediction.MatchesHTTP
	testkeys.MatchesTLS = prediction.MatchesTLS

	// Logging Results
	logger.Infof("Success: %s", strconv.FormatBool(testkeys.Success))
	logger.Infof("Censorship: %s", strconv.FormatBool(timeout == true && residual == true))

	return testkeys
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
