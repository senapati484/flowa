package compiler

import (
	"flowa/pkg/opcode"
	"testing"
)

func TestConstantFolding(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "1 + 2",
			expectedConstants: []interface{}{3},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "5 * 10",
			expectedConstants: []interface{}{50},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "100 - 50",
			expectedConstants: []interface{}{50},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "20 / 4",
			expectedConstants: []interface{}{5},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "5 > 3",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpTrue),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "2 < 1",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpFalse),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "10 == 10",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpTrue),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "5 != 3",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpTrue),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}
