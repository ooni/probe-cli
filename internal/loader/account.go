package loader

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// accountInventoryKey is the kvstore key containing the accountInventory.
const accountInventoryKey = "orchestra.state"

// accountInventoryState is the state of a given orchestra account.
type accountInventoryState struct {
	User     string
	Password string
	Token    string
	Expire   time.Time
}

// didExpire returns whether the state has expired.
func (ais *accountInventoryState) didExpire() bool {
	return time.Now().After(ais.Expire)
}

// accountInventoryExpectedVersion is the expected accountInventory version.
const accountInventoryExpectedVersion = 1

// accountInventory is the inventory of all the available orchestra accounts.
type accountInventory struct {
	Domain map[string]*accountInventoryState
	V      int
}

// errNoAccountForEndpoint indicates there's no account for an endpoint.
var errNoAccountForEndpoint = errors.New("loader: no account for endpoint")

// get obtains the accountInventoryState for the given endpoint
func (ai *accountInventory) get(endpoint string) (*accountInventoryState, error) {
	state := ai.Domain[endpoint]
	if state == nil {
		return nil, errNoAccountForEndpoint
	}
	return state, nil
}

// set sets the accountInventoryState for the given endpoint.
func (ai *accountInventory) set(endpoint string, state *accountInventoryState) {
	ai.Domain[endpoint] = state
}

// delete deletes the accountInventoryState for the given endpoint.
func (ai *accountInventory) delete(endpoint string) {
	delete(ai.Domain, endpoint)
}

// errInvalidAccountInventoryVersion indicates that the accountInventory version is invalid.
var errInvalidAccountInventoryVersion = errors.New("loader: invalid account inventory version")

// loadAccountInventory loads the accountInventory from the given key-value store.
func loadAccountInventory(store model.KeyValueStore) (*accountInventory, error) {
	data, err := store.Get(accountInventoryKey)
	if err != nil {
		return nil, err
	}

	var inventory accountInventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		return nil, err
	}

	if inventory.V != accountInventoryExpectedVersion {
		return nil, errInvalidAccountInventoryVersion
	}

	return &inventory, nil
}

// loadAccountInventoryOrDefault loads the account inventory or returns a default one.
func loadAccountInventoryOrDefault(store model.KeyValueStore) *accountInventory {
	inventory, err := loadAccountInventory(store)
	if err != nil {
		inventory = &accountInventory{
			Domain: map[string]*accountInventoryState{},
			V:      accountInventoryExpectedVersion,
		}
	}
	return inventory
}

// storeAccountInventory saves the account inventory into the key-value store.
func storeAccountInventory(store model.KeyValueStore, inventory *accountInventory) error {
	data, err := json.Marshal(inventory)
	if err != nil {
		return err
	}
	return store.Set(accountInventoryKey, data)
}

// accountCallWithToken ensures that the given function is called with the correct
// token. More in detail, this func performs the following actions:
//
// 1. make sure we have a valid token;
//
// 2. invoke the underlying func;
//
// 3. if still unauthorized, clear the state such that the next call causes
// the code to perform the login flow from scratch.
//
// We do not re-register immediately because we want to avoid registering again
// immediately, which could cause issues in case of backend bugs.
func (c *Client) accountCallWithToken(ctx context.Context, fx func(token string) error) error {
	// load account aip
	aip := loadAccountInventoryOrDefault(c.store)
	defer storeAccountInventory(c.store, aip)

	// get the token
	token, err := c.accountTokenOrLoginOrRegisterAndLogin(ctx, aip)
	if err != nil {
		return err
	}

	// invoke the underlying func
	err = fx(token)

	// clear the credentials if we're still unauthorized
	if errors.Is(err, errUnauthorized) {
		aip.delete(c.endpoint)
	}

	return err
}

// accountTokenOrLoginOrRegisterAndLogin returns a token directly, if possible, otherwise
// it logs in, if possible, otherwise it registers and performs a login.
func (c *Client) accountTokenOrLoginOrRegisterAndLogin(ctx context.Context, aip *accountInventory) (string, error) {
	// Token
	token, err := c.accountToken(aip)
	if err == nil {
		return token, nil
	}

	// OrLogin
	if errors.Is(err, errAccountNeedLogin) {
		if err := c.accountLogin(ctx, aip); err == nil {
			return c.accountToken(aip)
		}
	}

	// OrRegisterAndLogin
	if err := c.accountRegister(ctx, aip); err != nil {
		return "", err
	}
	if err := c.accountLogin(ctx, aip); err != nil {
		return "", err
	}
	return c.accountToken(aip)
}

