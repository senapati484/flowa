package main

import (
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run debug_parser.go '<code>'")
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
		fmt.Println()
	}

	if program != nil {
		fmt.Printf("AST:\n%s\n", program.String())
	}
}
