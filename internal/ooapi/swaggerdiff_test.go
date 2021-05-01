package ooapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/ooni/probe-cli/v3/internal/ooapi/internal/openapi"
)

const (
	productionURL = "https://api.ooni.io/apispec_1.json"
	testingURL    = "https://ams-pg-test.ooni.org/apispec_1.json"
)

func makeModel(data []byte) *openapi.Swagger {
	var out openapi.Swagger
	if err := json.Unmarshal(data, &out); err != nil {
		log.Fatal(err)
	}
	// We reduce irrelevant differences by producing a common header
	return &openapi.Swagger{Paths: out.Paths}
}

func getServerModel(serverURL string) *openapi.Swagger {
	resp, err := http.Get(serverURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return makeModel(data)
}

func getClientModel() *openapi.Swagger {
	return makeModel([]byte(swagger))
}

func simplifyRoundTrip(rt *openapi.RoundTrip) {
	// Normalize the used name when a parameter is in body. This
	// should only have a cosmetic impact on the spec.
	for _, param := range rt.Parameters {
		if param.In == "body" {
			param.Name = "body"
		}
	}

	// Sort parameters so the comparison does not depend on order.
	sort.SliceStable(rt.Parameters, func(i, j int) bool {
		left, right := rt.Parameters[i].Name, rt.Parameters[j].Name
		return strings.Compare(left, right) < 0
	})

	// Normalize description of 200 response
	rt.Responses.Successful.Description = "all good"
}

func simplifyInPlace(path *openapi.Path) *openapi.Path {
	if path.Get != nil && path.Post != nil {
		log.Fatal("unsupported configuration")
	}
	if path.Get != nil {
		simplifyRoundTrip(path.Get)
	}
	if path.Post != nil {
		simplifyRoundTrip(path.Post)
	}
	return path
}

func jsonify(model interface{}) string {
	data, err := json.MarshalIndent(model, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

type diffable struct {
	name  string
	value string
}

func computediff(server, client *diffable) string {
	d := gotextdiff.ToUnified(server.name, client.name, server.value, myers.ComputeEdits(
		span.URIFromPath(server.name), server.value, client.value,
	))
	return fmt.Sprint(d)
}

// maybediff emits the diff between the server and the client and
// returns the length of the diff itself in bytes.
func maybediff(key string, server, client *openapi.Path) int {
	diff := computediff(&diffable{
		name:  fmt.Sprintf("server%s.json", key),
		value: jsonify(simplifyInPlace(server)),
	}, &diffable{
		name:  fmt.Sprintf("client%s.json", key),
		value: jsonify(simplifyInPlace(client)),
	})
	if diff != "" {
		fmt.Printf("%s", diff)
	}
	return len(diff)
}

func compare(serverURL string) bool {
	good := true
	serverModel, clientModel := getServerModel(serverURL), getClientModel()
	// Implementation note: the server model is richer than the client
	// model, so we ignore everything not defined by the client.
	var count int
	for key := range serverModel.Paths {
		if _, found := clientModel.Paths[key]; !found {
			delete(serverModel.Paths, key)
			continue
		}
		count++
		if maybediff(key, serverModel.Paths[key], clientModel.Paths[key]) > 0 {
			good = false
		}
	}
	if count <= 0 {
		panic("no element found")
	}
	return good
}

func TestWithProductionAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	t.Log("using ", productionURL)
	if !compare(productionURL) {
		t.Fatal("model mismatch (see above)")
	}
}

func TestWithTestingAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	t.Log("using ", testingURL)
	if !compare(testingURL) {
		t.Fatal("model mismatch (see above)")
	}
}
