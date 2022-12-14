package wireguard

import (
	"encoding/base64"
	"encoding/hex"
)

// Config contains the experiment config.
type Config struct {
	ConfigFile    string `ooni:"Configuration file for the WireGuard experiment"`
	URLs          string `ooni:"comma-separated list of extra URLs to fetch through the tunnel"`
	PingCount     string `ooni:"number of icmp pings to send (default: 10)"`
	WithSpeedTest string `ooni:"if yes, perform a speed test instead of fetching the given list of urls"`
	Remote        string
	// Safe_XXX options are not sent to the backend for archival.
	SafePrivateKey   string
	SafePublicKey    string
	SafePresharedKey string
	SafeIP           string
	SafeLocalCreds   string
}

type options struct {
	ip           string
	pubKey       string
	privKey      string
	presharedKey string
	endpoint     string
	ns           string
}

func getOptionsFromConfig(c Config) (*options, error) {
	o := &options{}

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
	o.endpoint = c.Remote

	o.ns = defaultNameserver
	return o, nil

}
