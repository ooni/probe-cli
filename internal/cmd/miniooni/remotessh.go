package main

//
// SSH remote implementation
//

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var (
	// remoteSSHPort is the port used by default by this remote.
	remoteSSHPort string

	// remoteSSHInterface is the interface used by default by this remote.
	remoteSSHInterface string
)

// registerRemoteSSH registers the remotessh command.
func registerRemoteSSH(rootCmd *cobra.Command) {
	subCmd := &cobra.Command{
		Use:   "remotessh",
		Short: "RemoteSSH protocol server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			remoteSSHServerMain(remoteSSHPort, remoteSSHInterface)
		},
	}

	flags := subCmd.Flags()

	flags.StringVar(
		&remoteSSHPort,
		"port",
		"2222",
		"selects the port to use",
	)

	flags.StringVar(
		&remoteSSHInterface,
		"interface",
		"eth0",
		"selects the interface to use",
	)

	rootCmd.AddCommand(subCmd)
}

// remoteSSHServerMain is the main of the remotessh subcommand.
func remoteSSHServerMain(port, iface string) {
	config := &remoteServerConfig{
		iface: iface,
	}
	sh, err := newRemoteSSHServerHandler()
	runtimex.PanicOnError(err, "newRemoteSSHServerHandler failed")
	factory := &remoteListenerFactory{
		iface:    iface,
		port:     port,
		wrapconn: sh.wrapConn,
	}
	err = remoteServerMain(config, factory)
	runtimex.PanicOnError(err, "remoteServerMain failed")
}

// remoteSSHServerHandler handles incoming SSH conns.
type remoteSSHServerHandler struct {
	config *ssh.ServerConfig
}

// remoteSSHReadAuthorizedKeys reads and parses the authorized_keys file.
func remoteSSHReadAuthorizedKeys() (map[string]bool, error) {
	homeDir := gethomedir("")
	filename := filepath.Join(homeDir, ".ssh", "authorized_keys")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	akmap := map[string]bool{}
	for len(data) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			return nil, err
		}
		akmap[string(pubKey.Marshal())] = true
		data = rest
	}
	return akmap, nil
}

// remoteSSHReadSSHHostRSAKey reads the host's private key.
func remoteSSHReadSSHHostRSAKey() (ssh.Signer, error) {
	data, err := os.ReadFile("/etc/ssh/ssh_host_rsa_key")
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}

// newRemoteSSHServerHandler creates a new remoteSSHServerHandler instance.
func newRemoteSSHServerHandler() (*remoteSSHServerHandler, error) {
	akmap, err := remoteSSHReadAuthorizedKeys()
	if err != nil {
		return nil, err
	}
	signer, err := remoteSSHReadSSHHostRSAKey()
	if err != nil {
		return nil, err
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if akmap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(signer)
	handler := &remoteSSHServerHandler{
		config: config,
	}
	return handler, nil
}

// errRemoteSSHInvalidChannelType indicates that the channel type is invalid
var errRemoteSSHInvalidChannelType = errors.New("invalid SSH channel type")

// wrapConn wraps a server-side net.Conn to be an SSH conn.
func (h *remoteSSHServerHandler) wrapConn(conn net.Conn) (remoteConn, error) {
	sshConn, chans, sshReqs, err := ssh.NewServerConn(conn, h.config)
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(sshReqs)
	candidate := <-chans
	if candidate.ChannelType() != "miniooni-remote" {
		return nil, errRemoteSSHInvalidChannelType
	}
	channel, chanReqs, err := candidate.Accept()
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(chanReqs)
	rc := &remoteSSHServerRemoteConn{
		channel:   channel,
		closeOnce: &sync.Once{},
		conn:      sshConn,
	}
	return rc, nil
}

// remoteSSHServerRemoteConn implements remoteConn
type remoteSSHServerRemoteConn struct {
	channel   ssh.Channel
	closeOnce *sync.Once
	conn      *ssh.ServerConn
}

var _ remoteConn = &remoteSSHServerRemoteConn{}

func (c *remoteSSHServerRemoteConn) Read(data []byte) (int, error) {
	return c.channel.Read(data)
}

func (c *remoteSSHServerRemoteConn) Write(data []byte) (int, error) {
	return c.channel.Write(data)
}

func (c *remoteSSHServerRemoteConn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		if e := c.conn.Close(); e != nil {
			err = e
		}
	})
	return err
}

// newRemoteSSHClient creates a new remoteClient using SSH.
func newRemoteSSHClient(remote *remoteConfig) (*remoteClient, error) {
	hx, err := newRemoteSSHClientHandshaker(remote)
	if err != nil {
		return nil, err
	}
	dialer := &remoteDialer{
		remoteAddr: remote.Address,
		wrapConn:   hx.wrapConn,
	}
	return newRemoteClient(dialer)
}

// remoteSSHClientHandshaker performs the SSH handshake and returns
// a suitable connection for forwarding traffic.
type remoteSSHClientHandshaker struct {
	config *ssh.ClientConfig
}

// errRemoteSSHMissingConfig indicates SSH specific config is missing.
var errRemoteSSHMissingConfig = errors.New("SSH specific config is missing")

// newRemoteSSHClientHandshaker creates a new remoteSSHClientHandshaker.
func newRemoteSSHClientHandshaker(remote *remoteConfig) (*remoteSSHClientHandshaker, error) {
	if remote.SSH == nil {
		return nil, errRemoteSSHMissingConfig
	}
	agentClient, err := remoteSSHClientCreateSSHAgent()
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: remote.SSH.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agentClient.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	hx := &remoteSSHClientHandshaker{
		config: config,
	}
	return hx, nil
}

// wrapConn wraps a server-side net.Conn to be an SSH conn.
func (hx *remoteSSHClientHandshaker) wrapConn(conn net.Conn) (remoteConn, error) {
	sshConn, _, sshReqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), hx.config)
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(sshReqs)
	channel, chanReqs, err := sshConn.OpenChannel("miniooni-remote", nil)
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(chanReqs)
	rc := &remoteSSHClientRemoteConn{
		channel:   channel,
		closeOnce: &sync.Once{},
		conn:      sshConn,
	}
	return rc, nil
}

// remoteSSHClientRemoteConn implements remoteConn
type remoteSSHClientRemoteConn struct {
	channel   ssh.Channel
	closeOnce *sync.Once
	conn      ssh.Conn
}

var _ remoteConn = &remoteSSHClientRemoteConn{}

func (c *remoteSSHClientRemoteConn) Read(data []byte) (int, error) {
	return c.channel.Read(data)
}

func (c *remoteSSHClientRemoteConn) Write(data []byte) (int, error) {
	return c.channel.Write(data)
}

func (c *remoteSSHClientRemoteConn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		if e := c.conn.Close(); e != nil {
			err = e
		}
	})
	return err
}

// errRemoteSSHNoAuthSock indicates that there is no SSH_AUTH_SOCK variable
var errRemoteSSHNoAuthSock = errors.New("no SSH_AUTH_SOCK environment variable")

// remoteSSHClientCreateSSHAgent creates a SSH agent instance.
func remoteSSHClientCreateSSHAgent() (agent.ExtendedAgent, error) {
	socket, found := os.LookupEnv("SSH_AUTH_SOCK")
	if !found {
		return nil, errRemoteSSHNoAuthSock
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}
	agentClient := agent.NewClient(conn)
	return agentClient, nil
}
