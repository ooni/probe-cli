package dsljavascript

//
// Adapted from github.com/dop251/goja_nodejs
//
// SPDX-License-Identifier: MIT
//

import (
	"github.com/dop251/goja"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// newModuleConsole creates the console module in JavaScript
func (vm *VM) newModuleConsole(gojaVM *goja.Runtime, mod *goja.Object) {
	runtimex.Assert(vm.vm == gojaVM, "dsljavascript: unexpected gojaVM pointer value")
	exports := mod.Get("exports").(*goja.Object)
	runtimex.Try0(exports.Set("log", vm.consoleLog))
	runtimex.Try0(exports.Set("error", vm.consoleError))
	runtimex.Try0(exports.Set("warn", vm.consoleWarn))
}

// consoleLog implements console.log
func (vm *VM) consoleLog(call goja.FunctionCall) goja.Value {
	return vm.consoleDo(call, vm.logger.Info)
}

// consoleError implements console.Error
func (vm *VM) consoleError(call goja.FunctionCall) goja.Value {
	return vm.consoleDo(call, vm.logger.Warn)
}

// consoleWarn implements console.Warn
func (vm *VM) consoleWarn(call goja.FunctionCall) goja.Value {
	return vm.consoleDo(call, vm.logger.Warn)
}

func (vm *VM) consoleDo(call goja.FunctionCall, emit func(msg string)) goja.Value {
	format, ok := goja.AssertFunction(vm.util.Get("format"))
	runtimex.Assert(ok, "dsljavascript: util.format is not a function")
	ret := runtimex.Try1(format(vm.util, call.Arguments...))
	emit(ret.String())
	return nil
}
