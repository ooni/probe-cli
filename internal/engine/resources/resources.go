// Package resources contains code to download resources.
package resources

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// Client is a client for fetching resources.
type Client struct {
	// HTTPClient is the HTTP client to use.
	HTTPClient *http.Client

	// Logger is the logger to use.
	Logger model.Logger

	// OSMkdirAll allows testing os.MkdirAll failures.
	OSMkdirAll func(path string, perm os.FileMode) error

	// UserAgent is the user agent to use.
	UserAgent string

	// WorkDir is the directory where to save resources.
	WorkDir string
}

// Ensure ensures that resources are downloaded and current.
func (c *Client) Ensure(ctx context.Context) error {
	mkdirall := c.OSMkdirAll
	if mkdirall == nil {
		mkdirall = os.MkdirAll
	}
	if err := mkdirall(c.WorkDir, 0700); err != nil {
		return err
	}
	for name, resource := range All {
		if err := c.EnsureForSingleResource(
			ctx, name, resource, func(real, expected string) bool {
				return real == expected
			},
			gzip.NewReader, ioutil.ReadAll,
		); err != nil {
			return err
		}
	}
	return nil
}

// EnsureForSingleResource ensures that a single resource
// is downloaded and is current.
func (c *Client) EnsureForSingleResource(
	ctx context.Context, name string, resource ResourceInfo,
	equal func(real, expected string) bool,
	gzipNewReader func(r io.Reader) (*gzip.Reader, error),
	ioutilReadAll func(r io.Reader) ([]byte, error),
) error {
	fullpath := filepath.Join(c.WorkDir, name)
	data, err := ioutil.ReadFile(fullpath)
	if err == nil {
		sha256sum := fmt.Sprintf("%x", sha256.Sum256(data))
		if equal(sha256sum, resource.SHA256) {
			return nil
		}
		c.Logger.Debugf("resources: %s is outdated", fullpath)
	} else {
		c.Logger.Debugf("resources: can't read %s: %s", fullpath, err.Error())
	}
	data, err = (httpx.Client{
		BaseURL:    BaseURL,
		HTTPClient: c.HTTPClient,
		Logger:     c.Logger,
		UserAgent:  c.UserAgent,
	}).FetchResourceAndVerify(ctx, resource.URLPath, resource.GzSHA256)
	if err != nil {
		return err
	}
	c.Logger.Debugf("resources: uncompress %s", fullpath)
	gzreader, err := gzipNewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gzreader.Close()              // we already have a sha256 for it
	data, err = ioutilReadAll(gzreader) // small file
	if err != nil {
		return err
	}
	sha256sum := fmt.Sprintf("%x", sha256.Sum256(data))
	if equal(sha256sum, resource.SHA256) == false {
		return fmt.Errorf("resources: %s sha256 mismatch", fullpath)
	}
	c.Logger.Debugf("resources: overwrite %s", fullpath)
	return ioutil.WriteFile(fullpath, data, 0600)
}
