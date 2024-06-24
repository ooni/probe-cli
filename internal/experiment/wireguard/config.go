package wireguard

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// Config contains the experiment config.
//
// This contains all the settings that user can set to modify the behaviour
// of this experiment. By tagging these variables with `ooni:"..."`, we allow
// miniooni's -O flag to find them and set them.
type Config struct {
	Verbose bool `ooni:"Use extra-verbose mode in wireguard logs"`

	// These flags modify what sensitive information is stored in the report and submitted to the backend.
	PublicTarget            bool `ooni:"Treat the target endpoint as public data (if true, it will be included in the report)"`
	PublicAmneziaParameters bool `ooni:"Treat the AmneziaWG advanced security parameters as public data"`

	// Safe_XXX options are not sent to the backend for archival by default.
	SafeRemote string `ooni:"Remote to connect to using WireGuard"`
	SafeIP     string `ooni:"Allocated IP for this peer"`

	// Keys are base-64 encoded
	SafePrivateKey   string `ooni:"Private key to connect to remote (base64)"`
	SafePublicKey    string `ooni:"Public key of the remote (base64)"`
	SafePresharedKey string `ooni:"Pre-shared key for authentication (base64)"`

	// Optional obfuscation parameters for AmneziaWG
	SafeJc   string `ooni:"jc"`
	SafeJmin string `ooni:"jmin"`
	SafeJmax string `ooni:"jmax"`
	SafeS1   string `ooni:"s1"`
	SafeS2   string `ooni:"s2"`
	SafeH1   string `ooni:"h1"`
	SafeH2   string `ooni:"h2"`
	SafeH3   string `ooni:"h3"`
	SafeH4   string `ooni:"h4"`
}

type wireguardOptions struct {
	// common wireguard parameters
	endpoint string
	ip       string
	ns       string

	// keys are hex-encoded
	pubKey       string
	privKey      string
	presharedKey string

	// optional parameters for AmneziaWG nodes
	jc   string
	jmin string
	jmax string
	s1   string
	s2   string
	h1   string
	h2   string
	h3   string
	h4   string
}

// amneziaValues returns an array with all the amnezia-specific configuration
// parameters.
func (wo *wireguardOptions) amneziaValues() []string {
	return []string{
		wo.jc, wo.jmin, wo.jmax,
		wo.s1, wo.s2,
		wo.h1, wo.h2, wo.h3, wo.h4,
	}
}

// validate returns true if this looks like a sensible wireguard configuration.
func (wo *wireguardOptions) validate() bool {
	if wo.endpoint == "" || wo.ip == "" || wo.pubKey == "" || wo.privKey == "" || wo.presharedKey == "" {
		return false
	}
	if isAnyFilled(wo.amneziaValues()...) {
		return !isAnyEmpty(wo.amneziaValues()...)
	}
	return true
}

// isAmneziaFlavored returns true if none of the mandatory amnezia fields are empty.
func (wo *wireguardOptions) isAmneziaFlavored() bool {
	return !isAnyEmpty(wo.amneziaValues()...)
}

// amneziaConfigHash is a hash representation of the custom parameters in this amneziaWG node.
// intended to be used if PublicAmneziaParameters=false, so that we can verify that we're testing
// the same node.
func (wo *wireguardOptions) configurationHash() string {
	if !wo.isAmneziaFlavored() {
		return ""
	}
	return sha1Sum(append(wo.amneziaValues(), wo.endpoint)...)
}

func sha1Sum(strings ...string) string {
	hasher := sha1.New()
	for _, str := range strings {
		io.WriteString(hasher, str)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func newWireguardOptionsFromConfig(c *Config) (*wireguardOptions, error) {
	o := &wireguardOptions{}

	pub, err := base64.StdEncoding.DecodeString(c.SafePublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decode public key", ErrInvalidInput)
	}
	pubHex := hex.EncodeToString(pub)
	o.pubKey = pubHex

	priv, err := base64.StdEncoding.DecodeString(c.SafePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decode private key", ErrInvalidInput)
	}
	privHex := hex.EncodeToString(priv)
	o.privKey = privHex

	psk, err := base64.StdEncoding.DecodeString(c.SafePresharedKey)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decode pre-shared key", ErrInvalidInput)
	}
	pskHex := hex.EncodeToString(psk)
	o.presharedKey = pskHex

	// TODO(ainghazal): reconcile this with Input if c.PublicTarget=true
	o.endpoint = c.SafeRemote

	o.ip = c.SafeIP

	// amnezia parameters
	o.jc = c.SafeJc
	o.jmin = c.SafeJmin
	o.jmax = c.SafeJmax
	o.s1 = c.SafeS1
	o.s2 = c.SafeS2
	o.h1 = c.SafeH1
	o.h2 = c.SafeH2
	o.h3 = c.SafeH3
	o.h4 = c.SafeH4

	o.ns = defaultNameserver
	return o, nil
}

func isAnyFilled(fields ...string) bool {
	for _, f := range fields {
		if f != "" {
			return true
		}
	}
	return false
}

func isAnyEmpty(fields ...string) bool {
	for _, f := range fields {
		if f == "" {
			return true
		}
	}
	return false
}
