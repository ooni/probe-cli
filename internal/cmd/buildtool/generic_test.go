package main

/*
func TestGenericBuildPackage(t *testing.T) {

	type testspec struct {
		name       string
		product    *product
		psiphon    bool
		expectEnv  map[string]int
		expectArgv []string
	}

	var testcases = []testspec{{
		name:      "miniooni build with psiphon",
		product:   productMiniooni,
		psiphon:   true,
		expectEnv: map[string]int{},
		expectArgv: []string{
			runtimex.Try1(exec.LookPath("go")),
			"build", "-tags", "ooni_psiphon_config",
			"-ldflags", "-s -w",
			"./internal/cmd/miniooni",
		},
	}, {
		name:      "ooniprobe build without psiphon",
		product:   productOoniprobe,
		psiphon:   false,
		expectEnv: map[string]int{},
		expectArgv: []string{
			runtimex.Try1(exec.LookPath("go")),
			"build", "-ldflags", "-s -w", "./cmd/ooniprobe",
		},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			commands := []*exec.Cmd{}
			library := &shellxtesting.Library{
				MockCmdRun: func(c *exec.Cmd) error {
					commands = append(commands, c)
					return nil
				},
				MockLookPath: func(file string) (string, error) {
					return file, nil
				},
			}

			shellxtesting.WithCustomLibrary(library, func() {
				genericBuildPackage(testcase.product, testcase.psiphon)
			})

			if len(commands) != 1 {
				t.Fatal("expected a single command")
			}
			command := commands[0]
			envs := shellxtesting.RemoveCommonEnvironmentVariables(command)
			gotEnv := map[string]int{}
			for _, env := range envs {
				gotEnv[env]++
			}
			if diff := cmp.Diff(testcase.expectEnv, gotEnv); diff != "" {
				t.Fatal(diff)
			}
			gotArgv := shellxtesting.MustArgv(command)
			if diff := cmp.Diff(testcase.expectArgv, gotArgv); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
*/
