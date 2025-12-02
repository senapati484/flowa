package compiler

import (
	"flowa/pkg/opcode"
	"testing"
)

func TestGlobalVariables(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "x = 1\nx",
			expectedConstants: []interface{}{1},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpSetLocal, 0),
				opcode.Make(opcode.OpGetLocal, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "x = 1\ny = 2\nx + y",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpSetLocal, 0),
				opcode.Make(opcode.OpConstant, 1),
				opcode.Make(opcode.OpSetLocal, 1),
				opcode.Make(opcode.OpGetLocal, 0),
				opcode.Make(opcode.OpGetLocal, 1),
				opcode.Make(opcode.OpAdd),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "x = 1\nx = 2\nx",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpSetLocal, 0),
				opcode.Make(opcode.OpConstant, 1),
				opcode.Make(opcode.OpSetLocal, 0),
				opcode.Make(opcode.OpGetLocal, 0),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}
