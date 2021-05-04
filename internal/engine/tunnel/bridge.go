package tunnel

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
)

// ErrParseTorArgs indicates that we cannot parse the config.TorArgs field.
var ErrParseTorArgs = errors.New("tunnel: cannot parse TorArgs")

// torCmdlineArgs contains the parsed config.TorArgs.
type torCmdlineArgs struct {
	// normalArgs contains all the arguments that
	// do not have the "OONIBridge" key.
	normalArgs []string

	// ooniBridges only contains the arguments that
	// had a key equal to "OONIBridge".
	ooniBridges []string
}

// splitTorCmdlineArgs separates the config.TorArgs into normal
// arguments and arguments with key equal to "OONIBridge".
func splitTorCmdlineArgs(config *Config) (*torCmdlineArgs, error) {
	idx, size := 0, len(config.TorArgs)
	out := &torCmdlineArgs{}
	for {
		if idx >= size {
			break
		}
		key := config.TorArgs[idx]
		idx++
		if key != "OONIBridge" {
			out.normalArgs = append(out.normalArgs, key)
			continue
		}
		if idx >= size {
			return nil, fmt.Errorf("%w: missing value for %s", ErrParseTorArgs, key)
		}
		value := config.TorArgs[idx]
		out.ooniBridges = append(out.ooniBridges, value)
		idx++
	}
	return out, nil
}

// stoppableBridge is a stoppable bridge.
type stoppableBridge interface {
	// Stop stops the running bridge.
	Stop()
}

// managedBridge is a bridge that we manage directly from inside OONI.
type managedBridge struct {
	// stoppableBridge is a bridge we can stop.
	stoppableBridge

	// extraArgs contains the extra arguments to be passed to tor.
	extraArgs []string
}

// bridgeStarter starts bridges.
type bridgeStarter struct {
	// bridgeline is the bridge line.
	bridgeline string

	// logger is the logger to use.
	logger Logger

	// statedir is the directory where to store state.
	statedir string
}

// start starts the bridge using the given bridge line.
func (bs *bridgeStarter) start(ctx context.Context) (*managedBridge, error) {
	vals, err := shlex.Split(bs.bridgeline)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseTorArgs, err.Error())
	}
	if len(vals) < 1 {
		return nil, fmt.Errorf("%w: no tokens in bridge line", ErrParseTorArgs)
	}
	switch vals[0] {
	case "snowflake":
		if len(vals) != 3 {
			return nil, fmt.Errorf(
				"%w: expected three tokens for snowflake line", ErrParseTorArgs)
		}
		return bs.snowflake(ctx, vals[1], vals[2])
	case "obfs4":
		if len(vals) != 5 {
			return nil, fmt.Errorf(
				"%w: expected five tokens for obfs4 line", ErrParseTorArgs)
		}
		return bs.obfs4(ctx, vals[1], vals[2], vals[3:])
	default:
		return nil, fmt.Errorf("%w: unsupported bridge: %s", ErrParseTorArgs, vals[0])
	}
}

// snowflake starts a snowflake bridge.
func (bs *bridgeStarter) snowflake(
	ctx context.Context, address, fingerprint string) (*managedBridge, error) {
	sfk := &Snowflake{
		BrokerURL:        "https://snowflake-broker.torproject.net.global.prod.fastly.net/",
		Capacity:         3,
		FrontDomain:      "cdn.sstatic.net",
		ICEServersCommas: "stun:stun.voip.blackberry.com:3478,stun:stun.altar.com.pl:3478,stun:stun.antisip.com:3478,stun:stun.bluesip.net:3478,stun:stun.dus.net:3478,stun:stun.epygi.com:3478,stun:stun.sonetel.com:3478,stun:stun.sonetel.net:3478,stun:stun.stunprotocol.org:3478,stun:stun.uls.co.za:3478,stun:stun.voipgate.com:3478,stun:stun.voys.nl:3478",
		Logger:           bs.logger,
	}
	if err := sfk.Start(ctx); err != nil {
		return nil, err
	}
	return &managedBridge{
		stoppableBridge: sfk,
		extraArgs: []string{
			"Bridge", fmt.Sprintf("snowflake %s %s", address, fingerprint)},
	}, nil
}

// obfs4 starts an obfs4 bridge.
func (bs *bridgeStarter) obfs4(
	ctx context.Context, address, fingerprint string,
	options []string) (*managedBridge, error) {
	var (
		cert    string
		iatMode string
	)
	for _, v := range options {
		if strings.HasPrefix("cert=", v) {
			cert = v[len("cert="):]
			continue
		}
		if strings.HasPrefix("iat-mode", v) {
			iatMode = v[len("iat-mode="):]
			continue
		}
	}
	obfs := &OBFS4{
		Address: address,
		Cert:    cert,
		DataDir: filepath.Join(bs.statedir, "obfs4"),
		IATMode: iatMode,
		Logger:  bs.logger,
	}
	if err := obfs.Start(ctx); err != nil {
		return nil, err
	}
	return &managedBridge{
		stoppableBridge: obfs,
		extraArgs: []string{
			"Bridge", fmt.Sprintf(
				"obfs4 %s %s cert=%s iat-mode=%s", address, fingerprint, cert, iatMode)},
	}, nil
}
