package main

import (
	"flowa/pkg/compiler"
	"flowa/pkg/lexer"
	"flowa/pkg/opcode"
	"flowa/pkg/parser"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run inspect_bytecode.go '<code>'")
		os.Exit(1)
	}

	input := os.Args[1]
	l := lexer.New(input)
	p := parser.New(l)

	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		fmt.Println("Parser errors:")
		for _, msg := range p.Errors() {
			fmt.Printf("  %s\n", msg)
		}
		os.Exit(1)
	}

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		fmt.Printf("Compiler error: %s\n", err)
		os.Exit(1)
	}

	bytecode := comp.Bytecode()

	fmt.Printf("Constants (%d):\n", len(bytecode.Constants))
	for i, c := range bytecode.Constants {
		fmt.Printf("  [%d] %s\n", i, c.Inspect())
	}
	fmt.Println()

	fmt.Printf("Instructions (%d bytes):\n", len(bytecode.Instructions))
	ins := bytecode.Instructions
	i := 0
	for i < len(ins) {
		def, err := opcode.Lookup(ins[i])
		if err != nil {
			fmt.Printf("%04d ERROR: %s\n", i, err)
			i++
			continue
		}

		operands, read := opcode.ReadOperands(def, ins[i+1:])
		fmt.Printf("%04d %s", i, def.Name)

		for _, op := range operands {
			fmt.Printf(" %d", op)
		}
		fmt.Println()

		// Print hex dump for this instruction
		fmt.Printf("     Raw: ")
		for k := 0; k < 1+read; k++ {
			fmt.Printf("%02x ", ins[i+k])
		}
		fmt.Println()

		i += 1 + read
	}
}
