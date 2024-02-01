package buildtooltest

import (
	"path/filepath"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/fsx"
	"golang.org/x/sys/execabs"
)

func TestCheckManyCommands(t *testing.T) {

	type testcase struct {
		name      string
		cmd       []*execabs.Cmd
		tee       []ExecExpectations
		expectErr bool
	}

	var testcases = []testcase{{
		name: "where everything is working as intended",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}, {
			Path: "/usr/local/bin/go",
			Args: []string{"go", "env", "GOPATH"},
			Env: []string{
				"GOPATH=/tmp",
			},
		}},
		tee: []ExecExpectations{{
			Env:  []string{},
			Argv: []string{"go", "version"},
		}, {
			Env:  []string{"GOPATH=/tmp"},
			Argv: []string{"go", "env", "GOPATH"},
		}},
		expectErr: false,
	}, {
		name: "where we didn't find the environment we expected",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}},
		tee: []ExecExpectations{{
			Env:  []string{"GOPATH=/tmp"},
			Argv: []string{"go", "version"},
		}},
		expectErr: true,
	}, {
		name: "where a specific command line argument differs",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}},
		tee: []ExecExpectations{{
			Argv: []string{"go", "env"},
		}},
		expectErr: true,
	}, {
		name: "where the argvs have different length",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}},
		tee: []ExecExpectations{{
			Argv: []string{"go", "env", "GOPATH"},
		}},
		expectErr: true,
	}, {
		name: "where the argv[0] suffix does not match",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go1.17.9",
			Args: []string{"go1.17.9", "version"},
		}},
		tee: []ExecExpectations{{
			Argv: []string{"go", "version"},
		}},
		expectErr: true,
	}, {
		name: "where we got more environment than expected",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
			Env:  []string{"GOPATH=/tmp"},
		}},
		tee: []ExecExpectations{{
			Argv: []string{"go", "version"},
		}},
		expectErr: true,
	}, {
		name: "with mismatch between number of commands and expectations",
		cmd:  []*execabs.Cmd{},
		tee: []ExecExpectations{{
			Argv: []string{"go", "version"},
		}},
		expectErr: true,
	}}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckManyCommands(c.cmd, c.tee)
			if err != nil && !c.expectErr {
				t.Fatal("did not expect an error", err)
			}
			if err == nil && c.expectErr {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestSimpleCommandCollector(t *testing.T) {
	t.Run("LookPath", func(t *testing.T) {
		cc := &SimpleCommandCollector{}
		path, err := cc.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
		if path != "go" {
			t.Fatal("invalid path")
		}
	})

	t.Run("CmdOutput", func(t *testing.T) {
		cc := &SimpleCommandCollector{}
		cmd := &execabs.Cmd{}
		output, err := cc.CmdOutput(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if len(output) != 0 {
			t.Fatal("invalid output")
		}
		if cc.Commands[0] != cmd {
			t.Fatal("did not save the command")
		}
	})

	t.Run("CmdRun", func(t *testing.T) {
		cc := &SimpleCommandCollector{}
		cmd := &execabs.Cmd{}
		if err := cc.CmdRun(cmd); err != nil {
			t.Fatal(err)
		}
		if cc.Commands[0] != cmd {
			t.Fatal("did not save the command")
		}
	})
}

func TestDependenciesCallCounter(t *testing.T) {
	t.Run("AbsoluteCurDir", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		dir := cc.AbsoluteCurDir()
		if cc.Counter[TagAbsoluteCurDir] != 1 {
			t.Fatal("did not increment")
		}
		// Note: adding two ".." to offset for the fact that we're two
		// more directories deep with respect to when we run tests
		cdepsDir := filepath.Join(dir, "..", "..", "CDEPS")
		if !fsx.DirectoryExists(cdepsDir) {
			t.Fatal("directory does not exist", cdepsDir)
		}
	})

	t.Run("AndroidNDKCheck", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		dir := cc.AndroidNDKCheck("xo")
		if cc.Counter[TagAndroidNDKCheck] != 1 {
			t.Fatal("did not increment")
		}
		expect := filepath.Join("xo", "ndk", CanonicalNDKVersion)
		if dir != expect {
			t.Fatal("expected", expect, "but got", dir)
		}
	})

	t.Run("AndroidSDKCheck", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		dir := cc.AndroidSDKCheck()
		if cc.Counter[TagAndroidSDKCheck] != 1 {
			t.Fatal("did not increment")
		}
		expect := filepath.Join("Android", "sdk")
		if dir != expect {
			t.Fatal("expected", expect, "but got", dir)
		}
	})

	t.Run("GOPATH", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		dir := cc.GOPATH()
		if cc.Counter[TagGOPATH] != 1 {
			t.Fatal("did not increment")
		}
		expect := "/go/gopath"
		if dir != expect {
			t.Fatal("expected", expect, "but got", dir)
		}
	})

	t.Run("GolangCheck", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.GolangCheck()
		if cc.Counter[TagGolangCheck] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("LinuxReadGOVERSION", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.LinuxReadGOVERSION("xo")
		if cc.Counter[TagLinuxReadGOVERSION] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("LinuxWriteDOCKEFILE", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.LinuxWriteDockerfile("xo", nil, 0600)
		if cc.Counter[TagLinuxWriteDockerfile] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("MustChdir", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.MustChdir("xo")
		if cc.Counter[TagMustChdir] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("PsiphonFileExists", func(t *testing.T) {
		t.Run("if false", func(t *testing.T) {
			cc := &DependenciesCallCounter{}
			got := cc.PsiphonFilesExist()
			if got {
				t.Fatal("expected false here")
			}
			if cc.Counter[TagPsiphonFilesExist] != 1 {
				t.Fatal("did not increment")
			}
		})

		t.Run("if false", func(t *testing.T) {
			cc := &DependenciesCallCounter{
				HasPsiphon: true,
			}
			got := cc.PsiphonFilesExist()
			if !got {
				t.Fatal("expected true here")
			}
			if cc.Counter[TagPsiphonFilesExist] != 1 {
				t.Fatal("did not increment")
			}
		})
	})

	t.Run("PsiphonMaybeCopyConfigFiles", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.PsiphonMaybeCopyConfigFiles()
		if cc.Counter[TagPsiphonMaybeCopyConfigFiles] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("VerifySHA256", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.VerifySHA256("xo", "xo")
		if cc.Counter[TagVerifySHA256] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("WindowsMingwCheck", func(t *testing.T) {
		cc := &DependenciesCallCounter{}
		cc.WindowsMingwCheck()
		if cc.Counter[TagWindowsMingwCheck] != 1 {
			t.Fatal("did not increment")
		}
	})

	t.Run("GOOS", func(t *testing.T) {
		cc := &DependenciesCallCounter{
			OS: "linux",
		}
		got := cc.GOOS()
		if got != "linux" {
			t.Fatal("unexpected GOOS value")
		}
		if cc.Counter[TagGOOS] != 1 {
			t.Fatal("did not increment")
		}
	})
}
