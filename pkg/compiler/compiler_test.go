package compiler

import (
	"flowa/pkg/ast"
	"flowa/pkg/eval"
	"flowa/pkg/lexer"
	"flowa/pkg/opcode"
	"flowa/pkg/parser"
	"fmt"
	"testing"
)

type compilerTestCase struct {
	input                string
	expectedConstants    []interface{}
	expectedInstructions []opcode.Instructions
}

func TestIntegerArithmetic(t *testing.T) {
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
			input:             "1 - 2",
			expectedConstants: []interface{}{-1},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "1 * 2",
			expectedConstants: []interface{}{2},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "2 / 1",
			expectedConstants: []interface{}{2},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpConstant, 0),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestBooleanExpressions(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "true",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpTrue),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "false",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpFalse),
				opcode.Make(opcode.OpPop),
			},
		},
		{
			input:             "1 > 2",
			expectedConstants: []interface{}{},
			expectedInstructions: []opcode.Instructions{
				opcode.Make(opcode.OpFalse),
				opcode.Make(opcode.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func runCompilerTests(t *testing.T, tests []compilerTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		compiler := New()
		err := compiler.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		bytecode := compiler.Bytecode()

		err = testInstructions(tt.expectedInstructions, bytecode.Instructions)
		if err != nil {
			t.Fatalf("testInstructions failed: %s", err)
		}

		err = testConstants(t, tt.expectedConstants, bytecode.Constants)
		if err != nil {
			t.Fatalf("testConstants failed: %s", err)
		}
	}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testInstructions(expected []opcode.Instructions, actual opcode.Instructions) error {
	concatenated := concatInstructions(expected)

	if len(actual) != len(concatenated) {
		return fmt.Errorf("wrong instructions length.\nwant=%q\ngot =%q",
			concatenated, actual)
	}

	for i, ins := range concatenated {
		if actual[i] != ins {
			return fmt.Errorf("wrong instruction at %d.\nwant=%q\ngot =%q",
				i, concatenated, actual)
		}
	}

	return nil
}

func concatInstructions(s []opcode.Instructions) opcode.Instructions {
	out := opcode.Instructions{}

	for _, ins := range s {
		out = append(out, ins...)
	}

	return out
}

func testConstants(t *testing.T, expected []interface{}, actual []eval.Object) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("wrong number of constants. want=%d, got=%d",
			len(expected), len(actual))
	}

	for i, constant := range expected {
		switch constant := constant.(type) {
		case int:
			err := testIntegerObject(int64(constant), actual[i])
			if err != nil {
				return fmt.Errorf("constant %d - testIntegerObject failed: %s",
					i, err)
			}
		case []opcode.Instructions:
			fn, ok := actual[i].(*eval.Function)
			if !ok {
				return fmt.Errorf("constant %d - not a function: %T", i, actual[i])
			}
			err := testInstructions(constant, fn.Instructions)
			if err != nil {
				return fmt.Errorf("constant %d - instructions mismatch: %s", i, err)
			}
		}
	}

	return nil
}

func testIntegerObject(expected int64, actual eval.Object) error {
	result, ok := actual.(*eval.Integer)
	if !ok {
		return fmt.Errorf("object is not Integer. got=%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. want=%d, got=%d",
			expected, result.Value)
	}

	return nil
}
