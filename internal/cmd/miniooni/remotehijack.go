package main

//
// Client-side connection hijacking implementation
//

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"gopkg.in/yaml.v3"
)

// remoteMaybeHijack hijacks miniooni connections using the selected remote
// name unless the remote name is empty.
func remoteMaybeHijack(options *Options) error {
	remoteName := options.RemoteName
	if remoteName == "" {
		return nil
	}

	// obtain the remote configuration from the config file
	cfg, err := remoteReadConfigFile(options)
	if err != nil {
		return fmt.Errorf("remote: cannot read config file: %w", err)
	}
	remote := cfg.Remotes[remoteName]
	if remote == nil {
		return fmt.Errorf("remote: %s: no such remote", remoteName)
	}

	// establish the specified remote connection
	var client *remoteClient
	switch txp := remote.Transport; txp {
	case "tcp":
		client, err = newRemoteTCPClient(remote)
	case "ssh":
		client, err = newRemoteSSHClient(remote)
	default:
		return fmt.Errorf("remote: %s: no such transport", txp)
	}
	if err != nil {
		return err
	}

	// start routing traffic
	go client.route()

	// hijack netxlite's fundamental network operations
	netxlite.TProxyDialWithDialer = client.DialWithDialer
	netxlite.TProxyGetaddrinfoLookupANY = client.GetaddrinfoLookupANY
	netxlite.TProxyListenUDP = client.ListenUDP
	log.Infof("remote: %s: hijacked netxlite network primitives", remoteName)

	return nil
}

// remoteConfigFile contains the configuration file content.
type remoteConfigFile struct {
	// Remotes maps a remote name to its settings.
	Remotes map[string]*remoteConfig `yaml:"remotes"`
}

// remoteConfig is the configuration of a specific remote.
type remoteConfig struct {
	// Address is the remote endpoint to use.
	Address string `yaml:"address"`

	// Transport is the transport to use.
	Transport string `yaml:"transport"`

	// SSH contains optional SSH configuration.
	SSH *remoteConfigSSH `yaml:"ssh"`
}

// remoteConfigSSH contains SSH specific configuration.
type remoteConfigSSH struct {
	// User is the user name to use
	User string `yaml:"user"`
}

// remoteReadConfigFile reads the remote config file.
func remoteReadConfigFile(options *Options) (*remoteConfigFile, error) {
	miniooniDir := createAndReturnMiniooniDir(options)
	filename := filepath.Join(miniooniDir, "remote", "config.yaml")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg remoteConfigFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
