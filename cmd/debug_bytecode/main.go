package main

import (
	"flowa/pkg/compiler"
	"flowa/pkg/lexer"
	"flowa/pkg/opcode"
	"flowa/pkg/parser"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug_bytecode <file.flowa>")
		os.Exit(1)
	}

	content, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	comp := compiler.New()
	err = comp.Compile(program)
	if err != nil {
		fmt.Printf("Compilation failed: %s\n", err)
		os.Exit(1)
	}

	bytecode := comp.Bytecode()
	instructions := bytecode.Instructions

	// Find OpJumpIfLocalGreaterEqualConst instructions
	i := 0
	for i < len(instructions) {
		op := opcode.Opcode(instructions[i])
		if op == opcode.OpJumpIfLocalGreaterEqualConst {
			fmt.Printf("Found OpJumpIfLocalGreaterEqualConst at offset %d\n", i)
			fmt.Printf("Raw bytes: ")
			for j := 0; j < 6 && i+j < len(instructions); j++ {
				fmt.Printf("%02x ", instructions[i+j])
			}
			fmt.Println()

			// Decode manually
			localIdx := int(instructions[i+1])
			constIdx := int(opcode.ReadUint16(instructions[i+2:]))
			jumpPos := int(opcode.ReadUint16(instructions[i+4:]))
			fmt.Printf("Decoded: localIdx=%d, constIdx=%d, jumpPos=%d\n", localIdx, constIdx, jumpPos)
			fmt.Println()
		}

		// Skip to next instruction
		def, err := opcode.Lookup(byte(op))
		if err != nil {
			i++
			continue
		}
		i++
		for _, width := range def.OperandWidths {
			i += width
		}
	}
}
