package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func (d *Descriptor) genTestNewRequest(sb *strings.Builder) {
	fmt.Fprintf(sb, "\treq := &%s{}\n", d.RequestTypeNameAsStruct())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprint(sb, "\tff.fill(req)\n")
}

func (d *Descriptor) genTestInvalidURL(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sInvalidURL(t *testing.T) {\n", d.Name)
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tBaseURL: \"\\t\", // invalid\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err == nil || !strings.HasSuffix(err.Error(), \"invalid control character in URL\") {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithMissingToken(sb *strings.Builder) {
	if d.RequiresLogin == false {
		return // does not make sense when login isn't required
	}
	fmt.Fprintf(sb, "func Test%sWithMissingToken(t *testing.T) {\n", d.Name)
	fmt.Fprintf(sb, "\tapi := &%s{} // no token\n", d.APIStructName())
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, ErrMissingToken) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithHTTPErr(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithHTTPErr(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Err: errMocked}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestMarshalErr(sb *strings.Builder) {
	if d.Method != "POST" {
		return // does not make sense when we don't send a request body
	}
	fmt.Fprintf(sb, "func Test%sMarshalErr(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tJSONCodec: &FakeCodec{EncodeErr: errMocked},\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithNewRequestErr(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithNewRequestErr(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tRequestMaker: &FakeRequestMaker{Err: errMocked},\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWith401(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWith401(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{StatusCode: 401}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, ErrUnauthorized) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWith400(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWith400(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{StatusCode: 400}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, ErrHTTPFailure) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithResponseBodyReadErr(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithResponseBodyReadErr(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{\n")
	fmt.Fprint(sb, "\t\tStatusCode: 200,\n")
	fmt.Fprint(sb, "\t\tBody: &FakeBody{Err: errMocked},\n")
	fmt.Fprint(sb, "\t}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestWithUnmarshalFailure(sb *strings.Builder) {
	fmt.Fprintf(sb, "func Test%sWithUnmarshalFailure(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{\n")
	fmt.Fprint(sb, "\t\tStatusCode: 200,\n")
	fmt.Fprint(sb, "\t\tBody: &FakeBody{Data: []byte(`{}`)},\n")
	fmt.Fprint(sb, "\t}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	fmt.Fprintf(sb, "\t\tJSONCodec: &FakeCodec{DecodeErr: errMocked},\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprintf(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestRoundTrip(sb *strings.Builder) {
	// generate the type of the handler
	fmt.Fprintf(sb, "type handle%s struct {\n", d.Name)
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
		"func (h *handle%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {",
		d.Name)
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
	fmt.Fprint(sb, "\tff := fakeFill{}\n")
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
	fmt.Fprintf(sb, "func Test%sRoundTrip(t *testing.T) {\n", d.Name)

	fmt.Fprint(sb, "\t// setup\n")
	fmt.Fprintf(sb, "\thandler := &handle%s{}\n", d.Name)
	fmt.Fprint(sb, "\tsrvr := httptest.NewServer(handler)\n")
	fmt.Fprint(sb, "\tdefer srvr.Close()\n")
	fmt.Fprintf(sb, "\treq := &%s{}\n", d.RequestTypeNameAsStruct())
	fmt.Fprint(sb, "\tff := &fakeFill{}\n")
	fmt.Fprint(sb, "\tff.fill(&req)\n")
	fmt.Fprintf(sb, "\tapi := &%s{BaseURL: srvr.URL}\n", d.APIStructName())
	fmt.Fprint(sb, "\tff.fill(&api.UserAgent)\n")
	if d.RequiresLogin {
		fmt.Fprint(sb, "\tff.fill(&api.Token)\n")
	}

	fmt.Fprint(sb, "\t// issue request\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
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
	fmt.Fprint(sb, "\tif handler.userAgent != api.UserAgent {\n")
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

func (d *Descriptor) genTestResponseLiteralNull(sb *strings.Builder) {
	switch d.ResponseTypeKind() {
	case reflect.Map:
		// fallthrough
	case reflect.Struct:
		return // test not applicable
	}
	fmt.Fprintf(sb, "func Test%sResponseLiteralNull(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{\n")
	fmt.Fprint(sb, "\t\tStatusCode: 200,\n")
	fmt.Fprint(sb, "\t\tBody: &FakeBody{Data: []byte(`null`)},\n")
	fmt.Fprint(sb, "\t}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, ErrJSONLiteralNull) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestMandatoryFields(sb *strings.Builder) {
	fields := d.StructFieldsWithTag(d.Request, tagForRequired)
	if len(fields) < 1 {
		return // nothing to test
	}
	fmt.Fprintf(sb, "func Test%sMandatoryFields(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{\n")
	fmt.Fprint(sb, "\t\tStatusCode: 500,\n")
	fmt.Fprint(sb, "\t}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	fmt.Fprintf(sb, "\treq := &%s{} // deliberately empty\n", d.RequestTypeNameAsStruct())
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, ErrEmptyField) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

func (d *Descriptor) genTestTemplateErr(sb *strings.Builder) {
	if !d.URLPath.IsTemplate {
		return // nothing to test
	}
	fmt.Fprintf(sb, "func Test%sTemplateErr(t *testing.T) {\n", d.Name)
	fmt.Fprint(sb, "\terrMocked := errors.New(\"mocked error\")\n")
	fmt.Fprint(sb, "\tclnt := &FakeHTTPClient{Resp: &http.Response{\n")
	fmt.Fprint(sb, "\t\tStatusCode: 500,\n")
	fmt.Fprint(sb, "\t}}\n")
	fmt.Fprintf(sb, "\tapi := &%s{\n", d.APIStructName())
	fmt.Fprint(sb, "\t\tHTTPClient: clnt,\n")
	if d.RequiresLogin == true {
		fmt.Fprint(sb, "\t\tToken:      \"fakeToken\",\n")
	}
	fmt.Fprint(sb, "\t\tTemplateExecutor: &FakeTemplateExecutor{Err: errMocked},\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tctx := context.Background()\n")
	d.genTestNewRequest(sb)
	fmt.Fprint(sb, "\tresp, err := api.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif !errors.Is(err, errMocked) {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"not the error we expected\", err)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif resp != nil {\n")
	fmt.Fprint(sb, "\t\tt.Fatal(\"expected nil resp\")\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "}\n\n")
}

// TODO(bassosimone): we should add a panic for every switch for
// the type of a request or a response for robustness.

func (d *Descriptor) genAPITests(sb *strings.Builder) {
	d.genTestInvalidURL(sb)
	d.genTestWithMissingToken(sb)
	d.genTestWithHTTPErr(sb)
	d.genTestMarshalErr(sb)
	d.genTestWithNewRequestErr(sb)
	d.genTestWith401(sb)
	d.genTestWith400(sb)
	d.genTestWithResponseBodyReadErr(sb)
	d.genTestWithUnmarshalFailure(sb)
	d.genTestRoundTrip(sb)
	d.genTestResponseLiteralNull(sb)
	d.genTestMandatoryFields(sb)
	d.genTestTemplateErr(sb)
}

// GenAPIsTestGo generates apis_test.go.
func GenAPIsTestGo() {
	var sb strings.Builder
	fmt.Fprint(&sb, "// Code generated by go generate; DO NOT EDIT.\n")
	fmt.Fprintf(&sb, "// %s\n\n", time.Now())
	fmt.Fprint(&sb, "package ooapi\n\n")
	fmt.Fprint(&sb, "//go:generate go run ./internal/generator\n\n")
	fmt.Fprint(&sb, "import (\n")
	fmt.Fprint(&sb, "\t\"context\"\n")
	fmt.Fprint(&sb, "\t\"encoding/json\"\n")
	fmt.Fprint(&sb, "\t\"errors\"\n")
	fmt.Fprint(&sb, "\t\"io/ioutil\"\n")
	fmt.Fprint(&sb, "\t\"net/http/httptest\"\n")
	fmt.Fprint(&sb, "\t\"net/http\"\n")
	fmt.Fprint(&sb, "\t\"net/url\"\n")
	fmt.Fprint(&sb, "\t\"strings\"\n")
	fmt.Fprint(&sb, "\t\"testing\"\n")
	fmt.Fprint(&sb, "\t\"sync\"\n")
	fmt.Fprint(&sb, "\n")
	fmt.Fprint(&sb, "\t\"github.com/google/go-cmp/cmp\"\n")
	fmt.Fprint(&sb, "\t\"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel\"\n")
	fmt.Fprint(&sb, ")\n")
	for _, desc := range Descriptors {
		desc.genAPITests(&sb)
	}
	writefile("apis_test.go", &sb)
}
