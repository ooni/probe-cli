package dsljavascript

import (
	"time"

	"github.com/dop251/goja"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// newModuleGolang creates the _golang module in JavaScript
func (vm *VM) newModuleGolang(gojaVM *goja.Runtime, mod *goja.Object) {
	runtimex.Assert(vm.vm == gojaVM, "dsljavascript: unexpected gojaVM pointer value")
	exports := mod.Get("exports").(*goja.Object)
	runtimex.Try0(exports.Set("timeNow", vm.golangTimeNow))
}

// golangTimeNow returns the current time using golang [time.Now]
func (vm *VM) golangTimeNow(call goja.FunctionCall) goja.Value {
	runtimex.Assert(len(call.Arguments) == 0, "dsljavascript: _golang.timeNow expects zero arguments")
	return vm.vm.ToValue(time.Now())
}
