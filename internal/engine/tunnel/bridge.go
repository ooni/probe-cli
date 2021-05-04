package tunnel

import (
	"errors"
)

// BridgeInfo provides information about a bridge.
type BridgeInfo interface {
	// AsBridgeArgument returns the argument for tor's Bridge
	// command line option.
	AsBridgeArgument() string

	// AsClientTransportPluginArgument returns the argument for
	// tor's ClientTransportPlugin command line option.
	AsClientTransportPluginArgument() string
}

// Bridge is a tor bridge managed by OONI Probe. We manage
// two kind of bridges: obfs4 and snowflake.
type Bridge interface {
	// Stop is stops the running-in-background bridge. This
	// method is idempotent.
	Stop()

	// BridgeInfo is the base interface.
	BridgeInfo
}

// NewBridges creates instances of Bridge given a set of
// bridge lines passed as argument. This function will
// return a set of running bridges or an error. It returns
// an error in case the input slice is empty.
func NewBridges(lines []string, datadir string) ([]Bridge, error) {
	var (
		bridges []Bridge
		success bool
	)
	defer func() {
		if !success {
			for _, b := range bridges {
				b.Stop()
			}
		}
	}()
	for _, line := range lines {
		// check whether this is an obfs4 bridge line
		blp := &OBFS4BridgeLineParser{
			BridgeLine: line,
			DataDir:    datadir,
		}
		br, err := blp.Parse()
		if err == nil {
			if err := br.Start(); err != nil {
				return nil, err
			}
			bridges = append(bridges, br)
			continue
		}
		// if a line is not handled by any bridge type, then pass
		// such a line directly to the tor process
		bridges = append(bridges, &unknownBridge{line: line})
	}
	if len(bridges) < 1 {
		return nil, errors.New("tunnel: no bridges created")
	}
	success = true
	return bridges, nil
}

// unknownBridge is a bridge whose protocol we don't know.
type unknownBridge struct {
	line string
}

// AsBridgeArgument implements Bridge.AsBridgeArgument.
func (ub *unknownBridge) AsBridgeArgument() string {
	return ub.line
}

// AsClientTransportPluginArgument implements
// Bridge.AsClientTransportPluginArgument.
func (ub *unknownBridge) AsClientTransportPluginArgument() string {
	return "" // no argument will be added to cmdline
}

// Stop implements Bridge.Stop
func (ub *unknownBridge) Stop() {}