// errAccountNeedLogin means we need to login again.
var errAccountNeedLogin = errors.New("loader: account needs login")

// accountGetToken returns the token required to authenticate with the orchestra services.
func (c *Client) accountToken(aip *accountInventory) (string, error) {
	// load state for the endpoint
	state, err := aip.get(c.endpoint)
	if err != nil {
		return "", err
	}

	// return an error if the account expired
	if state.didExpire() {
		return "", errAccountNeedLogin
	}

	return state.Token, nil
}

// accountLogin logins using existing credentials and returns the token to use.
func (c *Client) accountLogin(ctx context.Context, aip *accountInventory) error {
	// load state for the endpoint
	state, err := aip.get(c.endpoint)
	if err != nil {
		return err
	}

	// fill the request body
	creds := &model.OOAPILoginCredentials{
		Username: state.User,
		Password: state.Password,
	}

	// create the request URL
	URL := &url.URL{
		Scheme: "https",
		Host:   c.endpoint,
		Path:   "/api/v1/login",
	}

	// serialize the request body
	rawReqBody := must.MarshalJSON(creds)
	c.logger.Debugf("raw login request: %s", string(rawReqBody))

	// create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL.String(), bytes.NewReader(rawReqBody))
	if err != nil {
		return err
	}

	// perform the HTTP round trip
	resp, err := c.txp.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// handle HTTP request failures
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d %s", ErrHTTPFailure, resp.StatusCode, resp.Status)
	}

	// read the raw response body
	rawRespBody, err := netxlite.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return err
	}
	c.logger.Debugf("raw login response: %s", string(rawRespBody))

	// parse the raw response body
	var res model.OOAPILoginAuth
	if err := json.Unmarshal(rawRespBody, &res); err != nil {
		return err
	}

	// update credentials
	state.Expire = res.Expire
	state.Token = res.Token
	return nil
}

// accountNewPassword generates a new password
func accountNewPassword() (string, error) {
	raw := make([]byte, 48)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// accountRegister creates a new orchestra account for this probe.
func (c *Client) accountRegister(ctx context.Context, aip *accountInventory) error {
	// create a password
	password, err := accountNewPassword()
	if err != nil {
		return err
	}

	// fill the request body
	//
	// The original implementation has as its only use case that we
	// were registering and logging in for sending an update regarding
	// the probe whereabouts. Yet here in probe-engine, the orchestra
	// is currently only used to fetch inputs. For this purpose, we don't
	// need to communicate any specific information. The code that will
	// perform an update used to be responsible of doing that. Now, we
	// are not using orchestra for this purpose anymore.
	creds := &model.OOAPIRegisterRequest{
		OOAPIProbeMetadata: model.OOAPIProbeMetadata{
			Platform:        "miniooni",
			ProbeASN:        "AS0",
			ProbeCC:         "ZZ",
			SoftwareName:    "miniooni",
			SoftwareVersion: "0.1.0-dev",
			SupportedTests:  []string{"web_connectivity"},
		},
		Password: password,
	}

	// create the request URL
	URL := &url.URL{
		Scheme: "https",
		Host:   c.endpoint,
		Path:   "/api/v1/register",
	}

	// serialize the request body
	rawReqBody := must.MarshalJSON(creds)
	c.logger.Debugf("raw register request: %s", string(rawReqBody))

	// create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL.String(), bytes.NewReader(rawReqBody))
	if err != nil {
		return err
	}

	// perform the HTTP round trip
	resp, err := c.txp.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// handle HTTP request failures
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d %s", ErrHTTPFailure, resp.StatusCode, resp.Status)
	}

	// read the raw response body
	rawRespBody, err := netxlite.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return err
	}
	c.logger.Debugf("raw register response: %s", string(rawRespBody))

	// parse the raw response body
	var res model.OOAPIRegisterResponse
	if err := json.Unmarshal(rawRespBody, &res); err != nil {
		return err
	}

	// overwrite the credentials and write back
	aip.set(c.endpoint, &accountInventoryState{
		User:     res.ClientID,
		Expire:   time.Time{}, // zero time
		Password: password,
		Token:    "", // unknown
	})
	return nil
}
