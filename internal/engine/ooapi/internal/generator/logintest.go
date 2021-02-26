package main

import (
	"fmt"
	"strings"
	"time"
)

func (d *Descriptor) genTestRegisterAndLoginSuccess(sb *strings.Builder) {
	fmt.Fprintf(sb, "func TestRegisterAndLogin%sSuccess(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tResponse: &apimodel.RegisterResponse{\n")
	fmt.Fprint(sb, "\t\t\tClientID: \"antani-antani\",\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t\tloginAPI := &FakeLoginAPI{\n")
	fmt.Fprint(sb, "\t\t\tResponse: &apimodel.LoginResponse{\n")
	fmt.Fprint(sb, "\t\t\t\tExpire: time.Now().Add(3600*time.Second),\n")
	fmt.Fprint(sb, "\t\t\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tResponse: expect,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif diff := cmp.Diff(expect, resp); diff != \"\" {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestContinueUsingToken(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sContinueUsingToken(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tResponse: &apimodel.RegisterResponse{\n")
	fmt.Fprint(sb, "\t\t\tClientID: \"antani-antani\",\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t\tloginAPI := &FakeLoginAPI{\n")
	fmt.Fprint(sb, "\t\t\tResponse: &apimodel.LoginResponse{\n")
	fmt.Fprint(sb, "\t\t\t\tExpire: time.Now().Add(3600*time.Second),\n")
	fmt.Fprint(sb, "\t\t\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tResponse: expect,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")

	fmt.Fprint(sb, "\t// step 1: we register and login and use the token\n")
	fmt.Fprint(sb, "\t// inside a scope just to avoid mistakes\n")

	fmt.Fprint(sb, "\t{\n")
	fmt.Fprint(sb, "\t\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\t\tif err != nil {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tif diff := cmp.Diff(expect, resp); diff != \"\" {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(diff)\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\t\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\t\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t// step 2: we disable register and login but we\n")
	fmt.Fprint(sb, "\t// should be okay because of the token\n")
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tregisterAPI.Err = errMocked\n")
	fmt.Fprint(sb, "\tregisterAPI.Response = nil\n")
	fmt.Fprint(sb, "\tloginAPI.Err = errMocked\n")
	fmt.Fprint(sb, "\tloginAPI.Response = nil\n")

	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif diff := cmp.Diff(expect, resp); diff != \"\" {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithValidButExpiredToken(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithValidButExpiredToken(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tErr: errMocked,\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t\tloginAPI := &FakeLoginAPI{\n")
	fmt.Fprint(sb, "\t\t\tResponse: &apimodel.LoginResponse{\n")
	fmt.Fprint(sb, "\t\t\t\tExpire: time.Now().Add(3600*time.Second),\n")
	fmt.Fprint(sb, "\t\t\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tResponse: expect,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tls := &loginState{\n")
	fmt.Fprintf(sb, "\t\tClientID: \"antani-antani\",\n")
	fmt.Fprintf(sb, "\t\tExpire: time.Now().Add(-5 * time.Second),\n")
	fmt.Fprintf(sb, "\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprintf(sb, "\t\tPassword: \"antani-antani-password\",\n")
	fmt.Fprintf(sb, "\t}\n")
	fmt.Fprintf(sb, "\tif err := login.writestate(ls); err != nil {\n")
	fmt.Fprintf(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprintf(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif diff := cmp.Diff(expect, resp); diff != \"\" {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 0 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithRegisterAPIError(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithRegisterAPIError(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tErr: errMocked,\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tResponse: expect,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil response\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithLoginFailure(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithLoginFailure(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tResponse: &apimodel.RegisterResponse{\n")
	fmt.Fprint(sb, "\t\t\tClientID: \"antani-antani\",\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\t\tloginAPI := &FakeLoginAPI{\n")
	fmt.Fprint(sb, "\t\t\tErr: errMocked,\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tResponse: expect,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil response\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestRegisterAndLoginThenFail(sb *strings.Builder) {
	fmt.Fprintf(sb, "func TestRegisterAndLogin%sThenFail(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tResponse: &apimodel.RegisterResponse{\n")
	fmt.Fprint(sb, "\t\t\tClientID: \"antani-antani\",\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t\tloginAPI := &FakeLoginAPI{\n")
	fmt.Fprint(sb, "\t\t\tResponse: &apimodel.LoginResponse{\n")
	fmt.Fprint(sb, "\t\t\t\tExpire: time.Now().Add(3600*time.Second),\n")
	fmt.Fprint(sb, "\t\t\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tErr: errMocked,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil response\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestTheDatabaseIsReplaced(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sTheDatabaseIsReplaced(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprint(sb, "\thandler := &LoginHandler{t: t}\n")
	fmt.Fprint(sb, "\tsrvr := httptest.NewServer(handler)\n")
	fmt.Fprint(sb, "\tdefer srvr.Close()\n")

	fmt.Fprint(sb, "\tregisterAPI := &RegisterAPI{\n")
	fmt.Fprint(sb, "\t\tHTTPClient: &VerboseHTTPClient{t: t},\n")
	fmt.Fprint(sb, "\t\tBaseURL: srvr.URL,\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\t\tloginAPI := &LoginAPI{\n")
	fmt.Fprint(sb, "\t\tHTTPClient: &VerboseHTTPClient{t: t},\n")
	fmt.Fprint(sb, "\t\tBaseURL: srvr.URL,\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprintf(sb, "\tbaseAPI := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: &VerboseHTTPClient{t: t},\n")
	fmt.Fprint(sb, "\t\tBaseURL: srvr.URL,\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\tAPI : baseAPI,\n")
	fmt.Fprint(sb, "\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")

	fmt.Fprint(sb, "\t// step 1: we register and login and use the token\n")
	fmt.Fprint(sb, "\t// inside a scope just to avoid mistakes\n")

	fmt.Fprint(sb, "\t{\n")
	fmt.Fprint(sb, "\t\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\t\tif err != nil {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\t\tif handler.logins != 1 {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"invalid handler.logins\")\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\t\tif handler.registers != 1 {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"invalid handler.registers\")\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t// step 2: we forget accounts and try again.\n")
	fmt.Fprint(sb, "\thandler.forgetLogins()\n")

	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif handler.logins != 3 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid handler.logins\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif handler.registers != 2 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid handler.registers\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestTheDatabaseIsReplacedThenFailure(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sTheDatabaseIsReplacedThenFailure(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprint(sb, "\thandler := &LoginHandler{t: t}\n")
	fmt.Fprint(sb, "\tsrvr := httptest.NewServer(handler)\n")
	fmt.Fprint(sb, "\tdefer srvr.Close()\n")

	fmt.Fprint(sb, "\tregisterAPI := &RegisterAPI{\n")
	fmt.Fprint(sb, "\t\tHTTPClient: &VerboseHTTPClient{t: t},\n")
	fmt.Fprint(sb, "\t\tBaseURL: srvr.URL,\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\t\tloginAPI := &LoginAPI{\n")
	fmt.Fprint(sb, "\t\tHTTPClient: &VerboseHTTPClient{t: t},\n")
	fmt.Fprint(sb, "\t\tBaseURL: srvr.URL,\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprintf(sb, "\tbaseAPI := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: &VerboseHTTPClient{t: t},\n")
	fmt.Fprint(sb, "\t\tBaseURL: srvr.URL,\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\tAPI : baseAPI,\n")
	fmt.Fprint(sb, "\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")

	fmt.Fprint(sb, "\t// step 1: we register and login and use the token\n")
	fmt.Fprint(sb, "\t// inside a scope just to avoid mistakes\n")

	fmt.Fprint(sb, "\t{\n")
	fmt.Fprint(sb, "\t\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\t\tif err != nil {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"expected non-nil response\")\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\t\tif handler.logins != 1 {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"invalid handler.logins\")\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\t\tif handler.registers != 1 {\n")
	fmt.Fprint(sb, "\t\t\tt.Fatal(\"invalid handler.registers\")\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t// step 2: we forget accounts and try again.\n")
	fmt.Fprint(sb, "\t// but registrations are also failing.\n")
	fmt.Fprint(sb, "\thandler.forgetLogins()\n")
	fmt.Fprint(sb, "\thandler.noRegister = true\n")

	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, ErrHTTPFailure) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil response\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif handler.logins != 2 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid handler.logins\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif handler.registers != 2 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid handler.registers\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}
func (d *Descriptor) genTestRegisterAndLoginCannotWriteState(sb *strings.Builder) {
	fmt.Fprintf(sb, "func TestRegisterAndLogin%sCannotWriteState(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\tregisterAPI := &FakeRegisterAPI{\n")
	fmt.Fprint(sb, "\t\tResponse: &apimodel.RegisterResponse{\n")
	fmt.Fprint(sb, "\t\t\tClientID: \"antani-antani\",\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t\tloginAPI := &FakeLoginAPI{\n")
	fmt.Fprint(sb, "\t\t\tResponse: &apimodel.LoginResponse{\n")
	fmt.Fprint(sb, "\t\t\t\tExpire: time.Now().Add(3600*time.Second),\n")
	fmt.Fprint(sb, "\t\t\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t}\n")

	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\t\tAPI: &Fake%s{\n", d.APIStructName())
	fmt.Fprintf(sb, "\t\t\tWithResult: &Fake%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\t\t\tResponse: expect,\n")
	fmt.Fprint(sb, "\t\t\t},\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t\tRegisterAPI: registerAPI,\n")
	fmt.Fprint(sb, "\t\tLoginAPI: loginAPI,\n")
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t\tJSONCodec: &FakeCodec{\n")
	fmt.Fprint(sb, "\t\t\tEncodeErr: errMocked,\n")
	fmt.Fprint(sb, "\t\t},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tvar req %s\n", d.RequestTypeName())
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := login.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil response\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif loginAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid loginAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\tif registerAPI.CountCall != 1 {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid registerAPI.CountCall\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestReadStateDecodeFailure(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sReadStateDecodeFailure(t *testing.T) {\n", d.APIStructName())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprintf(sb, "\tvar expect %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&expect)\n")

	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")

	fmt.Fprintf(sb, "\tlogin := &%s{\n", d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\t\tKVStore: &memkvstore{},\n")
	fmt.Fprint(sb, "\t\tJSONCodec: &FakeCodec{DecodeErr: errMocked},\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprintf(sb, "\tls := &loginState{\n")
	fmt.Fprintf(sb, "\t\tClientID: \"antani-antani\",\n")
	fmt.Fprintf(sb, "\t\tExpire: time.Now().Add(-5 * time.Second),\n")
	fmt.Fprintf(sb, "\t\tToken: \"antani-antani-token\",\n")
	fmt.Fprintf(sb, "\t\tPassword: \"antani-antani-password\",\n")
	fmt.Fprintf(sb, "\t}\n")
	fmt.Fprintf(sb, "\tif err := login.writestate(ls); err != nil {\n")
	fmt.Fprintf(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprintf(sb, "\t}\n")

	fmt.Fprintf(sb, "\tout, err := login.readstate()\n")
	fmt.Fprintf(sb, "if !errors.Is(err, errMocked) {\n")
	fmt.Fprintf(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprintf(sb, "\t}\n")
	fmt.Fprintf(sb, "if out != nil {\n")
	fmt.Fprintf(sb, "\t\tt.Fatal(\"expected nil here\")\n")
	fmt.Fprintf(sb, "\t}\n")

	fmt.Fprint(sb, "}\n\n")
}

// GenLoginTestGo generates login_test.go.
func GenLoginTestGo() {
	var sb strings.Builder
	fmt.Fprint(&sb, "// Code generated by go generate; DO NOT EDIT.\n")
	fmt.Fprintf(&sb, "// %s\n\n", time.Now())
	fmt.Fprint(&sb, "package ooapi\n\n")
	fmt.Fprint(&sb, "//go:generate go run ./internal/generator\n\n")
	fmt.Fprint(&sb, "import (\n")
	fmt.Fprint(&sb, "\t\"context\"\n")
	fmt.Fprint(&sb, "\t\"errors\"\n")
	fmt.Fprint(&sb, "\t\"net/http/httptest\"\n")
	fmt.Fprint(&sb, "\t\"testing\"\n")
	fmt.Fprint(&sb, "\t\"time\"\n")
	fmt.Fprint(&sb, "\n")
	fmt.Fprint(&sb, "\t\"github.com/google/go-cmp/cmp\"\n")
	fmt.Fprint(&sb, "\t\"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel\"\n")
	fmt.Fprint(&sb, ")\n")
	for _, desc := range Descriptors {
		if !desc.RequiresLogin {
			continue
		}
		desc.genTestRegisterAndLoginSuccess(&sb)
		desc.genTestContinueUsingToken(&sb)
		desc.genTestWithValidButExpiredToken(&sb)
		desc.genTestWithRegisterAPIError(&sb)
		desc.genTestWithLoginFailure(&sb)
		desc.genTestRegisterAndLoginThenFail(&sb)
		desc.genTestTheDatabaseIsReplaced(&sb)
		desc.genTestRegisterAndLoginCannotWriteState(&sb)
		desc.genTestReadStateDecodeFailure(&sb)
		desc.genTestTheDatabaseIsReplacedThenFailure(&sb)
	}
	writefile("login_test.go", &sb)
}
