package benchmarks

import (
	"flowa/pkg/compiler"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"flowa/pkg/vm"
	"testing"
)

// Go native benchmarks for comparison
func BenchmarkGoAddition(b *testing.B) {
	var result int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5 + 5
	}
	_ = result
}

func BenchmarkGoComparison(b *testing.B) {
	var result bool
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = 1 < 2
	}
	_ = result
}

// Benchmark with VM instance reuse
func BenchmarkVMAdditionReuse(b *testing.B) {
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
	machine := vm.New(bytecode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		machine.Reset(bytecode)
		err := machine.Run()
		if err != nil {
			b.Fatal(err)
		}
		result = machine.LastPoppedStackElem()
	}
}
