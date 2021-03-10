package main

import (
	"fmt"
	"strings"
	"time"
)

func (d *Descriptor) genTestClientCallRoundTrip(sb *strings.Builder) {
	// generate the type of the handler
	fmt.Fprintf(sb, "type handleClientCall%s struct {\n", d.Name)
	fmt.Fprint(sb, "\taccept string\n")
	fmt.Fprint(sb, "\tbody []byte\n")
	fmt.Fprint(sb, "\tcontentType string\n")
	fmt.Fprint(sb, "\tcount int32\n")
	fmt.Fprint(sb, "\tmethod string\n")
	fmt.Fprint(sb, "\tmu sync.Mutex\n")
	fmt.Fprintf(sb, "\tresp %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\turl *url.URL\n")
	fmt.Fprint(sb, "\tuserAgent string\n")
	fmt.Fprint(sb, "}\n\n")

	// generate the handling function
	fmt.Fprintf(sb,
		"func (h *handleClientCall%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {",
		d.Name)
	fmt.Fprint(sb, "\tff := fakeFill{}\n")
	if d.RequiresLogin {
		fmt.Fprintf(sb, "\tif r.URL.Path == \"/api/v1/register\" {\n")
		fmt.Fprintf(sb, "\t\tvar out apimodel.RegisterResponse\n")
		fmt.Fprintf(sb, "\t\tff.fill(&out)\n")
		fmt.Fprintf(sb, "\t\tdata, err := json.Marshal(out)\n")
		fmt.Fprintf(sb, "\t\tif err != nil {\n")
		fmt.Fprintf(sb, "\t\t\tw.WriteHeader(400)\n")
		fmt.Fprintf(sb, "\t\t\treturn\n")
		fmt.Fprintf(sb, "\t\t}\n")
		fmt.Fprintf(sb, "\t\tw.Write(data)\n")
		fmt.Fprintf(sb, "\t\treturn\n")
		fmt.Fprintf(sb, "\t}\n")
		fmt.Fprintf(sb, "\tif r.URL.Path == \"/api/v1/login\" {\n")
		fmt.Fprintf(sb, "\t\tvar out apimodel.LoginResponse\n")
		fmt.Fprintf(sb, "\t\tff.fill(&out)\n")
		fmt.Fprintf(sb, "\t\tdata, err := json.Marshal(out)\n")
		fmt.Fprintf(sb, "\t\tif err != nil {\n")
		fmt.Fprintf(sb, "\t\t\tw.WriteHeader(400)\n")
		fmt.Fprintf(sb, "\t\t\treturn\n")
		fmt.Fprintf(sb, "\t\t}\n")
		fmt.Fprintf(sb, "\t\tw.Write(data)\n")
		fmt.Fprintf(sb, "\t\treturn\n")
		fmt.Fprintf(sb, "\t}\n")
	}
	fmt.Fprint(sb, "\tdefer h.mu.Unlock()\n")
	fmt.Fprint(sb, "\th.mu.Lock()\n")
	fmt.Fprint(sb, "\tif h.count > 0 {\n")
	fmt.Fprint(sb, "\t\tw.WriteHeader(400)\n")
	fmt.Fprint(sb, "\t\treturn\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\th.count++\n")
	fmt.Fprint(sb, "\tif r.Body != nil {\n")
	fmt.Fprint(sb, "\t\tdata, err := ioutil.ReadAll(r.Body)\n")
	fmt.Fprint(sb, "\t\tif err != nil {\n")
	fmt.Fprintf(sb, "\t\t\tw.WriteHeader(400)\n")
	fmt.Fprintf(sb, "\t\t\treturn\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\th.body = data\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\th.method = r.Method\n")
	fmt.Fprint(sb, "\th.url = r.URL\n")
	fmt.Fprint(sb, "\th.accept = r.Header.Get(\"Accept\")\n")
	fmt.Fprint(sb, "\th.contentType = r.Header.Get(\"Content-Type\")\n")
	fmt.Fprint(sb, "\th.userAgent = r.Header.Get(\"User-Agent\")\n")
	fmt.Fprintf(sb, "\tvar out %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "\tff.fill(&out)\n")
	fmt.Fprintf(sb, "\th.resp = out\n")
	fmt.Fprintf(sb, "\tdata, err := json.Marshal(out)\n")
	fmt.Fprintf(sb, "\tif err != nil {\n")
	fmt.Fprintf(sb, "\t\tw.WriteHeader(400)\n")
	fmt.Fprintf(sb, "\t\treturn\n")
	fmt.Fprintf(sb, "\t}\n")
	fmt.Fprintf(sb, "\tw.Write(data)\n")
	fmt.Fprintf(sb, "\t}\n\n")

	// generate the test itself
	fmt.Fprintf(sb, "func Test%sClientCallRoundTrip(t *testing.T) {\n", d.Name)

	fmt.Fprint(sb, "\t// setup\n")
	fmt.Fprintf(sb, "\thandler := &handleClientCall%s{}\n", d.Name)
	fmt.Fprint(sb, "\tsrvr := httptest.NewServer(handler)\n")
	fmt.Fprint(sb, "\tdefer srvr.Close()\n")
	fmt.Fprintf(sb, "\treq := &%s{}\n", d.RequestTypeNameAsStruct())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprint(sb, "\tclnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}\n")
	fmt.Fprint(sb, "\tff.fill(&clnt.UserAgent)\n")

	fmt.Fprint(sb, "\t// issue request\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprintf(sb, "\tresp, err := clnt.%s(ctx, req)\n", d.Name)
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp == nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected non-nil response here\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t// compare our response and server's one\n")
	fmt.Fprint(sb, "\tif diff := cmp.Diff(handler.resp, resp); diff != \"\" {")
	fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t// check whether headers are OK\n")
	fmt.Fprint(sb, "\tif handler.accept != \"application/json\" {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid accept header\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif handler.userAgent != clnt.UserAgent {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid user-agent header\")\n")
	fmt.Fprint(sb, "\t}\n")

	fmt.Fprint(sb, "\t// check whether the method is OK\n")
	fmt.Fprintf(sb, "\tif handler.method != \"%s\" {\n", d.Method)
	fmt.Fprint(sb, "\t\tt.Fatal(\"invalid method\")\n")
	fmt.Fprint(sb, "\t}\n")

	if d.Method == "POST" {
		fmt.Fprint(sb, "\t// check the body\n")
		fmt.Fprint(sb, "\tif handler.contentType != \"application/json\" {\n")
		fmt.Fprint(sb, "\t\tt.Fatal(\"invalid content-type header\")\n")
		fmt.Fprint(sb, "\t}\n")
		fmt.Fprintf(sb, "\tgot := &%s{}\n", d.RequestTypeNameAsStruct())
		fmt.Fprintf(sb, "\tif err := json.Unmarshal(handler.body, &got); err != nil {\n")
		fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
		fmt.Fprint(sb, "\t}\n")
		fmt.Fprint(sb, "\tif diff := cmp.Diff(req, got); diff != \"\" {\n")
		fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
		fmt.Fprint(sb, "\t}\n")
	} else {
		fmt.Fprint(sb, "\t// check the query\n")
		fmt.Fprintf(sb, "\tapi := &%s{BaseURL: srvr.URL}\n", d.APIStructName())
		fmt.Fprint(sb, "\thttpReq, err := api.newRequest(context.Background(), req)\n")
		fmt.Fprint(sb, "\tif err != nil {\n")
		fmt.Fprint(sb, "\t\tt.Fatal(err)\n")
		fmt.Fprint(sb, "\t}\n")
		fmt.Fprint(sb, "\tif diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != \"\" {\n")
		fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
		fmt.Fprint(sb, "\t}\n")
		fmt.Fprint(sb, "\tif diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != \"\" {\n")
		fmt.Fprint(sb, "\t\tt.Fatal(diff)\n")
		fmt.Fprint(sb, "\t}\n")
	}

	fmt.Fprint(sb, "}\n\n")
}

// GenClientCallTestGo generates clientcall_test.go.
func GenClientCallTestGo(file string) {
	var sb strings.Builder
	fmt.Fprint(&sb, "// Code generated by go generate; DO NOT EDIT.\n")
	fmt.Fprintf(&sb, "// %s\n\n", time.Now())
	fmt.Fprint(&sb, "package ooapi\n\n")
	fmt.Fprintf(&sb, "//go:generate go run ./internal/generator -file %s\n\n", file)
	fmt.Fprint(&sb, "import (\n")
	fmt.Fprint(&sb, "\t\"context\"\n")
	fmt.Fprint(&sb, "\t\"encoding/json\"\n")
	fmt.Fprint(&sb, "\t\"io/ioutil\"\n")
	fmt.Fprint(&sb, "\t\"net/http/httptest\"\n")
	fmt.Fprint(&sb, "\t\"net/http\"\n")
	fmt.Fprint(&sb, "\t\"net/url\"\n")
	fmt.Fprint(&sb, "\t\"testing\"\n")
	fmt.Fprint(&sb, "\t\"sync\"\n")
	fmt.Fprint(&sb, "\n")
	fmt.Fprint(&sb, "\t\"github.com/google/go-cmp/cmp\"\n")
	fmt.Fprint(&sb, "\t\"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel\"\n")
	fmt.Fprint(&sb, ")\n")
	for _, desc := range Descriptors {
		if desc.Name == "Login" || desc.Name == "Register" {
			continue // they cannot be called directly
		}
		desc.genTestClientCallRoundTrip(&sb)
	}
	writefile(file, &sb)
}
