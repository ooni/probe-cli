package measurexlite

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

func TestWebGetTitle(t *testing.T) {
	type args struct {
		body string
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
	}{{
		name: "with empty input",
		args: args{
			body: "",
		},
		wantOut: "",
	}, {
		name: "with body containing no titles",
		args: args{
			body: "<HTML/>",
		},
		wantOut: "",
	}, {
		name: "success with UTF-7 body",
		args: args{
			body: "<HTML><TITLE>La community di MSN</TITLE></HTML>",
		},
		wantOut: "La community di MSN",
	}, {
		name: "success with UTF-8 body",
		args: args{
			body: "<HTML><TITLE>La comunità di MSN</TITLE></HTML>",
		},
		wantOut: "La comunità di MSN",
	}, {
		name: "when the title is too long",
		args: args{
			body: "<HTML><TITLE>" + randx.Letters(1024) + "</TITLE></HTML>",
		},
		wantOut: "",
	}, {
		name: "success with case variations",
		args: args{
			body: "<HTML><TiTLe>La commUNity di MSN</tITLE></HTML>",
		},
		wantOut: "La commUNity di MSN",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := WebGetTitle(tt.args.body)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
