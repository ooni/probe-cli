package ooapi

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
)

// loginState is the struct saved in the kvstore
// to keep track of the login state.
type loginState struct {
	ClientID string
	Expire   time.Time
	Password string
	Token    string
}

func (ls *loginState) credentialsValid() bool {
	return ls.ClientID != "" && ls.Password != ""
}

func (ls *loginState) tokenValid() bool {
	return ls.Token != "" && time.Now().Add(30*time.Second).After(ls.Expire)
}

// loginKey is the key with which loginState is saved
// into the key-value store used by Client.
const loginKey = "orchestra.state"

// newRandomPassword generates a new random password.
func newRandomPassword() string {
	b := make([]byte, 48)
	_, err := rand.Read(b)
	runtimex.PanicOnError(err, "rand.Read failed")
	return base64.StdEncoding.EncodeToString(b)
}

// newRegisterRequest creates a new RegisterRequest.
func newRegisterRequest() *apimodel.RegisterRequest {
	return &apimodel.RegisterRequest{
		Metadata: apimodel.RegisterRequestMetadata{
			// The original implementation has as its only use case that we
			// were registering and logging in for sending an update regarding
			// the probe whereabouts. Yet here in probe-engine, the orchestra
			// is currently only used to fetch inputs. For this purpose, we don't
			// need to communicate any specific information. The code that will
			// perform an update used to be responsible of doing that. Now, we
			// are not using orchestra for this purpose anymore.
			Platform:        "miniooni",
			ProbeASN:        "AS0",
			ProbeCC:         "ZZ",
			SoftwareName:    "miniooni",
			SoftwareVersion: "0.1.0-dev",
			SupportedTests:  []string{"web_connectivity"},
		},
		Password: newRandomPassword(),
	}
}
