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
		fmt.Println("Usage: inspect_bytecode <filename>")
		return
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	input := string(content)

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	comp := compiler.New()
	err = comp.Compile(program)
	if err != nil {
		panic(err)
	}

	bytecode := comp.Bytecode()

	fmt.Printf("MainNumLocals: %d\n", bytecode.MainNumLocals)
	fmt.Printf("Constants: %d\n", len(bytecode.Constants))
	for i, c := range bytecode.Constants {
		fmt.Printf("  [%d] = %s\n", i, c.Inspect())
	}

	fmt.Printf("\nInstructions (%d bytes):\n", len(bytecode.Instructions))
	i := 0
	for i < len(bytecode.Instructions) {
		op := bytecode.Instructions[i]
		def, err := opcode.Lookup(op)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		fmt.Printf("%04d %s", i, def.Name)

		operands, read := opcode.ReadOperands(def, bytecode.Instructions[i+1:])
		for _, operand := range operands {
			fmt.Printf(" %d", operand)
		}
		fmt.Println()

		i += 1 + read
	}
}
