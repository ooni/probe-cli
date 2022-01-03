package probeservices_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestFetchURLListSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	client := newclient()
	client.BaseURL = "https://ams-pg-test.ooni.org"
	config := model.OOAPIURLListConfig{
		Categories:  []string{"NEWS", "CULTR"},
		CountryCode: "IT",
		Limit:       17,
	}
	ctx := context.Background()
	result, err := client.FetchURLList(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 17 {
		t.Fatal("unexpected number of results")
	}
	for _, entry := range result {
		if entry.CategoryCode != "NEWS" && entry.CategoryCode != "CULTR" {
			t.Fatalf("unexpected category code: %+v", entry)
		}
	}
}

func TestFetchURLListFailure(t *testing.T) {
	client := newclient()
	client.BaseURL = "https://\t\t\t/" // cause test to fail
	config := model.OOAPIURLListConfig{
		Categories:  []string{"NEWS", "CULTR"},
		CountryCode: "IT",
		Limit:       17,
	}
	ctx := context.Background()
	result, err := client.FetchURLList(ctx, config)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if len(result) != 0 {
		t.Fatal("results?!")
	}
}
