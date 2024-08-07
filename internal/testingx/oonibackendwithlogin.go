package testingx

//
// Code for testing the OONI backend login flow.
//

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// OONIBackendWithLoginFlowUserRecord is a user record used by [OONIBackendWithLoginFlow].
type OONIBackendWithLoginFlowUserRecord struct {
	Expire   time.Time
	Password string
	Token    string
}

// OONIBackendWithLoginFlow implements the register and login workflows
// and serves the psiphon config and tor targets.
//
// The zero value is ready to use.
//
// This struct methods panics for several errors. Only use for testing purposes!
type OONIBackendWithLoginFlow struct {
	// logins maps the existing login names to the corresponding record.
	logins map[string]*OONIBackendWithLoginFlowUserRecord

	// mu provides mutual exclusion.
	mu sync.Mutex

	// openVPNConfig is the serialized openvpn config to send to clients.
	openVPNConfig []byte

	// psiphonConfig is the serialized psiphon config to send to authenticated clients.
	psiphonConfig []byte

	// tokens maps a token to a user record.
	tokens map[string]*OONIBackendWithLoginFlowUserRecord

	// torTargets is the serialized tor config to send to authenticated clients.
	torTargets []byte
}

// SetOpenVPNConfig sets openvpn configuration to use.
//
// This method is safe to call concurrently with incoming HTTP requests.
func (h *OONIBackendWithLoginFlow) SetOpenVPNConfig(config []byte) {
	defer h.mu.Unlock()
	h.mu.Lock()
	h.openVPNConfig = config
}

// SetPsiphonConfig sets psiphon configuration to use.
//
// This method is safe to call concurrently with incoming HTTP requests.
func (h *OONIBackendWithLoginFlow) SetPsiphonConfig(config []byte) {
	defer h.mu.Unlock()
	h.mu.Lock()
	h.psiphonConfig = config
}

// SetTorTargets sets tor targets to use.
//
// This method is safe to call concurrently with incoming HTTP requests.
func (h *OONIBackendWithLoginFlow) SetTorTargets(config []byte) {
	defer h.mu.Unlock()
	h.mu.Lock()
	h.torTargets = config
}

// DoWithLockedUserRecord performs an action with the given user record. The action will
// run while we're holding the [*OONIBackendWithLoginFlow] mutex.
func (h *OONIBackendWithLoginFlow) DoWithLockedUserRecord(
	username string, fx func(rec *OONIBackendWithLoginFlowUserRecord) error) error {
	defer h.mu.Unlock()
	h.mu.Lock()
	rec := h.logins[username]
	if rec == nil {
		return errors.New("no such record")
	}
	return fx(rec)
}

// NewMux constructs an [*http.ServeMux] configured with the correct routing.
func (h *OONIBackendWithLoginFlow) NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/register", h.handleRegister())
	mux.Handle("/api/v1/login", h.handleLogin())
	mux.Handle("/api/v1/test-list/psiphon-config", h.withAuthentication(h.handlePsiphonConfig()))
	mux.Handle("/api/v1/test-list/tor-targets", h.withAuthentication(h.handleTorTargets()))
	mux.Handle("/api/v2/ooniprobe/vpn-config/demovpn", h.handleOpenVPNConfig())
	return mux
}

func (h *OONIBackendWithLoginFlow) handleRegister() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the method is OK
		if r.Method != http.MethodPost {
			w.WriteHeader(501)
			return
		}

		// read the raw request body
		rawreqbody := runtimex.Try1(io.ReadAll(r.Body))

		// unmarshal the request
		var request model.OOAPIRegisterRequest
		must.UnmarshalJSON(rawreqbody, &request)

		// lock the users database
		h.mu.Lock()

		// make sure the map is usable
		if h.logins == nil {
			h.logins = make(map[string]*OONIBackendWithLoginFlowUserRecord)
		}

		// create new login
		userID := uuid.Must(uuid.NewRandom()).String()

		// save login
		h.logins[userID] = &OONIBackendWithLoginFlowUserRecord{
			Expire:   time.Time{},
			Password: request.Password,
			Token:    "",
		}

		// unlock the users database
		h.mu.Unlock()

		// prepare response
		response := &model.OOAPIRegisterResponse{
			ClientID: userID,
		}

		// send response
		_, _ = w.Write(must.MarshalJSON(response))
	})
}

func (h *OONIBackendWithLoginFlow) handleLogin() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the method is OK
		if r.Method != http.MethodPost {
			w.WriteHeader(501)
			return
		}

		// read the raw request body
		rawreqbody := runtimex.Try1(io.ReadAll(r.Body))

		// unmarshal the request
		var request model.OOAPILoginCredentials
		must.UnmarshalJSON(rawreqbody, &request)

		// lock the users database
		h.mu.Lock()

		// attempt to access user record
		record := h.logins[request.Username]

		// handle the case where the user does not exist
		if record == nil {
			// unlock the users database
			h.mu.Unlock()

			// return 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// handle the case where the password is invalid
		if request.Password != record.Password {
			// unlock the users database
			h.mu.Unlock()

			// return 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// create token
		token := uuid.Must(uuid.NewRandom()).String()

		// create expiry date
		expirydate := time.Now().Add(10 * time.Minute)

		// update record
		record.Token = token
		record.Expire = expirydate

		// create the token bearer header
		bearer := fmt.Sprintf("Bearer %s", token)

		// make sure the tokens map is okay
		if h.tokens == nil {
			h.tokens = make(map[string]*OONIBackendWithLoginFlowUserRecord)
		}

		// update the tokens map
		h.tokens[bearer] = record

		// unlock the users database
		h.mu.Unlock()

		// prepare response
		response := &model.OOAPILoginAuth{
			Expire: expirydate,
			Token:  token,
		}

		// send response
		_, _ = w.Write(must.MarshalJSON(response))
	})
}

func (h *OONIBackendWithLoginFlow) handleOpenVPNConfig() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the method is OK
		if r.Method != http.MethodGet {
			w.WriteHeader(501)
			return
		}

		// we must lock because of SetOpenVPNConfig
		h.mu.Lock()
		_, _ = w.Write(h.openVPNConfig)
		h.mu.Unlock()
	})
}

func (h *OONIBackendWithLoginFlow) handlePsiphonConfig() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the method is OK
		if r.Method != http.MethodGet {
			w.WriteHeader(501)
			return
		}

		// we must lock because of SetPsiphonConfig
		h.mu.Lock()
		_, _ = w.Write(h.psiphonConfig)
		h.mu.Unlock()
	})
}

func (h *OONIBackendWithLoginFlow) handleTorTargets() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the method is OK
		if r.Method != http.MethodGet {
			w.WriteHeader(501)
			return
		}

		// make sure the client has provided the right query string
		cc := r.URL.Query().Get("country_code")
		if cc == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// we must lock because of SetTorTargets
		h.mu.Lock()
		_, _ = w.Write(h.torTargets)
		h.mu.Unlock()
	})

}

func (h *OONIBackendWithLoginFlow) withAuthentication(child http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get the authorization header
		authorization := r.Header.Get("Authorization")

		// lock the users database
		h.mu.Lock()

		// check whether we have state
		record := h.tokens[authorization]

		// handle the case of nonexisting state
		if record == nil {
			// unlock the users database
			h.mu.Unlock()

			// return 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// handle the case of expired state
		if time.Until(record.Expire) <= 0 {
			// unlock the users database
			h.mu.Unlock()

			// return 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// unlock the users database
		h.mu.Unlock()

		// defer to the child handler
		child.ServeHTTP(w, r)
	})
}
