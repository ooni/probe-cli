package testingx

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

func TestOONIBackendWithLoginFlow(t *testing.T) {
	// create state
	state := &OONIBackendWithLoginFlow{}

	// create local testing server
	server := MustNewHTTPServer(state.NewMux())
	defer server.Close()

	// create a fake filler
	ff := &FakeFiller{}

	t.Run("it may be that there's no user record", func(t *testing.T) {
		err := state.DoWithLockedUserRecord("foobar", func(rec *OONIBackendWithLoginFlowUserRecord) error {
			panic("should not be called")
		})
		if err == nil || err.Error() != "no such record" {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("attempt login with invalid method", func(t *testing.T) {
		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/login", "")),
			nil,
		))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see not implemented
		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("attempt login with invalid credentials", func(t *testing.T) {
		// create fake login request
		request := &model.OOAPILoginCredentials{}

		// fill it with random data
		ff.Fill(&request)

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"POST",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/login", "")),
			bytes.NewReader(must.MarshalJSON(request)),
		))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to be unauthorized
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("attempt register with invalid method", func(t *testing.T) {
		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/register", "")),
			nil,
		))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see not implemented
		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	// registerflow attempts to register and returns the username and password
	registerflow := func(t *testing.T) (string, string) {
		// create register request
		//
		// we ignore the metadata because we're testing
		request := &model.OOAPIRegisterRequest{
			OOAPIProbeMetadata: model.OOAPIProbeMetadata{},
			Password:           uuid.Must(uuid.NewRandom()).String(),
		}

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"POST",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/register", "")),
			bytes.NewReader(must.MarshalJSON(request)),
		))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to be authorized
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", resp.StatusCode)
		}

		// read response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))

		// parse the response body
		var response model.OOAPIRegisterResponse
		must.UnmarshalJSON(rawrespbody, &response)

		// return username and password
		return response.ClientID, request.Password
	}

	t.Run("successful register", func(t *testing.T) {
		_, _ = registerflow(t)
	})

	loginrequest := func(username, password string) *http.Response {
		// create login request
		request := &model.OOAPILoginCredentials{
			Username: username,
			Password: password,
		}

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"POST",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/login", "")),
			bytes.NewReader(must.MarshalJSON(request)),
		))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		return resp
	}

	loginflow := func(username, password string) (string, time.Time) {
		// get the response
		resp := loginrequest(username, password)

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to be authorized
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", resp.StatusCode)
		}

		// read response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))

		// parse the response body
		var response model.OOAPILoginAuth
		must.UnmarshalJSON(rawrespbody, &response)

		// return token and expiry date
		return response.Token, response.Expire
	}

	t.Run("successful login", func(t *testing.T) {
		_, _ = loginflow(registerflow(t))
	})

	t.Run("login with invalid password", func(t *testing.T) {
		// obtain the credentials
		username, _ := registerflow(t)

		// obtain the response using a completely different password
		resp := loginrequest(username, "antani")

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see 401
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("get psiphon config with invalid method", func(t *testing.T) {
		// obtain the token
		token, _ := loginflow(registerflow(t))

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"DELETE",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/psiphon-config", "")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see not implemented
		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("get tor targets with invalid method", func(t *testing.T) {
		// obtain the token
		token, _ := loginflow(registerflow(t))

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"DELETE",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/tor-targets", "")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see not implemented
		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("get psiphon config with invalid token", func(t *testing.T) {
		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/psiphon-config", "")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "antani"))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see 401
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("get psiphon config with expired token", func(t *testing.T) {
		// obtain the credentials
		username, password := registerflow(t)

		// obtain the token
		token, _ := loginflow(username, password)

		// modify the token expiry time so that it's expired
		state.DoWithLockedUserRecord(username, func(rec *OONIBackendWithLoginFlowUserRecord) error {
			rec.Expire = time.Now().Add(-1 * time.Hour)
			return nil
		})

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/psiphon-config", "")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see 401
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})

	t.Run("we can get psiphon config", func(t *testing.T) {
		// define the expected body
		expectedbody := []byte(`bonsoir elliot`)

		// set the config
		state.SetPsiphonConfig(expectedbody)

		// obtain the credentials
		username, password := registerflow(t)

		// obtain the token
		token, _ := loginflow(username, password)

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/psiphon-config", "")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", resp.StatusCode)
		}

		// read the full body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))

		// make sure we've got the expected body
		if diff := cmp.Diff(expectedbody, rawrespbody); err != nil {
			t.Fatal(diff)
		}
	})

	t.Run("we can get tor targets", func(t *testing.T) {
		// define the expected body
		expectedbody := []byte(`bonsoir elliot`)

		// set the targets
		state.SetTorTargets(expectedbody)

		// obtain the credentials
		username, password := registerflow(t)

		// obtain the token
		token, _ := loginflow(username, password)

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/tor-targets", "country_code=IT")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", resp.StatusCode)
		}

		// read the full body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))

		// make sure we've got the expected body
		if diff := cmp.Diff(expectedbody, rawrespbody); err != nil {
			t.Fatal(diff)
		}
	})

	t.Run("we need query string to get tor targets", func(t *testing.T) {
		// define the expected body
		expectedbody := []byte(`bonsoir elliot`)

		// set the targets
		state.SetTorTargets(expectedbody)

		// obtain the credentials
		username, password := registerflow(t)

		// obtain the token
		token, _ := loginflow(username, password)

		// create HTTP request
		req := runtimex.Try1(http.NewRequest(
			"GET",
			runtimex.Try1(urlx.ResolveReference(server.URL, "/api/v1/test-list/tor-targets", "")),
			nil,
		))

		// create the authorization token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// perform the round trip
		resp, err := http.DefaultClient.Do(req)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we eventually close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})
}
