package torx

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

// ErrControlInvalidAuthChallengeResponse indicates that we received
// and invalid AUTHCHALLENGE response from tor.
var ErrControlInvalidAuthChallengeResponse = errors.New("torx: control: invalid AUTHCHALLENGE response")

// ErrControlInvalidServerHash indicates that the server hash is invalid.
var ErrControlInvalidServerHash = errors.New("torx: control: invalid SERVERHASH value")

// ErrControlInvalidServerNonce indicates that the server nonce is invalid.
var ErrControlInvalidServerNonce = errors.New("torx: control: invalid SERVERNONCE value")

// ErrControlInvalidCookieFile indicates that the cookie file is invalid.
var ErrControlInvalidCookieFile = errors.New("torx: control: invalid cookie file")

// ErrControlServerHashMismatch indicates that there is a server hash mismatch.
var ErrControlServerHashMismatch = errors.New("torx: control: server hash mismatch")

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
func AuthChallenge(ctx context.Context, conn ControlTransport, pinfo *ProtocolInfoResponse) (*AuthChallengeResponse, error) {
	cookie, err := os.ReadFile(pinfo.CookieFile)
	if err != nil {
		return nil, err
	}
	if len(cookie) != 32 {
		return nil, ErrControlInvalidCookieFile
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
		return nil, ErrControlRequestFailed
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
		return nil, ErrControlInvalidAuthChallengeResponse
	}

	// extract the server hash
	serverHash, err := hex.DecodeString(splitResp[1][11:])
	if err != nil {
		return nil, ErrControlInvalidServerHash
	}
	if len(serverHash) != 32 {
		return nil, ErrControlInvalidServerHash
	}

	// extract the server nonce
	serverNonce, err := hex.DecodeString(splitResp[2][12:])
	if err != nil {
		return nil, ErrControlInvalidServerNonce
	}
	if len(serverNonce) != 32 {
		return nil, ErrControlInvalidServerNonce
	}

	// make sure that the server hash is correct
	m := hmac.New(sha256.New, []byte("Tor safe cookie authentication server-to-controller hash"))
	m.Write(cookie)
	m.Write(clientNonce[:])
	m.Write(serverNonce)
	computedServerHash := m.Sum(nil)
	if !hmac.Equal(serverHash, computedServerHash) {
		return nil, ErrControlServerHashMismatch
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
