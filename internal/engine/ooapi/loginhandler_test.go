package ooapi

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

// LoginHandler is an http.Handler to test login
type LoginHandler struct {
	failCallWith []int // ignored by login and register
	mu           sync.Mutex
	noRegister   bool
	state        []*loginState
	t            *testing.T
	logins       int32
	registers    int32
}

func (lh *LoginHandler) forgetLogins() {
	defer lh.mu.Unlock()
	lh.mu.Lock()
	lh.state = nil
}

func (lh *LoginHandler) forgetTokens() {
	defer lh.mu.Unlock()
	lh.mu.Lock()
	for _, entry := range lh.state {
		// This should be enough to cause all tokens to
		// be expired and force clients to relogin.
		//
		// (It does not matter much whether the client
		// clock is off, or the server clock is off,
		// thanks Galileo for explaining this to us <3.)
		entry.Expire = time.Now().Add(-3600 * time.Second)
	}
}

func (lh *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implementation note: we don't check for the method
	// for simplicity since it's already tested.
	switch r.URL.Path {
	case "/api/v1/register":
		atomic.AddInt32(&lh.registers, 1)
		lh.register(w, r)
	case "/api/v1/login":
		atomic.AddInt32(&lh.logins, 1)
		lh.login(w, r)
	case "/api/v1/test-list/psiphon-config":
		lh.psiphon(w, r)
	case "/api/v1/test-list/tor-targets":
		lh.tor(w, r)
	default:
		w.WriteHeader(500)
	}
}

func (lh *LoginHandler) register(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		w.WriteHeader(400)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	var req apimodel.RegisterRequest
	if err := json.Unmarshal(data, &req); err != nil {
		w.WriteHeader(400)
		return
	}
	if req.Password == "" {
		w.WriteHeader(400)
		return
	}
	defer lh.mu.Unlock()
	lh.mu.Lock()
	if lh.noRegister {
		// We have been asked to stop registering clients so
		// we're going to make a boo boo.
		w.WriteHeader(500)
		return
	}
	var resp apimodel.RegisterResponse
	ff := &fakeFill{}
	ff.fill(&resp)
	lh.state = append(lh.state, &loginState{
		ClientID: resp.ClientID, Password: req.Password})
	data, err = json.Marshal(&resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	lh.t.Logf("register: %+v", string(data))
	w.Write(data)
}

func (lh *LoginHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		w.WriteHeader(400)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	var req apimodel.LoginRequest
	if err := json.Unmarshal(data, &req); err != nil {
		w.WriteHeader(400)
		return
	}
	defer lh.mu.Unlock()
	lh.mu.Lock()
	for _, s := range lh.state {
		if req.ClientID == s.ClientID && req.Password == s.Password {
			var resp apimodel.LoginResponse
			ff := &fakeFill{}
			ff.fill(&resp)
			// We want the token to be many seconds in the future while
			// ff.fill only sets the tokent to now plus a small delta.
			resp.Expire = time.Now().Add(3600 * time.Second)
			s.Expire = resp.Expire
			s.Token = resp.Token
			data, err = json.Marshal(&resp)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			lh.t.Logf("login: %+v", string(data))
			w.Write(data)
			return
		}
	}
	lh.t.Log("login: 401")
	w.WriteHeader(401)
}

func (lh *LoginHandler) psiphon(w http.ResponseWriter, r *http.Request) {
	defer lh.mu.Unlock()
	lh.mu.Lock()
	if len(lh.failCallWith) > 0 {
		code := lh.failCallWith[0]
		lh.failCallWith = lh.failCallWith[1:]
		w.WriteHeader(code)
		return
	}
	token := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)
	for _, s := range lh.state {
		if token == s.Token && time.Now().Before(s.Expire) {
			var resp apimodel.PsiphonConfigResponse
			ff := &fakeFill{}
			ff.fill(&resp)
			data, err := json.Marshal(&resp)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			lh.t.Logf("psiphon: %+v", string(data))
			w.Write(data)
			return
		}
	}
	lh.t.Log("psiphon: 401")
	w.WriteHeader(401)
}

func (lh *LoginHandler) tor(w http.ResponseWriter, r *http.Request) {
	defer lh.mu.Unlock()
	lh.mu.Lock()
	if len(lh.failCallWith) > 0 {
		code := lh.failCallWith[0]
		lh.failCallWith = lh.failCallWith[1:]
		w.WriteHeader(code)
		return
	}
	token := strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)
	for _, s := range lh.state {
		if token == s.Token && time.Now().Before(s.Expire) {
			var resp apimodel.TorTargetsResponse
			ff := &fakeFill{}
			ff.fill(&resp)
			data, err := json.Marshal(&resp)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			lh.t.Logf("tor: %+v", string(data))
			w.Write(data)
			return
		}
	}
	lh.t.Log("tor: 401")
	w.WriteHeader(401)
}
