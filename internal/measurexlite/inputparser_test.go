package measurexlite

import (
	"errors"
	"net/url"
	"testing"
)

func TestInputParser(t *testing.T) {
	t.Run("invalid configuration", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{},
			AllowEndpoints:  true,
			DefaultScheme:   "",
		}
		parsed, err := inputParser.Parse("www.example.com:80")
		if !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatal("unexpected error")
		}
		if parsed != nil {
			t.Fatal("expected nil url")
		}
	})

	t.Run("returns URL on success", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		}
		parsed, err := inputParser.Parse("https://example.com")
		if err != nil {
			t.Fatal("unexpected error")
		}
		if parsed.Scheme != "https" {
			t.Fatal("unexpected scheme in url")
		}
		if parsed.Host != "example.com" {
			t.Fatal("unexpected host in url")
		}
	})

	t.Run("failure on invalid input", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  true,
			DefaultScheme:   "tlshandshake",
		}
		parsed, err := inputParser.Parse("\t")
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatal("unexpected error")
		}
		if parsed != nil {
			t.Fatal("expected nil url")
		}
	})

	t.Run("forwards endpoints to maybeAllowEndpoints", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  true,
			DefaultScheme:   "tlshandshake",
		}
		parsed, err := inputParser.Parse("www.example.com:80")
		if err != nil {
			t.Fatal("unexpected error")
		}
		if parsed.Scheme != "tlshandshake" {
			t.Fatal("unexpected url scheme")
		}
		if parsed.Host != "www.example.com:80" {
			t.Fatal("unexpected url host")
		}
	})

	t.Run("forwards endpoints to maybeAllowEndpoints on error", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  true,
			DefaultScheme:   "tlshandshake",
		}
		parsed, err := inputParser.Parse("example.com:80")
		if err != nil {
			t.Fatal("unexpected error")
		}
		if parsed.Scheme != "tlshandshake" {
			t.Fatal("unexpected url scheme")
		}
		if parsed.Host != "example.com:80" {
			t.Fatal("unexpected url host")
		}
	})

	t.Run("falure in scheme", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		}
		parsed, err := inputParser.Parse("tlshandshake://example.com")
		if !errors.Is(err, ErrInvalidScheme) {
			t.Fatal("unexpected error")
		}
		if parsed != nil {
			t.Fatal("expected nil url")
		}
	})
}

func TestMaybeAllowEndpoints(t *testing.T) {
	t.Run("invalid configuration", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		}
		url := &url.URL{
			Host: "example.com",
		}
		parsed, err := inputParser.maybeAllowEndpoints(url)
		if !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatal("unexpected error")
		}
		if parsed != nil {
			t.Fatal("expected nil url")
		}
	})

	t.Run("returns URL on success", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  true,
			DefaultScheme:   "tlshandshake",
		}
		url := &url.URL{
			Scheme: "www.example.com",
			Opaque: "80",
		}
		parsed, err := inputParser.maybeAllowEndpoints(url)
		if err != nil {
			t.Fatal("unexpected error")
		}
		if parsed.Scheme != "tlshandshake" {
			t.Fatal("unexpected url scheme")
		}
		if parsed.Host != "www.example.com:80" {
			t.Fatal("unexpected url host")
		}
	})

	t.Run("failure on invalid input", func(t *testing.T) {
		inputParser := &InputParser{
			AcceptedSchemes: []string{"https"},
			AllowEndpoints:  true,
			DefaultScheme:   "tlshandshake",
		}
		url := &url.URL{
			Host: "example.com",
		}
		parsed, err := inputParser.maybeAllowEndpoints(url)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatal("unexpected error")
		}
		if parsed != nil {
			t.Fatal("expected nil url")
		}
	})
}
