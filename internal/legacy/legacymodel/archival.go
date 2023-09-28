package legacymodel

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

// ArchivalMaybeBinaryData is a possibly binary string. We use this helper class
// to define a custom JSON encoder that allows us to choose the proper
// representation depending on whether the Value field is valid UTF-8 or not.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md#maybebinarydata
//
// Deprecated: do not use this type in new code.
//
// Removing this struct is TODO(https://github.com/ooni/probe/issues/2543).
type ArchivalMaybeBinaryData struct {
	Value string
}

// MarshalJSON marshals a string-like to JSON following the OONI spec that
// says that UTF-8 content is represented as string and non-UTF-8 content is
// instead represented using `{"format":"base64","data":"..."}`.
func (hb ArchivalMaybeBinaryData) MarshalJSON() ([]byte, error) {
	// if we can serialize as UTF-8 string, do that
	if utf8.ValidString(hb.Value) {
		return json.Marshal(hb.Value)
	}

	// otherwise fallback to the ooni/spec representation for binary data
	er := make(map[string]string)
	er["format"] = "base64"
	er["data"] = base64.StdEncoding.EncodeToString([]byte(hb.Value))
	return json.Marshal(er)
}

// UnmarshalJSON is the opposite of MarshalJSON.
func (hb *ArchivalMaybeBinaryData) UnmarshalJSON(d []byte) error {
	if err := json.Unmarshal(d, &hb.Value); err == nil {
		return nil
	}
	er := make(map[string]string)
	if err := json.Unmarshal(d, &er); err != nil {
		return err
	}
	if v, ok := er["format"]; !ok || v != "base64" {
		return errors.New("missing or invalid format field")
	}
	if _, ok := er["data"]; !ok {
		return errors.New("missing data field")
	}
	b64, err := base64.StdEncoding.DecodeString(er["data"])
	if err != nil {
		return err
	}
	hb.Value = string(b64)
	return nil
}
