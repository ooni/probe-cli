package dsljavascript

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dop251/goja"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/x/dslengine"
	"github.com/ooni/probe-cli/v3/internal/x/dsljson"
)

// newModuleOONI creates the console module in JavaScript
func (vm *VM) newModuleOONI(gojaVM *goja.Runtime, mod *goja.Object) {
	runtimex.Assert(vm.vm == gojaVM, "dsljavascript: unexpected gojaVM pointer value")
	exports := mod.Get("exports").(*goja.Object)
	exports.Set("runDSL", vm.ooniRunDSL)
}

func (vm *VM) ooniRunDSL(jsAST *goja.Object, zeroTime time.Time) (string, error) {
	// serialize the incoming JS object
	rawAST, err := jsAST.MarshalJSON()
	if err != nil {
		return "", err
	}

	// parse the raw AST into the loadable AST format
	var root dsljson.RootNode
	if err := json.Unmarshal(rawAST, &root); err != nil {
		return "", err
	}

	// create a background context for now but ideally we should allow to interrupt
	ctx := context.Background()

	// create a runtime for executing the DSL
	// TODO(bassosimone): maybe we should configure the parallelism?
	rtx := dslengine.NewRuntimeMeasurexLite(
		vm.logger, zeroTime,
		dslengine.OptionMaxActiveDNSLookups(4),
		dslengine.OptionMaxActiveConns(16),
	)

	// interpret the JSON representation of the DSL
	if err := dsljson.Run(ctx, rtx, &root); err != nil {
		return "", err
	}

	// serialize the observations to JSON and return
	resultRaw := runtimex.Try1(json.Marshal(rtx.Observations()))
	return string(resultRaw), nil
}
