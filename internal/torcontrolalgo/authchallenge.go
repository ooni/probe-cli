package torcontrolalgo

//
// autchallenge.go - implements the AUTHCHALLENGE command.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// ErrInvalidAuthChallengeResponse indicates that we received
// and invalid AUTHCHALLENGE response from tor.
var ErrInvalidAuthChallengeResponse = errors.New("torcontrol: invalid AUTHCHALLENGE response")

// ErrInvalidServerHash indicates that the server hash is invalid.
var ErrInvalidServerHash = errors.New("torcontrol: invalid SERVERHASH value")

// ErrInvalidServerNonce indicates that the server nonce is invalid.
var ErrInvalidServerNonce = errors.New("torcontrol: invalid SERVERNONCE value")

// ErrInvalidCookieFile indicates that the cookie file is invalid.
var ErrInvalidCookieFile = errors.New("torcontrol: invalid cookie file")

// ErrServerHashMismatch indicates that there is a server hash mismatch.
var ErrServerHashMismatch = errors.New("torcontrol: server hash mismatch")

// AuthChallengeResponse is the response returned by the AUTHCHALLENGE command.
type AuthChallengeResponse struct {
	// ClientNonce is the client nonce.
	ClientNonce []byte

	// Cookie is the content of the cookie file.
	Cookie []byte

	// ServerHash is the server-computed server hash.
	ServerHash []byte

	// ServerNonce is the server nonce.
	ServerNonce []byte
}

// AuthChallenge sends the AUTHCHALLENGE command.
func AuthChallenge(ctx context.Context, conn Conn, pinfo *ProtocolInfoResponse) (*AuthChallengeResponse, error) {
	cookie, err := os.ReadFile(pinfo.CookieFile)
	if err != nil {
		return nil, err
	}
	if len(cookie) != 32 {
		return nil, ErrInvalidCookieFile
	}

	// generate a random client nonce
	var clientNonce [32]byte
	if _, err := rand.Read(clientNonce[:]); err != nil {
		return nil, err
	}

	// send request and receive response
	resp, err := conn.SendRecv(ctx, "AUTHCHALLENGE SAFECOOKIE %s", hex.EncodeToString(clientNonce[:]))
	if err != nil {
		return nil, err
	}

	// make sure the response is successful
	if resp.Status != torcontrolnet.StatusOk {
		return nil, ErrRequestFailed
	}

	// This is the expected format according to the spec:
	//
	//	"250 AUTHCHALLENGE"
	//		SP "SERVERHASH=" ServerHash
	//		SP "SERVERNONCE=" ServerNonce
	//		CRLF
	//
	// We account for possibly additional fields after SERVERNONCE=.
	splitResp := strings.Split(resp.EndReplyLine, " ")
	if len(splitResp) < 3 || !strings.HasPrefix(splitResp[1], "SERVERHASH=") ||
		!strings.HasPrefix(splitResp[2], "SERVERNONCE=") {
		return nil, ErrInvalidAuthChallengeResponse
	}

	// extract the server hash
	serverHash, err := hex.DecodeString(splitResp[1][11:])
	if err != nil {
		return nil, ErrInvalidServerHash
	}
	if len(serverHash) != 32 {
		return nil, ErrInvalidServerHash
	}

	// extract the server nonce
	serverNonce, err := hex.DecodeString(splitResp[2][12:])
	if err != nil {
		return nil, ErrInvalidServerNonce
	}
	if len(serverNonce) != 32 {
		return nil, ErrInvalidServerNonce
	}

	// make sure that the server hash is correct
	m := hmac.New(sha256.New, []byte("Tor safe cookie authentication server-to-controller hash"))
	m.Write(cookie)
	m.Write(clientNonce[:])
	m.Write(serverNonce)
	computedServerHash := m.Sum(nil)
	if !hmac.Equal(serverHash, computedServerHash) {
		return nil, ErrServerHashMismatch
	}

	// we're good!
	acr := &AuthChallengeResponse{
		ClientNonce: clientNonce[:],
		Cookie:      cookie,
		ServerHash:  serverHash,
		ServerNonce: serverNonce,
	}
	return acr, nil
}
