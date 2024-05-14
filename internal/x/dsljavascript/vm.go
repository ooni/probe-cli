package dsljavascript

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/util"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// VMConfig contains configuration for creating a VM.
type VMConfig struct {
	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// ScriptBaseDir is the MANDATORY script base dir to use.
	ScriptBaseDir string
}

// errVMConfig indicates that some setting in the [*VMConfig] is invalid.
var errVMConfig = errors.New("dsljavascript: invalid VMConfig")

// check returns an explanatory error if the [*VMConfig] is invalid.
func (cfg *VMConfig) check() error {
	if cfg.Logger == nil {
		return fmt.Errorf("%w: the Logger field is nil", errVMConfig)
	}

	if cfg.ScriptBaseDir == "" {
		return fmt.Errorf("%w: the ScriptBaseDir field is empty", errVMConfig)
	}

	return nil
}

// VM wraps the [*github.com/dop251/goja.Runtime]. The zero value of this
// struct is invalid; please, use [NewVM] to construct.
type VM struct {
	// logger is the logger to use.
	logger model.Logger

	// registry is the JavaScript package registry to use.
	registry *require.Registry

	// scriptBaseDir is the base directory containing scripts.
	scriptBaseDir string

	// util is a reference to goja's util model.
	util *goja.Object

	// vm is a reference to goja's runtime.
	vm *goja.Runtime
}

// RunScript runs the given script using a transient VM.
func RunScript(config *VMConfig, scriptPath string) error {
	// create a VM
	vm, err := NewVM(config, scriptPath)
	if err != nil {
		return err
	}

	// run the script
	return vm.RunScript(scriptPath)
}

// NewVM creates a new VM instance.
func NewVM(config *VMConfig, scriptPath string) (*VM, error) {
	// make sure the provided config is correct
	if err := config.check(); err != nil {
		return nil, err
	}

	// convert the script base dir to be an absolute path
	scriptBaseDir, err := filepath.Abs(config.ScriptBaseDir)
	if err != nil {
		return nil, err
	}

	// create package registry ("By default, a registry's global folders list is empty")
	registry := require.NewRegistry(require.WithGlobalFolders(scriptBaseDir))

	// create the goja virtual machine
	gojaVM := goja.New()

	// enable 'require' for the virtual machine
	registry.Enable(gojaVM)

	// create the virtual machine wrapper
	vm := &VM{
		logger:        config.Logger,
		registry:      registry,
		scriptBaseDir: scriptBaseDir,
		util:          require.Require(gojaVM, util.ModuleName).(*goja.Object),
		vm:            gojaVM,
	}

	// register the console module in JavaScript
	registry.RegisterNativeModule("console", vm.newModuleConsole)

	// make sure the 'console' object exists in the VM before running scripts
	runtimex.Try0(gojaVM.Set("console", require.Require(gojaVM, "console")))

	// register the _golang module in JavaScript
	registry.RegisterNativeModule("_golang", vm.newModuleGolang)

	// register the _ooni module in JavaScript
	registry.RegisterNativeModule("_ooni", vm.newModuleOONI)

	return vm, nil
}

// LoadExperiment loads the given experiment file and returns a new VM primed
// to execute the experiment several times for several inputs.
func LoadExperiment(config *VMConfig, exPath string) (*VM, error) {
	// create a new VM instance
	vm, err := NewVM(config, exPath)
	if err != nil {
		return nil, err
	}

	// make sure there's an empty dictionary containing exports
	runtimex.Try0(vm.vm.Set("exports", vm.vm.NewObject()))

	// run the script
	if err := vm.RunScript(exPath); err != nil {
		return nil, err
	}

	return vm, nil
}

func (vm *VM) RunScript(exPath string) error {
	// read the file content
	content, err := os.ReadFile(exPath) // #nosec G304 - this is working as intended
	if err != nil {
		return err
	}

	// interpret the script defining the experiment
	if _, err = vm.vm.RunScript(exPath, string(content)); err != nil {
		return err
	}

	return nil
}

func (vm *VM) findExportedSymbol(name string) (goja.Value, error) {
	// obtain the toplevel exports object
	value := vm.vm.Get("exports")
	if value == nil {
		return nil, errors.New("cannot find symbol: exports")
	}

	// convert to object
	exports := value.ToObject(vm.vm)
	if exports == nil {
		return nil, errors.New("cannot convert exports to object")
	}

	// obtain the symbol inside exports
	symbol := exports.Get(name)
	if symbol == nil {
		return nil, fmt.Errorf("cannot find symbol: exports.%s", name)
	}

	return symbol, nil
}

// ExperimentName returns the experiment name. Invoking this method
// before invoking LoadScript always produces an error.
func (vm *VM) ExperimentName() (string, error) {
	var experimentName func() (string, error)
	value, err := vm.findExportedSymbol("experimentName")
	if err != nil {
		return "", err
	}
	if err := vm.vm.ExportTo(value, &experimentName); err != nil {
		return "", err
	}
	return experimentName()
}

// ExperimentVersion returns the experiment version. Invoking this method
// before invoking LoadScript always produces an error.
func (vm *VM) ExperimentVersion() (string, error) {
	var experimentVersion func() (string, error)
	value, err := vm.findExportedSymbol("experimentVersion")
	if err != nil {
		return "", err
	}
	if err := vm.vm.ExportTo(value, &experimentVersion); err != nil {
		return "", err
	}
	return experimentVersion()
}

// Run performs a measurement and returns the test keys.
func (vm *VM) Run(input string) (string, error) {
	var run func(string) (string, error)
	value, err := vm.findExportedSymbol("run")
	if err != nil {
		return "", err
	}
	if err := vm.vm.ExportTo(value, &run); err != nil {
		return "", err
	}
	return run(input)
}
