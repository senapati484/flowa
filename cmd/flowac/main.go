package main

import (
	"bufio"
	"flowa/pkg/ast"
	"flowa/pkg/eval"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"fmt"
	"io"
	"os"
)

const PROMPT = ">>> "

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Flowa Programming Language v0.1 (MVP)")
		fmt.Println("Usage:")
		fmt.Println("  flowa <file.flowa>  - Run a Flowa script")
		fmt.Println("  flowa repl          - Start interactive REPL")
		os.Exit(1)
	}

	command := os.Args[1]

	// If the first argument ends with .flowa, treat it as a file to run
	if len(command) > 6 && command[len(command)-6:] == ".flowa" {
		runFile(command)
		return
	}

	// Otherwise, check for explicit commands
	switch command {
	case "repl":
		startREPL()
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: flowa run <file>")
			os.Exit(1)
		}
		runFile(os.Args[2])
	default:
		fmt.Println("Unknown command:", command)
		fmt.Println("Usage: flowa <file.flowa> or flowa repl")
		os.Exit(1)
	}
}

func startREPL() {
	scanner := bufio.NewScanner(os.Stdin)
	env := eval.NewEnvironment()

	fmt.Println("Flowa REPL v0.1 (MVP)")
	fmt.Println("Type expressions or statements and press Enter")

	for {
		fmt.Print(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(os.Stdout, p.Errors())
			continue
		}

		evaluated := eval.Eval(program, env)
		if evaluated != nil {
			io.WriteString(os.Stdout, evaluated.Inspect())
			io.WriteString(os.Stdout, "\n")
		}
	}
}

func runFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(data))
	p := parser.New(l)

	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		printParserErrors(os.Stderr, p.Errors())
		os.Exit(1)
	}

	env := eval.NewEnvironment()
	evaluated := eval.Eval(program, env)
	if evaluated != nil && evaluated.Type() == "ERROR" {
		fmt.Fprintf(os.Stderr, "%s\n", evaluated.Inspect())
		os.Exit(1)
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "Parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}

func printAST(node ast.Node) {
	fmt.Println(node.String())
}
