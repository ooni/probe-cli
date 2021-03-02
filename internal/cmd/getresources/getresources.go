// Command getresources downloads the resources
package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/internal/engine/resources"
)

func main() {
	for name, ri := range resources.All {
		if err := getit(name, &ri); err != nil {
			log.Fatal(err)
		}
	}
}

func getit(name string, ri *resources.ResourceInfo) error {
	workDir := filepath.Join("internal", "engine", "resourcesmanager")
	URL, err := url.Parse(resources.BaseURL)
	if err != nil {
		return err
	}
	URL.Path = ri.URLPath
	log.Println("fetching", URL.String())
	resp, err := http.Get(URL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("http request failed")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	if checksum != ri.GzSHA256 {
		return errors.New("sha256 mismatch")
	}
	fullpath := filepath.Join(workDir, name+".gz")
	return ioutil.WriteFile(fullpath, data, 0644)
}
