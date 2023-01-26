package main

//
// Utility functions.
//

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func newJob(w io.Writer, name, runsOn, needs string, permissions map[string]string) {
	mustFprintf(w, "  %s:\n", name)
	mustFprintf(w, "    runs-on: %s\n", runsOn)
	if needs != "" {
		mustFprintf(w, "    needs: %s\n", needs)
	}
	if len(permissions) > 0 {
		mustFprintf(w, "    permissions:\n")
		for key, value := range permissions {
			mustFprintf(w, "      %s: %s\n", key, value)
		}
	}
	mustFprintf(w, "    steps:\n")
}

func newStepCheckout(w io.Writer) {
	mustFprintf(w, "      - uses: actions/checkout@v2\n")
	mustFprintf(w, "        with:\n")
	mustFprintf(w, "          fetch-depth: 0\n")
	mustFprintf(w, "\n")
}

func newStepSetupGo(w io.Writer, cacheName string) {
	mustFprintf(w, "      - name: Get GOVERSION content\n")
	mustFprintf(w, "        id: goversion\n")
	mustFprintf(w, "        run: echo ::set-output name=version::$(cat GOVERSION)\n")
	mustFprintf(w, "      - uses: magnetikonline/action-golang-cache@v2\n")
	mustFprintf(w, "        with:\n")
	mustFprintf(w, "          go-version: \"${{ steps.goversion.outputs.version }}\"\n")
	mustFprintf(w, "          cache-key-suffix: \"-%s-${{ steps.goversion.outputs.version }}\"\n", cacheName)
	mustFprintf(w, "\n")
}

func newStepSetupPsiphon(w io.Writer) {
	mustFprintf(w, "      - run: |\n")
	mustFprintf(w, "          echo -n $PSIPHON_CONFIG_KEY > ./internal/engine/psiphon-config.key\n")
	mustFprintf(w, "          echo $PSIPHON_CONFIG_JSON_AGE_BASE64 | base64 -d > ./internal/engine/psiphon-config.json.age\n")
	mustFprintf(w, "        env:\n")
	mustFprintf(w, "          PSIPHON_CONFIG_KEY: ${{ secrets.PSIPHON_CONFIG_KEY }}\n")
	mustFprintf(w, "          PSIPHON_CONFIG_JSON_AGE_BASE64: ${{ secrets.PSIPHON_CONFIG_JSON_AGE_BASE64 }}\n")
	mustFprintf(w, "\n")
}

func newStepMake(w io.Writer, target string) {
	mustFprintf(w, "      - run: make %s\n", target)
	mustFprintf(w, "\n")
}

func newStepUploadArtifacts(w io.Writer, artifacts []string) {
	for _, arti := range artifacts {
		mustFprintf(w, "      - uses: actions/upload-artifact@v2\n")
		mustFprintf(w, "        with:\n")
		mustFprintf(w, "          name: %s\n", filepath.Base(arti))
		mustFprintf(w, "          path: %s\n", arti)
		mustFprintf(w, "\n")
	}
}

func newStepDownloadArtifacts(w io.Writer, artifacts []string) {
	for _, arti := range artifacts {
		mustFprintf(w, "      - uses: actions/download-artifact@v2\n")
		mustFprintf(w, "        with:\n")
		mustFprintf(w, "          name: %s\n", filepath.Base(arti))
		mustFprintf(w, "\n")
	}
}

func newStepGHPublish(w io.Writer, artifacts []string) {
	runtimex.Assert(len(artifacts) > 0, "expected at least one artifact")
	artifactsNames := []string{}
	for _, arti := range artifacts {
		artifactsNames = append(artifactsNames, filepath.Base(arti))
	}
	mustFprintf(w, "      - run: ./script/ghpublish.bash %s\n", strings.Join(artifactsNames, " "))
	mustFprintf(w, "        env:\n")
	mustFprintf(w, "          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
	mustFprintf(w, "\n")
}

func newStepSetupLinuxDockerGoCache(w io.Writer, name string) {
	mustFprintf(w, "      - uses: actions/cache@v3\n")
	mustFprintf(w, "        with:\n")
	mustFprintf(w, "          path: GOCACHE\n")
	mustFprintf(w, "          key: linux-build-cache-%s\n", name)
	mustFprintf(w, "\n")

}

func newSetupInstallQemuUserStatic(w io.Writer) {
	mustFprintf(w, "      - run: sudo apt-get update -q\n")
	mustFprintf(w, "      - run: sudo apt-get install -y qemu-user-static\n")
	mustFprintf(w, "\n")
}

func newStepInstallTor(w io.Writer) {
	mustFprintf(w, "      - run: sudo apt-get update -q\n")
	mustFprintf(w, "      - run: sudo apt-get install -y tor\n")
	mustFprintf(w, "\n")
}

func newStepRunOONIProbeIntegrationTests(w io.Writer, os, arch, ext string) {
	executable := fmt.Sprintf("ooniprobe-%s-%s%s", os, arch, ext)
	if os != "windows" {
		mustFprintf(w, "      - run: chmod +x %s\n", executable)
	}
	mustFprintf(w, "      - run: ./E2E/ooniprobe.bash ./%s\n", executable)
	mustFprintf(w, "        shell: bash\n")
	mustFprintf(w, "\n")
}

func newStepRunMiniooniIntegrationTests(w io.Writer, os, arch, ext string) {
	executable := fmt.Sprintf("miniooni-%s-%s%s", os, arch, ext)
	if os != "windows" {
		mustFprintf(w, "      - run: chmod +x %s\n", executable)
	}
	mustFprintf(w, "      - run: ./E2E/miniooni.bash ./%s\n", executable)
	mustFprintf(w, "        shell: bash\n")
	mustFprintf(w, "\n")
}

func newStepInstallMingwW64(w io.Writer) {
	mustFprintf(w, "      - run: sudo apt-get update -q\n")
	mustFprintf(w, "      - run: sudo apt-get install -y mingw-w64\n")
	mustFprintf(w, "\n")
}

func mustFprintf(w io.Writer, format string, v ...any) {
	_, err := fmt.Fprintf(w, format, v...)
	runtimex.PanicOnError(err, "fmt.Fprintf failed")
}

func mustClose(c io.Closer) {
	err := c.Close()
	runtimex.PanicOnError(err, "c.Close failed")
}

func generateWorkflowFile(name string, jobs []Job) {
	filename := filepath.Join(".github", "workflows", name+".yml")
	fp, err := os.Create(filename)
	runtimex.PanicOnError(err, "os.Create failed")
	defer mustClose(fp)
	mustFprintf(fp, "# File generated by `go run ./internal/cmd/ghgen`; DO NOT EDIT.\n")
	mustFprintf(fp, "\n")
	mustFprintf(fp, "name: %s\n", name)
	mustFprintf(fp, "on:\n")
	mustFprintf(fp, "  push:\n")
	mustFprintf(fp, "    branches:\n")
	mustFprintf(fp, "      - \"release/**\"\n")
	mustFprintf(fp, "      - \"fullbuild\"\n")
	mustFprintf(fp, "      - \"%sbuild\"\n", name)
	mustFprintf(fp, "    tags:\n")
	mustFprintf(fp, "      - \"v*\"\n")
	mustFprintf(fp, "  schedule:\n")
	mustFprintf(fp, "    - cron: \"17 1 * * *\"\n")
	mustFprintf(fp, "\n")
	mustFprintf(fp, "jobs:\n")
	for _, job := range jobs {
		job.Action(fp, &job)
	}
	mustFprintf(fp, "# End of autogenerated file\n")
}
