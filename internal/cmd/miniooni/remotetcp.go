package main

//
// TCP remote implementation
//

import (
	"net"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

var (
	// remoteTCPPort is the port used by default by this remote.
	remoteTCPPort string

	// remoteTCPInterface is the interface used by default by this remote.
	remoteTCPInterface string
)

// registerRemoteTCP registers the remotetcp command.
func registerRemoteTCP(rootCmd *cobra.Command) {
	subCmd := &cobra.Command{
		Use:   "remotetcp",
		Short: "RemoteTCP protocol server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			remoteTCPServerMain(remoteTCPPort, remoteTCPInterface)
		},
	}

	flags := subCmd.Flags()

	flags.StringVar(
		&remoteTCPPort,
		"port",
		"5555",
		"selects the port to use",
	)

	flags.StringVar(
		&remoteTCPInterface,
		"interface",
		"eth0",
		"selects the interface to use",
	)

	rootCmd.AddCommand(subCmd)
}

// remoteTCPServerMain is the main of the remotetcp subcommand.
func remoteTCPServerMain(port, iface string) {
	config := &remoteServerConfig{
		iface: iface,
	}
	factory := &remoteListenerFactory{
		iface: iface,
		port:  port,
		wrapconn: func(conn net.Conn) (remoteConn, error) {
			return conn, nil
		},
	}
	err := remoteServerMain(config, factory)
	runtimex.PanicOnError(err, "remoteServerMain failed")
}

// newRemoteTCPClient creates a new remoteClient using TCP.
func newRemoteTCPClient(remote *remoteConfig) (*remoteClient, error) {
	dialer := &remoteDialer{
		remoteAddr: remote.Address,
		wrapConn: func(conn net.Conn) (remoteConn, error) {
			return conn, nil
		},
	}
	return newRemoteClient(dialer)
}
