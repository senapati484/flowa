package benchmarks

import (
	"flowa/pkg/compiler"
	"flowa/pkg/eval"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"flowa/pkg/vm"
	"testing"
)

var result eval.Object

func BenchmarkVMAddition(b *testing.B) {
	input := `
5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		b.Fatal(err)
	}

	bytecode := comp.Bytecode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		machine := vm.New(bytecode)
		machine.Run()
		result = machine.LastPoppedStackElem()
	}
}

func BenchmarkTreeWalkAddition(b *testing.B) {
	input := `
5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := eval.NewEnvironment()
		result = eval.Eval(program, env)
	}
}

func BenchmarkVMComparison(b *testing.B) {
	input := "1 < 2"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		b.Fatal(err)
	}

	bytecode := comp.Bytecode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		machine := vm.New(bytecode)
		machine.Run()
		result = machine.LastPoppedStackElem()
	}
}

func BenchmarkTreeWalkComparison(b *testing.B) {
	input := "1 < 2"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := eval.NewEnvironment()
		result = eval.Eval(program, env)
	}
}
