package main

import (
	"fmt"
	"strings"
	"time"
)

func (d *Descriptor) genNewLogin(sb *strings.Builder) {
	fmt.Fprintf(sb, "// %s implements login for %s.\n",
		d.WithLoginAPIStructName(), d.APIStructName())
	fmt.Fprintf(sb, "type %s struct {\n", d.WithLoginAPIStructName())
	fmt.Fprintf(sb, "\tAPI %s // mandatory\n", d.ClonerInterfaceName())
	fmt.Fprint(sb, "\tJSONCodec JSONCodec // optional\n")
	fmt.Fprint(sb, "\tKVStore KVStore // mandatory\n")
	fmt.Fprint(sb, "\tRegisterAPI RegisterCaller // mandatory\n")
	fmt.Fprint(sb, "\tLoginAPI LoginCaller // mandatory\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "// Call logins, if needed, then calls the API.\n")
	fmt.Fprintf(sb, "func (api *%s) Call(ctx context.Context, req %s) (%s, error) {\n",
		d.WithLoginAPIStructName(), d.RequestTypeName(), d.ResponseTypeName())
	fmt.Fprint(sb, "\ttoken, err := api.maybeLogin(ctx)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tresp, err := api.API.WithToken(token).Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif errors.Is(err, ErrUnauthorized) {\n")
	fmt.Fprint(sb, "\t\t// Maybe the clock is just off? Let's try to obtain\n")
	fmt.Fprint(sb, "\t\t// a token again and see if this fixes it.\n")
	fmt.Fprint(sb, "\t\tif token, err = api.forceLogin(ctx); err == nil {\n")
	fmt.Fprint(sb, "\t\t\tswitch resp, err = api.API.WithToken(token).Call(ctx, req); err {\n")
	fmt.Fprint(sb, "\t\t\tcase nil:\n")
	fmt.Fprint(sb, "\t\t\t\treturn resp, nil\n")
	fmt.Fprint(sb, "\t\t\tcase ErrUnauthorized:\n")
	fmt.Fprint(sb, "\t\t\t\t// fallthrough\n")
	fmt.Fprint(sb, "\t\t\tdefault:\n")
	fmt.Fprint(sb, "\t\t\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t\t\t}\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\t// Okay, this seems a broader problem. How about we try\n")
	fmt.Fprint(sb, "\t\t// and re-register ourselves again instead?\n")
	fmt.Fprint(sb, "\t\ttoken, err = api.forceRegister(ctx)\n")
	fmt.Fprint(sb, "\t\tif err != nil {\n")
	fmt.Fprint(sb, "\t\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tresp, err = api.API.WithToken(token).Call(ctx, req)\n")
	fmt.Fprint(sb, "\t\t// fallthrough\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn resp, nil\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) jsonCodec() JSONCodec {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\tif api.JSONCodec != nil {\n")
	fmt.Fprint(sb, "\t\treturn api.JSONCodec\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn &defaultJSONCodec{}\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) readstate() (*loginState, error) {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\tdata, err := api.KVStore.Get(loginKey)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tvar ls loginState\n")
	fmt.Fprint(sb, "\tif err := api.jsonCodec().Decode(data, &ls); err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn &ls, nil\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) writestate(ls *loginState) error {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\tdata, err := api.jsonCodec().Encode(*ls)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn api.KVStore.Set(loginKey, data)\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) doRegister(ctx context.Context, password string) (string, error) {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\treq := newRegisterRequest(password)\n")
	fmt.Fprint(sb, "\tls := &loginState{}\n")
	fmt.Fprint(sb, "\tresp, err := api.RegisterAPI.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn \"\", err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tls.ClientID = resp.ClientID\n")
	fmt.Fprint(sb, "\tls.Password = req.Password\n")
	fmt.Fprint(sb, "\treturn api.doLogin(ctx, ls)\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) forceRegister(ctx context.Context) (string, error) {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\tvar password string\n")
	fmt.Fprint(sb, "\t// If we already have a previous password, let us keep\n")
	fmt.Fprint(sb, "\t// using it. This will allow a new version of the API to\n")
	fmt.Fprint(sb, "\t// be able to continue to identify this probe. (This\n")
	fmt.Fprint(sb, "\t// assumes that we have a stateless API that generates\n")
	fmt.Fprint(sb, "\t// the user ID as a signature of the password plus a\n")
	fmt.Fprint(sb, "\t// timestamp and that the key to generate the signature\n")
	fmt.Fprint(sb, "\t// is not lost. If all these conditions are met, we\n")
	fmt.Fprint(sb, "\t// can then serve better test targets to more long running\n")
	fmt.Fprint(sb, "\t// (and therefore trusted) probes.)\n")
	fmt.Fprint(sb, "\tif ls, err := api.readstate(); err == nil {\n")
	fmt.Fprint(sb, "\t\tpassword = ls.Password\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif password == \"\" {\n")
	fmt.Fprint(sb, "\t\tpassword = newRandomPassword()\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn api.doRegister(ctx, password)\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) forceLogin(ctx context.Context) (string, error) {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\tls, err := api.readstate()\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn \"\", err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn api.doLogin(ctx, ls)\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) maybeLogin(ctx context.Context) (string, error) {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\tls, _ := api.readstate()\n")
	fmt.Fprint(sb, "\tif ls == nil || !ls.credentialsValid() {\n")
	fmt.Fprint(sb, "\t\treturn api.forceRegister(ctx)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif !ls.tokenValid() {\n")
	fmt.Fprint(sb, "\t\treturn api.doLogin(ctx, ls)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn ls.Token, nil\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (api *%s) doLogin(ctx context.Context, ls *loginState) (string, error) {\n",
		d.WithLoginAPIStructName())
	fmt.Fprint(sb, "\treq := &apimodel.LoginRequest{\n")
	fmt.Fprint(sb, "\t\tClientID: ls.ClientID,\n")
	fmt.Fprint(sb, "\t\tPassword: ls.Password,\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tresp, err := api.LoginAPI.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn \"\", err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tls.Token = resp.Token\n")
	fmt.Fprint(sb, "\tls.Expire = resp.Expire\n")
	fmt.Fprint(sb, "\tif err := api.writestate(ls); err != nil {\n")
	fmt.Fprint(sb, "\t\treturn \"\", err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn ls.Token, nil\n")
	fmt.Fprint(sb, "}\n\n")
	fmt.Fprintf(sb, "var _ %s = &%s{}\n\n", d.CallerInterfaceName(),
		d.WithLoginAPIStructName())
}

// GenLoginGo generates login.go.
func GenLoginGo(file string) {
	var sb strings.Builder
	fmt.Fprint(&sb, "// Code generated by go generate; DO NOT EDIT.\n")
	fmt.Fprintf(&sb, "// %s\n\n", time.Now())
	fmt.Fprint(&sb, "package ooapi\n\n")
	fmt.Fprintf(&sb, "//go:generate go run ./internal/generator -file %s\n\n", file)
	fmt.Fprint(&sb, "import (\n")
	fmt.Fprint(&sb, "\t\"context\"\n")
	fmt.Fprint(&sb, "\t\"errors\"\n")
	fmt.Fprint(&sb, "\n")
	fmt.Fprint(&sb, "\t\"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel\"\n")
	fmt.Fprint(&sb, ")\n")
	for _, desc := range Descriptors {
		if !desc.RequiresLogin {
			continue
		}
		desc.genNewLogin(&sb)
	}
	writefile(file, &sb)
}
