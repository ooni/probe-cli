package mlablocatev2_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mlablocatev2"
)

func Example_usage() {
	clnt := mlablocatev2.NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	results, err := clnt.QueryNDT7(context.Background())
	if err != nil {
		log.WithError(err).Fatal("clnt.QueryNDT7 failed")
	}
	fmt.Printf("%+v\n", results)
}
