package ooapi_test

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ooapi"
	"github.com/ooni/probe-cli/v3/internal/ooapi/apimodel"
)

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
		RunType:         model.RunTypeTimed,
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: apimodel.CheckInRequestWebConnectivity{
			CategoryCodes: []string{"NEWS", "CULTR"},
		},
	}
	httpClnt := &ooapi.VerboseHTTPClient{T: t}
	clnt := &ooapi.Client{HTTPClient: httpClnt, KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.CheckIn(ctx, req)
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
	t.Skip("see https://github.com/ooni/probe/issues/2098")
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	req := &apimodel.CheckReportIDRequest{
		ReportID: "20210223T093606Z_ndt_JO_8376_n1_kDYToqrugDY54Soy",
	}
	clnt := &ooapi.Client{KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.CheckReportID(ctx, req)
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
	clnt := &ooapi.Client{KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.MeasurementMeta(ctx, req)
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
	clnt := &ooapi.Client{KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.OpenReport(ctx, req)
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
	httpClnt := &ooapi.VerboseHTTPClient{T: t}
	clnt := &ooapi.Client{HTTPClient: httpClnt, KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.PsiphonConfig(ctx, req)
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
	httpClnt := &ooapi.VerboseHTTPClient{T: t}
	clnt := &ooapi.Client{HTTPClient: httpClnt, KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.TorTargets(ctx, req)
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
	clnt := &ooapi.Client{KVStore: &kvstore.Memory{}}
	ctx := context.Background()
	resp, err := clnt.URLs(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil pointer here")
	}
	t.Logf("%+v", resp)
}
