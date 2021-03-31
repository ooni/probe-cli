package nettests

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestPreventMistakesWithCategories(t *testing.T) {
	input := []model.URLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "HACK",
		URL:          "https://2600.com",
		CountryCode:  "XX",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	desired := []model.URLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	output := preventMistakes(input, []string{"NEWS", "FILE"})
	if diff := cmp.Diff(desired, output); diff != "" {
		t.Fatal(diff)
	}
}

func TestPreventMistakesWithoutCategories(t *testing.T) {
	input := []model.URLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "HACK",
		URL:          "https://2600.com",
		CountryCode:  "XX",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	output := preventMistakes(input, nil)
	if diff := cmp.Diff(input, output); diff != "" {
		t.Fatal(diff)
	}
}
