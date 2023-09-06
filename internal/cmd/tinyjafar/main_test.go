package main

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMain(t *testing.T) {
	t.Run("test the main function with ReturnImmediately", func(t *testing.T) {
		defer returnImmediately.Store(false)
		returnImmediately.Store(true)
		main()
		if mainWithArgsCalled.Load() != 1 {
			t.Fatal("main did not call mainWithArguments")
		}
	})

	// testcase is a test case for this function
	type testcase struct {
		// name is the test case name
		name string

		// args contains the arguments passed to the command line
		args []string

		// expect contains the expected program output
		expect []string
	}

	testcases := []testcase{
		{
			name: "without any command line argument",
			args: []string{},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},

		{
			name: "with -iptables-drop-ip",
			args: []string{
				"-iptables-drop-ip", "130.192.16.171",
			},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -A JAFAR_OUTPUT -d 130.192.16.171 -j DROP",
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},

		{
			name: "with -iptables-drop-keyword-hex",
			args: []string{
				"-iptables-drop-keyword-hex", "|07 65 78 61 6d 70 6c 65 03 63 6f 6d|",
			},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				`+ iptables -A JAFAR_OUTPUT -m string --algo kmp --hex-string "|07 65 78 61 6d 70 6c 65 03 63 6f 6d|" -j DROP`,
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},

		{
			name: "with -iptables-drop-keyword",
			args: []string{
				"-iptables-drop-keyword", "ooni.org",
			},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -A JAFAR_OUTPUT -m string --algo kmp --string ooni.org -j DROP",
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},

		{
			name: "with -iptables-reset-ip",
			args: []string{
				"-iptables-reset-ip", "130.192.16.171",
			},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -A JAFAR_OUTPUT --proto tcp -d 130.192.16.171 -j REJECT --reject-with tcp-reset",
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},

		{
			name: "with -iptables-reset-keyword-hex",
			args: []string{
				"-iptables-reset-keyword-hex", "|6F 6F 6E 69|",
			},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				`+ iptables -A JAFAR_OUTPUT -m string --proto tcp --algo kmp --hex-string "|6F 6F 6E 69|" -j REJECT --reject-with tcp-reset`,
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},

		{
			name: "with -iptables-reset-keyword",
			args: []string{
				"-iptables-reset-keyword", "ooni.org",
			},
			expect: []string{
				"+ iptables -N JAFAR_INPUT",
				"+ iptables -N JAFAR_OUTPUT",
				"+ iptables -t nat -N JAFAR_NAT_OUTPUT",
				"+ iptables -I OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -I INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -A JAFAR_OUTPUT -m string --proto tcp --algo kmp --string ooni.org -j REJECT --reject-with tcp-reset",
				"",
				"Use Ctrl-C to terminate",
				"",
				"+ iptables -D OUTPUT -j JAFAR_OUTPUT",
				"+ iptables -D INPUT -j JAFAR_INPUT",
				"+ iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT",
				"+ iptables -F JAFAR_INPUT",
				"+ iptables -X JAFAR_INPUT",
				"+ iptables -F JAFAR_OUTPUT",
				"+ iptables -X JAFAR_OUTPUT",
				"+ iptables -t nat -F JAFAR_NAT_OUTPUT",
				"+ iptables -t nat -X JAFAR_NAT_OUTPUT",
				"",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var builder strings.Builder
			input := append([]string{}, tc.args...)
			input = append(input, "-dry-run")

			sigChan := make(chan os.Signal)
			close(sigChan) // so mainWithArgs would not block

			t.Logf("executing with %+v", input)
			mainWithArgs(&builder, sigChan, input...)

			output := builder.String()
			lines := strings.Split(output, "\n")
			if diff := cmp.Diff(tc.expect, lines); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
