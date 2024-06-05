package wireguard

import (
	"encoding/base64"
	"encoding/hex"
)

// Config contains the experiment config.
//
// This contains all the settings that user can set to modify the behaviour
// of this experiment. By tagging these variables with `ooni:"..."`, we allow
// miniooni's -O flag to find them and set them.
type Config struct {
	ConfigFile string `ooni:"Configuration file for the WireGuard experiment"`

	// TODO(ainghzal): honor it
	PublicTarget bool `ooni:"Treat the target endpoint as public data (if true, it will be included in the report)`

	Verbose bool `ooni:"Use extra-verbose mode in wireguard logs"`

	// Safe_XXX options are not sent to the backend for archival.
	SafeRemote       string `ooni:"Remote to connect to using WireGuard"`
	SafePrivateKey   string `ooni:"Private key to connect to remote"`
	SafePublicKey    string `ooni:"Public key of the remote"`
	SafePresharedKey string `ooni:"Pre-shared key for authentication"`
	SafeIP           string `ooni:"Allocated IP for this peer"`

	// Optional obfuscation parameters for AmneziaWG
	Jc   string `ooni:"jc"`
	Jmin string `ooni:"jmin"`
	Jmax string `ooni:"jmax"`
	S1   string `ooni:"s1"`
	S2   string `ooni:"s2"`
	H1   string `ooni:"h1"`
	H2   string `ooni:"h2"`
	H3   string `ooni:"h3"`
	H4   string `ooni:"h4"`
}

type options struct {
	// common wireguard parameters
	endpoint     string
	ip           string
	pubKey       string
	privKey      string
	presharedKey string
	ns           string

	// parameters from AmneziaWG
	// TODO(ainghazal: make these optional)
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

func getOptionsFromConfig(c Config) (options, error) {
	o := options{}

	pub, _ := base64.StdEncoding.DecodeString(c.SafePublicKey)
	pubHex := hex.EncodeToString(pub)
	o.pubKey = pubHex

	priv, _ := base64.StdEncoding.DecodeString(c.SafePrivateKey)
	privHex := hex.EncodeToString(priv)
	o.privKey = privHex

	psk, _ := base64.StdEncoding.DecodeString(c.SafePresharedKey)
	pskHex := hex.EncodeToString(psk)
	o.presharedKey = pskHex

	o.ip = c.SafeIP

	// TODO: reconcile this with Input if c.PublicTarget=true
	o.endpoint = c.SafeRemote

	o.jc = c.Jc
	o.jmin = c.Jmin
	o.jmax = c.Jmax
	o.s1 = c.S1
	o.s2 = c.S2
	o.h1 = c.H1
	o.h2 = c.H2
	o.h3 = c.H3
	o.h4 = c.H4

	o.ns = defaultNameserver
	return o, nil
}
