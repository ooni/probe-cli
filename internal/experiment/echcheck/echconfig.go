package echcheck

import (
	"encoding/base64"
)

const extensionEncryptedClientHello uint16 = 0xfe0d

type echConfigList struct {
	Configs []echConfig
	raw     []byte
}

// The raw data of the ECHConfigList as it would appear in the DNS record.
func (ecl echConfigList) Base64() string {
	return base64.StdEncoding.EncodeToString(ecl.raw)
}

// Uses go's own tls package's parsing implementation and returns
// the result along with the raw data in an echConfigList.
func parseRawEchConfig(data []byte) (echConfigList, error) {
	configs, err := parseECHConfigList(data)
	if err != nil {
		return echConfigList{}, err
	}
	return echConfigList{
		raw:     data,
		Configs: configs,
	}, nil
}
