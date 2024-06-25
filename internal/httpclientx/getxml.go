package httpclientx

//
// getxml.go - GET an XML response.
//

import (
	"context"
	"encoding/xml"
)

// GetXML sends a GET request and reads an XML response.
//
// Arguments:
//
// - ctx is the cancellable context;
//
// - base is the HTTP [*BaseURL] to use;
//
// - config is the config to use.
//
// This function either returns an error or a valid Output.
func GetXML[Output any](ctx context.Context, base *BaseURL, config *Config) (Output, error) {
	return OverlappedIgnoreIndex(NewOverlappedGetXML[Output](config).Run(ctx, base))
}

func getXML[Output any](ctx context.Context, base *BaseURL, config *Config) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, base, config)

	// handle the case of error
	if err != nil {
		return zeroValue[Output](), err
	}

	// parse the response body as JSON
	var output Output
	if err := xml.Unmarshal(rawrespbody, &output); err != nil {
		return zeroValue[Output](), err
	}

	// TODO(bassosimone): it's unclear to me whether output can be nil when unmarshaling
	// XML input, since there is no "null" in XML. In any case, the code below checks for
	// and avoids emitting nil, so I guess we should be fine here.

	// avoid returning nil pointers, maps, slices
	return NilSafetyErrorIfNil(output)
}
