// Code generated by go generate; DO NOT EDIT.
// 2021-03-10 13:17:34.342751918 +0100 CET m=+0.000196026

package ooapi

//go:generate go run ./internal/generator -file login.go

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

// withLoginPsiphonConfigAPI implements login for simplePsiphonConfigAPI.
type withLoginPsiphonConfigAPI struct {
	API         clonerForPsiphonConfigAPI // mandatory
	JSONCodec   JSONCodec                 // optional
	KVStore     KVStore                   // mandatory
	RegisterAPI callerForRegisterAPI      // mandatory
	LoginAPI    callerForLoginAPI         // mandatory
}

// Call logins, if needed, then calls the API.
func (api *withLoginPsiphonConfigAPI) Call(ctx context.Context, req *apimodel.PsiphonConfigRequest) (apimodel.PsiphonConfigResponse, error) {
	token, err := api.maybeLogin(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := api.API.WithToken(token).Call(ctx, req)
	if errors.Is(err, ErrUnauthorized) {
		// Maybe the clock is just off? Let's try to obtain
		// a token again and see if this fixes it.
		if token, err = api.forceLogin(ctx); err == nil {
			switch resp, err = api.API.WithToken(token).Call(ctx, req); err {
			case nil:
				return resp, nil
			case ErrUnauthorized:
				// fallthrough
			default:
				return nil, err
			}
		}
		// Okay, this seems a broader problem. How about we try
		// and re-register ourselves again instead?
		token, err = api.forceRegister(ctx)
		if err != nil {
			return nil, err
		}
		resp, err = api.API.WithToken(token).Call(ctx, req)
		// fallthrough
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (api *withLoginPsiphonConfigAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *withLoginPsiphonConfigAPI) readstate() (*loginState, error) {
	data, err := api.KVStore.Get(loginKey)
	if err != nil {
		return nil, err
	}
	var ls loginState
	if err := api.jsonCodec().Decode(data, &ls); err != nil {
		return nil, err
	}
	return &ls, nil
}

func (api *withLoginPsiphonConfigAPI) writestate(ls *loginState) error {
	data, err := api.jsonCodec().Encode(*ls)
	if err != nil {
		return err
	}
	return api.KVStore.Set(loginKey, data)
}

func (api *withLoginPsiphonConfigAPI) doRegister(ctx context.Context, password string) (string, error) {
	req := newRegisterRequest(password)
	ls := &loginState{}
	resp, err := api.RegisterAPI.Call(ctx, req)
	if err != nil {
		return "", err
	}
	ls.ClientID = resp.ClientID
	ls.Password = req.Password
	return api.doLogin(ctx, ls)
}

func (api *withLoginPsiphonConfigAPI) forceRegister(ctx context.Context) (string, error) {
	var password string
	// If we already have a previous password, let us keep
	// using it. This will allow a new version of the API to
	// be able to continue to identify this probe. (This
	// assumes that we have a stateless API that generates
	// the user ID as a signature of the password plus a
	// timestamp and that the key to generate the signature
	// is not lost. If all these conditions are met, we
	// can then serve better test targets to more long running
	// (and therefore trusted) probes.)
	if ls, err := api.readstate(); err == nil {
		password = ls.Password
	}
	if password == "" {
		password = newRandomPassword()
	}
	return api.doRegister(ctx, password)
}

func (api *withLoginPsiphonConfigAPI) forceLogin(ctx context.Context) (string, error) {
	ls, err := api.readstate()
	if err != nil {
		return "", err
	}
	return api.doLogin(ctx, ls)
}

func (api *withLoginPsiphonConfigAPI) maybeLogin(ctx context.Context) (string, error) {
	ls, _ := api.readstate()
	if ls == nil || !ls.credentialsValid() {
		return api.forceRegister(ctx)
	}
	if !ls.tokenValid() {
		return api.doLogin(ctx, ls)
	}
	return ls.Token, nil
}

func (api *withLoginPsiphonConfigAPI) doLogin(ctx context.Context, ls *loginState) (string, error) {
	req := &apimodel.LoginRequest{
		ClientID: ls.ClientID,
		Password: ls.Password,
	}
	resp, err := api.LoginAPI.Call(ctx, req)
	if err != nil {
		return "", err
	}
	ls.Token = resp.Token
	ls.Expire = resp.Expire
	if err := api.writestate(ls); err != nil {
		return "", err
	}
	return ls.Token, nil
}

var _ callerForPsiphonConfigAPI = &withLoginPsiphonConfigAPI{}

// withLoginTorTargetsAPI implements login for simpleTorTargetsAPI.
type withLoginTorTargetsAPI struct {
	API         clonerForTorTargetsAPI // mandatory
	JSONCodec   JSONCodec              // optional
	KVStore     KVStore                // mandatory
	RegisterAPI callerForRegisterAPI   // mandatory
	LoginAPI    callerForLoginAPI      // mandatory
}

// Call logins, if needed, then calls the API.
func (api *withLoginTorTargetsAPI) Call(ctx context.Context, req *apimodel.TorTargetsRequest) (apimodel.TorTargetsResponse, error) {
	token, err := api.maybeLogin(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := api.API.WithToken(token).Call(ctx, req)
	if errors.Is(err, ErrUnauthorized) {
		// Maybe the clock is just off? Let's try to obtain
		// a token again and see if this fixes it.
		if token, err = api.forceLogin(ctx); err == nil {
			switch resp, err = api.API.WithToken(token).Call(ctx, req); err {
			case nil:
				return resp, nil
			case ErrUnauthorized:
				// fallthrough
			default:
				return nil, err
			}
		}
		// Okay, this seems a broader problem. How about we try
		// and re-register ourselves again instead?
		token, err = api.forceRegister(ctx)
		if err != nil {
			return nil, err
		}
		resp, err = api.API.WithToken(token).Call(ctx, req)
		// fallthrough
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (api *withLoginTorTargetsAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *withLoginTorTargetsAPI) readstate() (*loginState, error) {
	data, err := api.KVStore.Get(loginKey)
	if err != nil {
		return nil, err
	}
	var ls loginState
	if err := api.jsonCodec().Decode(data, &ls); err != nil {
		return nil, err
	}
	return &ls, nil
}

func (api *withLoginTorTargetsAPI) writestate(ls *loginState) error {
	data, err := api.jsonCodec().Encode(*ls)
	if err != nil {
		return err
	}
	return api.KVStore.Set(loginKey, data)
}

func (api *withLoginTorTargetsAPI) doRegister(ctx context.Context, password string) (string, error) {
	req := newRegisterRequest(password)
	ls := &loginState{}
	resp, err := api.RegisterAPI.Call(ctx, req)
	if err != nil {
		return "", err
	}
	ls.ClientID = resp.ClientID
	ls.Password = req.Password
	return api.doLogin(ctx, ls)
}

func (api *withLoginTorTargetsAPI) forceRegister(ctx context.Context) (string, error) {
	var password string
	// If we already have a previous password, let us keep
	// using it. This will allow a new version of the API to
	// be able to continue to identify this probe. (This
	// assumes that we have a stateless API that generates
	// the user ID as a signature of the password plus a
	// timestamp and that the key to generate the signature
	// is not lost. If all these conditions are met, we
	// can then serve better test targets to more long running
	// (and therefore trusted) probes.)
	if ls, err := api.readstate(); err == nil {
		password = ls.Password
	}
	if password == "" {
		password = newRandomPassword()
	}
	return api.doRegister(ctx, password)
}

func (api *withLoginTorTargetsAPI) forceLogin(ctx context.Context) (string, error) {
	ls, err := api.readstate()
	if err != nil {
		return "", err
	}
	return api.doLogin(ctx, ls)
}

func (api *withLoginTorTargetsAPI) maybeLogin(ctx context.Context) (string, error) {
	ls, _ := api.readstate()
	if ls == nil || !ls.credentialsValid() {
		return api.forceRegister(ctx)
	}
	if !ls.tokenValid() {
		return api.doLogin(ctx, ls)
	}
	return ls.Token, nil
}

func (api *withLoginTorTargetsAPI) doLogin(ctx context.Context, ls *loginState) (string, error) {
	req := &apimodel.LoginRequest{
		ClientID: ls.ClientID,
		Password: ls.Password,
	}
	resp, err := api.LoginAPI.Call(ctx, req)
	if err != nil {
		return "", err
	}
	ls.Token = resp.Token
	ls.Expire = resp.Expire
	if err := api.writestate(ls); err != nil {
		return "", err
	}
	return ls.Token, nil
}

var _ callerForTorTargetsAPI = &withLoginTorTargetsAPI{}
