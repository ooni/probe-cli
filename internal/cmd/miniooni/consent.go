package main

//
// Acquiring user's consent
//

import (
	"os"
	"path"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// acquireUserConsent ensures the user is okay with using miniooni. This function
// panics if we do not have acquired the user consent.
func acquireUserConsent(miniooniDir string, currentOptions *Options) {
	consentFile := path.Join(miniooniDir, "informed")
	err := maybeWriteConsentFile(currentOptions.Yes, consentFile)
	runtimex.PanicOnError(err, "cannot write informed consent file")
	runtimex.Assert(
		regularFileExists(consentFile),
		riskOfRunningOONI,
	)
}

// maybeWriteConsentFile writes the consent file iff the yes argument is true
func maybeWriteConsentFile(yes bool, filepath string) (err error) {
	if yes {
		err = os.WriteFile(filepath, []byte("\n"), 0644)
	}
	return
}

// riskOfRunningOONI is miniooni's informed consent text.
const riskOfRunningOONI = `

Do you consent to OONI Probe data collection?

OONI Probe collects evidence of internet censorship and measures
network performance:

- OONI Probe will likely test objectionable sites and services;

- Anyone monitoring your internet activity (such as a government
or Internet provider) may be able to tell that you are using OONI Probe;

- The network data you collect will be published automatically
unless you use miniooni's -n command line flag.

To learn more, see https://ooni.org/about/risks/.

If you're onboard, re-run the same command and add the --yes flag, to
indicate that you understand the risks. This will create an empty file
named 'consent' in $HOME/.miniooni, meaning that we know you opted in
and we will not ask you this question again.

`
