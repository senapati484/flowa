package vm

import (
	"flowa/pkg/compiler"
)

// Reset resets the VM state for reuse, avoiding new allocations
func (vm *VM) Reset(bytecode *compiler.Bytecode) {
	vm.constants = bytecode.Constants
	vm.sp = 0

	// Reset main frame
	vm.currentFrame().fn.Instructions = bytecode.Instructions
	vm.currentFrame().ip = -1
	vm.framesIndex = 1

	// Clear globals (optional - could keep for persistent state)
	for i := range vm.globals {
		vm.globals[i] = nil
	}
}
