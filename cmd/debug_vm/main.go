package main

import (
	"flowa/pkg/compiler"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"flowa/pkg/vm"
	"fmt"
)

func main() {
	input := "x = 1\ny = 2\nx + y"

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		panic(err)
	}

	bytecode := comp.Bytecode()

	fmt.Printf("Constants: %d\n", len(bytecode.Constants))
	for i, c := range bytecode.Constants {
		fmt.Printf("  [%d] = %s\n", i, c.Inspect())
	}

	fmt.Printf("\nInstructions (%d bytes):\n", len(bytecode.Instructions))
	for i := 0; i < len(bytecode.Instructions); i++ {
		fmt.Printf("%02d: %02x\n", i, bytecode.Instructions[i])
	}

	machine := vm.New(bytecode)
	err = machine.Run()
	if err != nil {
		panic(err)
	}

	result := machine.LastPoppedStackElem()
	fmt.Printf("\nResult: %s\n", result.Inspect())
}
