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
			expectedConstants: []interface{}{0, 5},
			expectedInstructions: []opcode.Instructions{
				// count = 0
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpSetLocal, 0),
				// while count < 5: (optimized with OpJumpIfLocalGreaterEqualConst)
				opcode.Make(opcode.OpJumpIfLocalGreaterEqualConst, 0, 1, 16), // if count >= 5 goto byte 16
				// count = count + 1 (optimized with OpIncLocal)
				opcode.Make(opcode.OpIncLocal, 0),
				opcode.Make(opcode.OpJump, 5), // jump back to byte 5
				// count (end of loop, byte 16)
				opcode.Make(opcode.OpGetLocal, 0),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}
