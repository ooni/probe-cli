package ooapi_test

import (
	"context"
	"fmt"
	"log"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/ooapi"
	"github.com/ooni/probe-cli/v3/internal/ooapi/apimodel"
)

func ExampleClient() {
	clnt := &ooapi.Client{
		KVStore: &kvstore.Memory{},
	}
	ctx := context.Background()
	resp, err := clnt.CheckIn(ctx, &apimodel.CheckInRequest{
		Charging:        false,
		OnWiFi:          false,
		Platform:        "linux",
		ProbeASN:        "AS30722",
		ProbeCC:         "IT",
		RunType:         "timed",
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		WebConnectivity: apimodel.CheckInRequestWebConnectivity{
			CategoryCodes: []string{"NEWS"},
		},
	})
	fmt.Printf("%+v\n", err)
	// Output: <nil>
	if resp == nil {
		log.Fatal("expected non-nil response")
	}
}
