package compiler

import (
	"flowa/pkg/opcode"
	"testing"
)

func TestWhileLoops(t *testing.T) {
	tests := []compilerTestCase{
		{
			input: `
count = 0
while count < 5:
    count = count + 1
count
`,
			expectedConstants: []interface{}{5}, // Only need the 5 constant, 0 is now OpSetLocalZero
			expectedInstructions: []opcode.Instructions{
				// count = 0 (optimized with OpSetLocalZero)
				opcode.Make(opcode.OpSetLocalZero, 0),
				// while count < 5: (optimized with OpJumpIfLocalGreaterEqualConst)
				opcode.Make(opcode.OpJumpIfLocalGreaterEqualConst, 0, 0, 13), // if count >= 5 goto byte 13
				// count = count + 1 (optimized with OpIncLocal)
				opcode.Make(opcode.OpIncLocal, 0),
				opcode.Make(opcode.OpJump, 2), // jump back to byte 2
				// count (end of loop, byte 13)
				opcode.Make(opcode.OpGetLocal, 0),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}
