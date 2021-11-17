package mlablocate_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mlablocate"
)

func Example_usage() {
	clnt := mlablocate.NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	result, err := clnt.Query(context.Background(), "neubot/dash")
	if err != nil {
		log.WithError(err).Fatal("clnt.Query failed")
	}
	fmt.Printf("%s\n", result.FQDN)
}
