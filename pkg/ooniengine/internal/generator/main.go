// Command generator generates abi.go, abi.dart, etc.
package main

import (
	"fmt"
	"path/filepath"
	"time"
)

// computeABIVersion computes the current ABI version.
func computeABIVersion() string {
	const timeFormat = "200601021504"
	return time.Now().Format(timeFormat)
}

func main() {
	abiVersion := computeABIVersion()
	dartDir := filepath.Join(".", "dart", "ooniengine")

	abiGoPath := filepath.Join(".", "pkg", "ooniengine", "abi.go")
	abiGo := openFile(abiGoPath)
	generateABIGo(abiGo, abiVersion, OONIEngine)
	closeFile(abiGo)
	fmt.Printf("GEN %s\n", abiGoPath)
	execute("go", "fmt", abiGoPath)

	abiDartPath := filepath.Join(dartDir, "lib", "abi.dart")
	abiDart := openFile(abiDartPath)
	generateABIDart(abiDart, abiVersion, OONIEngine)
	closeFile(abiDart)
	fmt.Printf("GEN %s\n", abiDartPath)
	execute("dart", "format", abiDartPath)

	registryGoPath := filepath.Join(".", "pkg", "ooniengine", "registry.go")
	registryGo := openFile(registryGoPath)
	generateRegistryGo(registryGo, OONIEngine)
	closeFile(registryGo)
	fmt.Printf("GEN %s\n", registryGoPath)
	execute("go", "fmt", registryGoPath)

	tasksDartPath := filepath.Join(dartDir, "lib", "tasks.dart")
	tasksDart := openFile(tasksDartPath)
	generateTasksDart(tasksDart, OONIEngine)
	closeFile(tasksDart)
	fmt.Printf("GEN %s\n", tasksDartPath)
	execute("dart", "format", tasksDartPath)

	// Ensure both FFI and JSON wrapping code are up to date.
	chdirAndExecute(dartDir, "dart", "run", "ffigen")
	chdirAndExecute(dartDir, "dart", "run", "build_runner", "build")
}
