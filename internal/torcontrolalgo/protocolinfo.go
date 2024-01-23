package torcontrolalgo

//
// protocolinfo.go - implements the PROTOCOLINFO command.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"errors"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// ProtocolInfoResponse is the response returned by the PROTOCOLINFO command.
type ProtocolInfoResponse struct {
	// AuthMethods contains the valid auth methods.
	AuthMethods []string

	// CookieFile is the path to the cookie file.
	CookieFile string

	// TorVersion is the version of the tor we're using.
	TorVersion string
}

// ProtocolInfo sends a PROTOCOLINFO request and returns the parsed response.
func ProtocolInfo(ctx context.Context, conn Conn) (*ProtocolInfoResponse, error) {
	// send PROTOCOLINFO and receive response
	resp, err := conn.SendRecv(ctx, "PROTOCOLINFO")
	if err != nil {
		return nil, err
	}

	// construct ProtocolInfo
	return NewProtocolInfo(resp)
}

// HasAuthMethod checks if ProtocolInfoResponse contains the requested auth method.
func (p *ProtocolInfoResponse) HasAuthMethod(authMethod string) bool {
	for _, m := range p.AuthMethods {
		if m == authMethod {
			return true
		}
	}
	return false
}

// ErrRequestFailed indicates that a given [*torcontrolnet.Response] is not successful.
var ErrRequestFailed = errors.New("torcontrol: request failed")

// ErrInvalidProtocolVersion indicates that the protocol version is not valid.
var ErrInvalidProtocolVersion = errors.New("torcontrol: invalid protocol version")

// NewProtocolInfo constructs a [*ProtocolInfoResponse] from a [*Response].
func NewProtocolInfo(resp *torcontrolnet.Response) (*ProtocolInfoResponse, error) {
	// make sure the response is successful
	if resp.Status != torcontrolnet.StatusOk {
		return nil, ErrRequestFailed
	}

	// initialize the response
	pinfo := &ProtocolInfoResponse{
		AuthMethods: []string{},
		CookieFile:  "",
		TorVersion:  "",
	}

	parser := map[string]func(value string) error{
		"PROTOCOLINFO": pinfo.onProtocolInfo,
		"AUTH":         pinfo.onAuth,
		"VERSION":      pinfo.onVersion,
	}

	// Process each line containining data
	for _, entry := range resp.Data {
		// the entries we recognize are all like <KEY> <SP> <VALUE>
		key, value, ok := partitionString(entry, ' ')
		if !ok {
			continue
		}
		fx := parser[key]
		if fx == nil {
			continue // be liberal and allow for future extensions w/o breaking
		}
		if err := fx(value); err != nil {
			return nil, err
		}
	}

	return pinfo, nil
}

func (p *ProtocolInfoResponse) onProtocolInfo(value string) error {
	if value != "1" {
		return ErrInvalidProtocolVersion
	}
	return nil
}

// ErrMissingMethodsPrefix indicates the tor version is missing the 'METHODS' prefix.
var ErrMissingMethodsPrefix = errors.New("torcontrol: missing METHODS= prefix")

// ErrMissingCookiefilePrefix indicates the tor version is missing the 'COOKIEFILE' prefix.
var ErrMissingCookiefilePrefix = errors.New("torcontrol: missing COOKIEFILE= prefix")

func (p *ProtocolInfoResponse) onAuth(value string) error {
	// This is the format:
	//
	//	AuthLine = "250-AUTH" SP "METHODS=" AuthMethod *("," AuthMethod)
	//		*(SP "COOKIEFILE=" AuthCookieFile) CRLF
	//
	// The COOKIEFILE is optional.
	methods, cookieFile, _ := partitionString(value, ' ')

	// Make sure there's the METHODS= prefix
	if !strings.HasPrefix(methods, "METHODS=") {
		return ErrMissingMethodsPrefix
	}

	// Register the auth methods
	p.AuthMethods = strings.Split(methods[8:], ",")

	// Handle the optional COOKIEFILE
	if cookieFile == "" {
		return nil
	}
	if !strings.HasPrefix(cookieFile, "COOKIEFILE=") {
		return ErrMissingCookiefilePrefix
	}
	cookieFile, err := unescapeSimpleQuotedString(cookieFile[11:])
	if err != nil {
		return err
	}
	p.CookieFile = cookieFile
	return nil
}

// ErrMissingTorPrefix indicates the tor version is missing the 'Tor=' prefix.
var ErrMissingTorPrefix = errors.New("torcontrol: missing Tor prefix")

func (p *ProtocolInfoResponse) onVersion(value string) error {
	// The format is the following
	//
	//	VersionLine = "250-VERSION" SP "Tor=" TorVersion OptArguments CRLF
	//
	// We need to take into account the case in which there are no
	// optional arguments before the end of line.
	torVersion, _, _ := partitionString(value, ' ')

	// make sure there is the 'Tor=' prefix
	if !strings.HasPrefix(torVersion, "Tor=") {
		return ErrMissingTorPrefix
	}

	// unescape the version
	version, err := unescapeSimpleQuotedString(torVersion[4:])
	if err != nil {
		return err
	}

	// success!
	p.TorVersion = version
	return nil
}
