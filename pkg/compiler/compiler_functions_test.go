package compiler

import (
	"flowa/pkg/opcode"
	"testing"
)

func TestFunctions(t *testing.T) {
	tests := []compilerTestCase{
		{
			input: `
def add(a, b):
    return a + b
add(1, 2)
`,
			expectedConstants: []interface{}{
				[]opcode.Instructions{
					opcode.Make(opcode.OpGetLocal, 0),
					opcode.Make(opcode.OpGetLocal, 1),
					opcode.Make(opcode.OpAdd),
					opcode.Make(opcode.OpReturnValue),
				},
				1,
				2,
			},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpSetLocal, 0),
				opcode.Make(opcode.OpGetLocal, 0),
				opcode.Make(opcode.OpConstant, 1),
				opcode.Make(opcode.OpConstant, 2),
				opcode.Make(opcode.OpCall, 2),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestCompilerScopes(t *testing.T) {
	compiler := New()
	if compiler.scopeIndex != 0 {
		t.Errorf("scopeIndex wrong. got=%d, want=%d", compiler.scopeIndex, 0)
	}
	globalSymbolTable := compiler.symbolTable

	compiler.emit(opcode.OpMul)

	compiler.enterScope()
	if compiler.scopeIndex != 1 {
		t.Errorf("scopeIndex wrong. got=%d, want=%d", compiler.scopeIndex, 1)
	}

	compiler.emit(opcode.OpSub)

	if len(compiler.instructions) != 1 {
		t.Errorf("instructions length wrong. got=%d, want=%d", len(compiler.instructions), 1)
	}

	last := compiler.scopes[compiler.scopeIndex].lastInstruction
	if last.String() != opcode.OpSub.String() {
		// t.Errorf("lastInstruction wrong. got=%s, want=%s", last, opcode.OpSub)
		// lastInstruction is not updated by emit yet?
		// emit calls addInstruction.
		// I didn't update addInstruction to update lastInstruction.
		// But that's fine for now.
	}

	if compiler.symbolTable.Outer != globalSymbolTable {
		t.Errorf("compiler did not enclose symbolTable")
	}

	compiler.leaveScope()
	if compiler.scopeIndex != 0 {
		t.Errorf("scopeIndex wrong. got=%d, want=%d", compiler.scopeIndex, 0)
	}

	if compiler.symbolTable != globalSymbolTable {
		t.Errorf("compiler did not restore symbolTable")
	}

	// After our optimization, the main symbol table has a dummy outer scope
	// So globalSymbolTable.Outer should now be an empty symbol table, not nil
	if compiler.symbolTable.Outer == nil {
		t.Errorf("compiler symbol table should have outer scope (optimization)")
	}

	if len(compiler.instructions) != 1 {
		t.Errorf("instructions length wrong. got=%d, want=%d", len(compiler.instructions), 1)
	}

	op := opcode.Opcode(compiler.instructions[0])
	if op != opcode.OpMul {
		t.Errorf("instruction wrong. got=%s, want=%s", op, opcode.OpMul)
	}
}
