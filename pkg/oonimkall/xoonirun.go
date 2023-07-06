package oonimkall

//
// eXperimental OONI Run code.
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// XOONIRunDescriptor describes a list of nettests to run together.
type XOONIRunDescriptor struct {
	// Author contains the author's name.
	Author string `json:"author"`

	// Description contains a long description.
	Description string `json:"description"`

	// DescriptionIntl_ contains the i18n descriptions.
	DescriptionIntl_ map[string]string `json:"description_intl"`

	// Icon contains the icon name.
	Icon string `json:"icon"`

	// IsArchived indicates whether the descriptor has been archived.
	IsArchived bool `json:"is_archived"`

	// Name is the name of this descriptor.
	Name string `json:"name"`

	// NameIntl_ contains the i18n names.
	NameIntl_ map[string]string `json:"name_intl"`

	// Nettests contains the list of nettests to run.
	Nettests []*XOONIRunNettest `json:"nettests"`

	// ShortDescription contains a descriptor short description.
	ShortDescription string `json:"short_description"`

	// ShortDescriptionIntl contains i18n short descriptions.
	ShortDescriptionIntl_ map[string]string `json:"short_description_intl"`
}

// ErrXOONIRunNoSuchTranslation indicates that there is no translation for the given string.
var ErrXOONIRunNoSuchTranslation = errors.New("xoonirun: no such translation")

// DescriptionIntl returns the internationalized description.
func (d *XOONIRunDescriptor) DescriptionIntl(lang string) (string, error) {
	value, good := d.DescriptionIntl_[lang]
	if !good {
		return "", ErrXOONIRunNoSuchTranslation
	}
	return value, nil
}

// NameIntl returns the internationalized name.
func (d *XOONIRunDescriptor) NameIntl(lang string) (string, error) {
	value, good := d.NameIntl_[lang]
	if !good {
		return "", ErrXOONIRunNoSuchTranslation
	}
	return value, nil
}

// ShortDescriptionIntl returns the internationalized short description.
func (d *XOONIRunDescriptor) ShortDescriptionIntl(lang string) (string, error) {
	value, good := d.ShortDescriptionIntl_[lang]
	if !good {
		return "", ErrXOONIRunNoSuchTranslation
	}
	return value, nil
}

// ArrayOfXOONIRunNettest is a list of XOONIRunNettest.
type ArrayOfXOONIRunNettest struct {
	v []*XOONIRunNettest
}

// GetNettests returns the nettests.
func (d *XOONIRunDescriptor) GetNettests() *ArrayOfXOONIRunNettest {
	return &ArrayOfXOONIRunNettest{d.Nettests}
}

// Size returns the array size.
func (arr *ArrayOfXOONIRunNettest) Size() int64 {
	return int64(len(arr.v))
}

// ErrXOONIRunIndexOutOfBounds indicates that you used an index out of bounds.
var ErrXOONIRunIndexOutOfBounds = errors.New("xoonirun: index out of bounds")

// At returns the array element at the given position.
func (arr *ArrayOfXOONIRunNettest) At(idx int64) (*XOONIRunNettest, error) {
	if idx < 0 || idx >= int64(len(arr.v)) {
		return nil, ErrXOONIRunIndexOutOfBounds
	}
	return arr.v[idx], nil
}

// XOONIRunNettest specifies how a nettest should run.
type XOONIRunNettest struct {
	// Inputs_ contains inputs for the experiment.
	Inputs_ []string `json:"inputs"`

	// TestName contains the nettest name.
	TestName string `json:"test_name"`
}

// ArrayOfString contains an array of strings.
type ArrayOfString struct {
	v []string
}

// Inputs returns the inputs
func (nt *XOONIRunNettest) Inputs() *ArrayOfString {
	return &ArrayOfString{nt.Inputs_}
}

// Size returns the array size.
func (arr *ArrayOfString) Size() int64 {
	return int64(len(arr.v))
}

// At returns the array element at the given position.
func (arr *ArrayOfString) At(idx int64) (string, error) {
	if idx < 0 || idx >= int64(len(arr.v)) {
		return "", ErrXOONIRunIndexOutOfBounds
	}
	return arr.v[idx], nil
}

// XOONIRunFetchResponse is the response to the fetch API.
type XOONIRunFetchResponse struct {
	CreationTime string              `json:"creation_time"`
	Descriptor   *XOONIRunDescriptor `json:"descriptor"`
	V            int64               `json:"v"`
}

// OONIRunFetch fetches a given OONI run descriptor.
func (sess *Session) OONIRunFetch(ctx *Context, ID int64) (*XOONIRunFetchResponse, error) {
	sess.mtx.Lock()
	defer sess.mtx.Unlock()

	clnt := sess.sessp.DefaultHTTPClient()

	// https://ams-pg-test.ooni.org/api/_/ooni_run/fetch/297500125102
	URL := &url.URL{
		Scheme:      "https",
		Opaque:      "",
		User:        nil,
		Host:        "ams-pg-test.ooni.org",
		Path:        fmt.Sprintf("/api/_/ooni_run/fetch/%d", ID),
		RawPath:     "",
		OmitHost:    false,
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}

	req, err := http.NewRequestWithContext(ctx.ctx, "GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("xoonirun: HTTP request failed")
	}

	rawResp, err := netxlite.ReadAllContext(ctx.ctx, resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp XOONIRunFetchResponse
	if err := json.Unmarshal(rawResp, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}
