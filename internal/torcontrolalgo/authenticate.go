package torcontrolalgo

//
// authenticate.go - implements the AUTHENTICATE command.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// AuthenticateFlowWithSafeCookie runs the full authenticate flow assuming that we
// asked tor to create a cookie file and we want to use SAFECOOKIE.
func AuthenticateFlowWithSafeCookie(ctx context.Context, conn Conn) error {
	// obtain protocol info
	pinfo, err := ProtocolInfo(ctx, conn)
	if err != nil {
		return err
	}

	// challenge the tor server
	challenge, err := AuthChallenge(ctx, conn, pinfo)
	if err != nil {
		return err
	}

	// finish authentication
	return AuthenticateWithSafeCookie(ctx, conn, challenge)
}

// AuthenticateWithSafeCookie authenticates the client using the SAFECOOKIE command.
func AuthenticateWithSafeCookie(ctx context.Context, conn Conn, challenge *AuthChallengeResponse) error {
	// calculate the client hash
	m := hmac.New(sha256.New, []byte("Tor safe cookie authentication controller-to-server hash"))
	m.Write(challenge.Cookie)
	m.Write(challenge.ClientNonce)
	m.Write(challenge.ServerNonce)
	authBytes := m.Sum(nil)

	// send request and receive response
	resp, err := conn.SendRecv(ctx, "AUTHENTICATE %v", hex.EncodeToString(authBytes))
	if err != nil {
		return err
	}

	// make sure the response is successful
	if resp.Status != torcontrolnet.StatusOk {
		return ErrRequestFailed
	}

	// we do not otherwise care about the response
	return nil
}
