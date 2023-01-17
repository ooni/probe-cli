package probeservices

import (
	"context"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestCheckInSuccess(t *testing.T) {
	client := newclient()
	client.BaseURL = "https://ams-pg-test.ooni.org"
	config := model.OOAPICheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		ProbeASN:        "AS12353",
		ProbeCC:         "PT",
		RunType:         model.RunTypeTimed,
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{"NEWS", "CULTR"},
		},
	}
	ctx := context.Background()
	result, err := client.CheckIn(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Tests.WebConnectivity == nil {
		t.Fatal("got nil result or WebConnectivity")
	}
	if result.Tests.WebConnectivity.ReportID == "" {
		t.Fatal("ReportID is empty")
	}
	if len(result.Tests.WebConnectivity.URLs) < 1 {
		t.Fatal("unexpected number of URLs")
	}
	for _, entry := range result.Tests.WebConnectivity.URLs {
		if entry.CategoryCode != "NEWS" && entry.CategoryCode != "CULTR" {
			t.Fatalf("unexpected category code: %+v", entry)
		}
	}
}

func TestCheckInFailure(t *testing.T) {
	client := newclient()
	client.BaseURL = "https://\t\t\t/" // cause test to fail
	config := model.OOAPICheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		ProbeASN:        "AS12353",
		ProbeCC:         "PT",
		RunType:         model.RunTypeTimed,
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{"NEWS", "CULTR"},
		},
	}
	ctx := context.Background()
	result, err := client.CheckIn(ctx, config)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("results?!")
	}
}
