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
		fmt.Println("Flowa Compiler v0.1 (MVP)")
		fmt.Println("Usage:")
		fmt.Println("  flowac repl      - Start REPL")
		fmt.Println("  flowac run <file> - Run a file")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "repl":
		startREPL()
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: flowac run <file>")
			os.Exit(1)
		}
		runFile(os.Args[2])
	default:
		fmt.Println("Unknown command:", os.Args[1])
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
