package ooapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

type VerboseHTTPClient struct {
	t *testing.T
}

func (c *VerboseHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.t.Logf("> %s %s", req.Method, req.URL.String())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Logf("< %s", err.Error())
		return nil, err
	}
	c.t.Logf("< %d", resp.StatusCode)
	return resp, nil
}

func TestWithRealServerDoCheckIn(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.CheckInRequest{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		ProbeASN:        "AS12353",
		ProbeCC:         "IT",
		RunType:         "timed",
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: apimodel.CheckInRequestWebConnectivity{
			CategoryCodes: []string{"NEWS", "CULTR"},
		},
	}
	httpClnt := &VerboseHTTPClient{t: t}
	api := &simpleCheckInAPI{
		HTTPClient: httpClnt,
	}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	for idx, url := range resp.Tests.WebConnectivity.URLs {
		if idx >= 3 {
			break
		}
		t.Logf("- %+v", url)
	}
}

func TestWithRealServerDoCheckReportID(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.CheckReportIDRequest{
		ReportID: "20210223T093606Z_ndt_JO_8376_n1_kDYToqrugDY54Soy",
	}
	api := &simpleCheckReportIDAPI{}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp)
}

func TestWithRealServerDoMeasurementMeta(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.MeasurementMetaRequest{
		ReportID: "20210223T093606Z_ndt_JO_8376_n1_kDYToqrugDY54Soy",
	}
	api := &simpleMeasurementMetaAPI{}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp)
}

func TestWithRealServerDoOpenReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.OpenReportRequest{
		DataFormatVersion: "0.2.0",
		Format:            "json",
		ProbeASN:          "AS137",
		ProbeCC:           "IT",
		SoftwareName:      "miniooni",
		SoftwareVersion:   "0.1.0-dev",
		TestName:          "example",
		TestStartTime:     "2018-11-01 15:33:20",
		TestVersion:       "0.1.0",
	}
	api := &simpleOpenReportAPI{}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp)
}

func TestWithRealServerDoPsiphonConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.PsiphonConfigRequest{}
	httpClnt := &VerboseHTTPClient{t: t}
	api := &withLoginPsiphonConfigAPI{
		API: &simplePsiphonConfigAPI{
			HTTPClient: httpClnt,
		},
		KVStore: &memkvstore{},
		RegisterAPI: &simpleRegisterAPI{
			HTTPClient: httpClnt,
		},
		LoginAPI: &simpleLoginAPI{
			HTTPClient: httpClnt,
		},
	}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp != nil)
}

func TestWithRealServerDoTorTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.TorTargetsRequest{}
	httpClnt := &VerboseHTTPClient{t: t}
	api := &withLoginTorTargetsAPI{
		API: &simpleTorTargetsAPI{
			HTTPClient: httpClnt,
		},
		KVStore: &memkvstore{},
		RegisterAPI: &simpleRegisterAPI{
			HTTPClient: httpClnt,
		},
		LoginAPI: &simpleLoginAPI{
			HTTPClient: httpClnt,
		},
	}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp != nil)
}

func TestWithRealServerDoURLs(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.URLsRequest{
		CountryCode: "IT",
		Limit:       3,
	}
	api := &simpleURLsAPI{}
	ctx := context.Background()
	resp, err := api.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp)
}
