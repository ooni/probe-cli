package nettests

// Tor test implementation
type Tor struct {
}

// Run starts the test
func (h Tor) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"tor",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// TorTestKeys contains the test keys
type TorTestKeys struct {
	DirPortTotal            int64 `json:"dir_port_total"`
	DirPortAccessible       int64 `json:"dir_port_accessible"`
	IsAnomaly               bool  `json:"-"`
	OBFS4Total              int64 `json:"obfs4_total"`
	OBFS4Accessible         int64 `json:"obfs4_accessible"`
	ORPortDirauthTotal      int64 `json:"or_port_dirauth_total"`
	ORPortDirauthAccessible int64 `json:"or_port_dirauth_accessible"`
	ORPortTotal             int64 `json:"or_port_total"`
	ORPortAccessible        int64 `json:"or_port_accessible"`
}

// GetTestKeys generates a summary for a test run
func (h Tor) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	testKeys := TorTestKeys{IsAnomaly: false}
	// Implementation note: when Go marshals into an interface, it marshals to
	// float64 rather than int64, so we need to do some more work here.
	//
	// See <https://golang.org/pkg/encoding/json/#Unmarshal>.
	if tk["dir_port_total"] != nil {
		testKeys.DirPortTotal = int64(tk["dir_port_total"].(float64))
	}
	if tk["dir_port_accessible"] != nil {
		testKeys.DirPortAccessible = int64(tk["dir_port_accessible"].(float64))
	}
	if tk["obfs4_total"] != nil {
		testKeys.OBFS4Total = int64(tk["obfs4_total"].(float64))
	}
	if tk["obfs4_accessible"] != nil {
		testKeys.OBFS4Accessible = int64(tk["obfs4_accessible"].(float64))
	}
	if tk["or_port_dirauth_total"] != nil {
		testKeys.ORPortDirauthTotal = int64(tk["or_port_dirauth_total"].(float64))
	}
	if tk["or_port_dirauth_accessible"] != nil {
		testKeys.ORPortDirauthAccessible = int64(tk["or_port_dirauth_accessible"].(float64))
	}
	if tk["or_port_total"] != nil {
		testKeys.ORPortTotal = int64(tk["or_port_total"].(float64))
	}
	if tk["or_port_accessible"] != nil {
		testKeys.ORPortAccessible = int64(tk["or_port_accessible"].(float64))
	}
	testKeys.IsAnomaly = ((testKeys.DirPortAccessible <= 0 && testKeys.DirPortTotal > 0) ||
		(testKeys.OBFS4Accessible <= 0 && testKeys.OBFS4Total > 0) ||
		(testKeys.ORPortDirauthAccessible <= 0 && testKeys.ORPortDirauthTotal > 0) ||
		(testKeys.ORPortAccessible <= 0 && testKeys.ORPortTotal > 0))
	return testKeys, nil
}

// LogSummary writes the summary to the standard output
func (h Tor) LogSummary(s string) error {
	return nil
}
