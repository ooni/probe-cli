package main

//
// Utility functions.
//

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func newJob(w io.Writer, name, runsOn, needs string, permissions map[string]string) {
	must.Fprintf(w, "  %s:\n", name)
	must.Fprintf(w, "    runs-on: %s\n", runsOn)
	if needs != "" {
		must.Fprintf(w, "    needs: %s\n", needs)
	}
	if len(permissions) > 0 {
		must.Fprintf(w, "    permissions:\n")
		for key, value := range permissions {
			must.Fprintf(w, "      %s: %s\n", key, value)
		}
	}
	must.Fprintf(w, "    steps:\n")
}

func newStepCheckout(w io.Writer) {
	must.Fprintf(w, "      - uses: actions/checkout@v2\n")
	must.Fprintf(w, "        with:\n")
	must.Fprintf(w, "          fetch-depth: 0\n")
	must.Fprintf(w, "\n")
}

func newStepSetupGo(w io.Writer, cacheName string) {
	must.Fprintf(w, "      - name: Get GOVERSION content\n")
	must.Fprintf(w, "        id: goversion\n")
	must.Fprintf(w, "        run: echo ::set-output name=version::$(cat GOVERSION)\n")
	must.Fprintf(w, "      - uses: magnetikonline/action-golang-cache@v2\n")
	must.Fprintf(w, "        with:\n")
	must.Fprintf(w, "          go-version: \"${{ steps.goversion.outputs.version }}\"\n")
	must.Fprintf(w, "          cache-key-suffix: \"-%s-${{ steps.goversion.outputs.version }}\"\n", cacheName)
	must.Fprintf(w, "\n")
}

func newStepSetupPsiphon(w io.Writer) {
	must.Fprintf(w, "      - run: |\n")
	must.Fprintf(w, "          echo -n $PSIPHON_CONFIG_KEY > ./internal/engine/psiphon-config.key\n")
	must.Fprintf(w, "          echo $PSIPHON_CONFIG_JSON_AGE_BASE64 | base64 -d > ./internal/engine/psiphon-config.json.age\n")
	must.Fprintf(w, "        env:\n")
	must.Fprintf(w, "          PSIPHON_CONFIG_KEY: ${{ secrets.PSIPHON_CONFIG_KEY }}\n")
	must.Fprintf(w, "          PSIPHON_CONFIG_JSON_AGE_BASE64: ${{ secrets.PSIPHON_CONFIG_JSON_AGE_BASE64 }}\n")
	must.Fprintf(w, "\n")
}

func newStepMake(w io.Writer, target string) {
	must.Fprintf(w, "      - run: make %s\n", target)
	must.Fprintf(w, "\n")
}

func newStepUploadArtifacts(w io.Writer, artifacts []string) {
	for _, arti := range artifacts {
		must.Fprintf(w, "      - uses: actions/upload-artifact@v2\n")
		must.Fprintf(w, "        with:\n")
		must.Fprintf(w, "          name: %s\n", filepath.Base(arti))
		must.Fprintf(w, "          path: %s\n", arti)
		must.Fprintf(w, "\n")
	}
}

func newStepDownloadArtifacts(w io.Writer, artifacts []string) {
	for _, arti := range artifacts {
		must.Fprintf(w, "      - uses: actions/download-artifact@v2\n")
		must.Fprintf(w, "        with:\n")
		must.Fprintf(w, "          name: %s\n", filepath.Base(arti))
		must.Fprintf(w, "\n")
	}
}

func newStepGHPublish(w io.Writer, artifacts []string) {
	runtimex.Assert(len(artifacts) > 0, "expected at least one artifact")
	artifactsNames := []string{}
	for _, arti := range artifacts {
		artifactsNames = append(artifactsNames, filepath.Base(arti))
	}
	must.Fprintf(w, "      - run: ./script/ghpublish.bash %s\n", strings.Join(artifactsNames, " "))
	must.Fprintf(w, "        env:\n")
	must.Fprintf(w, "          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
	must.Fprintf(w, "\n")
}

func newStepSetupLinuxDockerGoCache(w io.Writer, name string) {
	must.Fprintf(w, "      - uses: actions/cache@v3\n")
	must.Fprintf(w, "        with:\n")
	must.Fprintf(w, "          path: GOCACHE\n")
	must.Fprintf(w, "          key: linux-build-cache-%s\n", name)
	must.Fprintf(w, "\n")

}

func newSetupInstallQemuUserStatic(w io.Writer) {
	must.Fprintf(w, "      - run: sudo apt-get update -q\n")
	must.Fprintf(w, "      - run: sudo apt-get install -y qemu-user-static\n")
	must.Fprintf(w, "\n")
}

func newStepInstallTor(w io.Writer) {
	must.Fprintf(w, "      - run: sudo apt-get update -q\n")
	must.Fprintf(w, "      - run: sudo apt-get install -y tor\n")
	must.Fprintf(w, "\n")
}

func newStepRunOONIProbeIntegrationTests(w io.Writer, os, arch, ext string) {
	executable := fmt.Sprintf("ooniprobe-%s-%s%s", os, arch, ext)
	if os != "windows" {
		must.Fprintf(w, "      - run: chmod +x %s\n", executable)
	}
	must.Fprintf(w, "      - run: ./E2E/ooniprobe.bash ./%s\n", executable)
	must.Fprintf(w, "        shell: bash\n")
	must.Fprintf(w, "\n")
}

func newStepRunMiniooniIntegrationTests(w io.Writer, os, arch, ext string) {
	executable := fmt.Sprintf("miniooni-%s-%s%s", os, arch, ext)
	if os != "windows" {
		must.Fprintf(w, "      - run: chmod +x %s\n", executable)
	}
	must.Fprintf(w, "      - run: ./E2E/miniooni.bash ./%s\n", executable)
	must.Fprintf(w, "        shell: bash\n")
	must.Fprintf(w, "\n")
}

func newStepInstallMingwW64(w io.Writer) {
	must.Fprintf(w, "      - run: sudo apt-get update -q\n")
	must.Fprintf(w, "      - run: sudo apt-get install -y mingw-w64\n")
	must.Fprintf(w, "\n")
}

func generateWorkflowFile(name string, jobs []Job) {
	filename := filepath.Join(".github", "workflows", name+".yml")
	fp := must.CreateFile(filename)
	defer fp.MustClose()
	must.Fprintf(fp, "# File generated by `go run ./GHGEN`; DO NOT EDIT.\n")
	must.Fprintf(fp, "\n")
	must.Fprintf(fp, "name: %s\n", name)
	must.Fprintf(fp, "on:\n")
	must.Fprintf(fp, "  push:\n")
	must.Fprintf(fp, "    branches:\n")
	must.Fprintf(fp, "      - \"release/**\"\n")
	must.Fprintf(fp, "      - \"fullbuild\"\n")
	must.Fprintf(fp, "      - \"%sbuild\"\n", name)
	must.Fprintf(fp, "    tags:\n")
	must.Fprintf(fp, "      - \"v*\"\n")
	must.Fprintf(fp, "  schedule:\n")
	must.Fprintf(fp, "    - cron: \"17 1 * * *\"\n")
	must.Fprintf(fp, "\n")
	must.Fprintf(fp, "jobs:\n")
	for _, job := range jobs {
		job.Action(fp, &job)
	}
	must.Fprintf(fp, "# End of autogenerated file\n")
}
