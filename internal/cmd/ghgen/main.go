// Command ghgen regenerates selected GitHub actions.
package main

import (
	_ "embed"
)

func main() {
	for name, jobs := range Config {
		generateWorkflowFile(name, jobs)
	}
}
