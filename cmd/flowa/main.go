package main

import (
	"bufio"
	"flowa/pkg/ast"
	"flowa/pkg/eval"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"flowa/pkg/version"
	"fmt"
	"io"
	"os"
	"strings"
)

const PROMPT = ">>> "

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		os.Exit(1)
	}

	if handled := handleGlobalFlags(args); handled {
		return
	}

	command := args[0]
	remaining := args[1:]

	if strings.HasSuffix(command, ".flowa") {
		runFile(command)
		return
	}

	switch command {
	case "repl":
		startREPL()
	case "run":
		if len(remaining) == 0 {
			fmt.Println("Usage: flowa run <file>")
			os.Exit(1)
		}
		runFile(remaining[0])
	case "inspect":
		if len(remaining) == 0 {
			fmt.Println("Usage: flowa inspect <file>")
			os.Exit(1)
		}
		inspectFile(remaining[0])
	case "pipelines":
		if len(remaining) == 0 {
			fmt.Println("Usage: flowa pipelines <file>")
			os.Exit(1)
		}
		printPipelineOverview(remaining[0])
	case "ast":
		if len(remaining) == 0 {
			fmt.Println("Usage: flowa ast <file>")
			os.Exit(1)
		}
		printProgramAST(remaining[0])
	case "version":
		printVersion()
	case "help":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printHelp()
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
	program, parserErrors, err := parseProgramFromFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	if len(parserErrors) != 0 {
		printParserErrors(os.Stderr, parserErrors)
		os.Exit(1)
	}

	env := eval.NewEnvironment()
	evaluated := eval.Eval(program, env)
	if evaluated != nil && evaluated.Type() == "ERROR" {
		fmt.Fprintf(os.Stderr, "%s\n", evaluated.Inspect())
		os.Exit(1)
	}
}

func inspectFile(filename string) {
	program, parserErrors, err := parseProgramFromFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	if len(parserErrors) != 0 {
		printParserErrors(os.Stderr, parserErrors)
		os.Exit(1)
	}

	insights := analyzeProgram(program)
	printFunctionInsights(insights.Functions)
	printPipelineInsights(insights.Pipelines)
}

func printPipelineOverview(filename string) {
	program, parserErrors, err := parseProgramFromFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	if len(parserErrors) != 0 {
		printParserErrors(os.Stderr, parserErrors)
		os.Exit(1)
	}

	insights := analyzeProgram(program)
	printPipelineInsights(insights.Pipelines)
}

func printProgramAST(filename string) {
	program, parserErrors, err := parseProgramFromFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	if len(parserErrors) != 0 {
		printParserErrors(os.Stderr, parserErrors)
		os.Exit(1)
	}
	printAST(program)
}

func parseProgramFromFile(filename string) (*ast.Program, []string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	l := lexer.New(string(data))
	p := parser.New(l)

	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) != 0 {
		return nil, errs, nil
	}

	return program, nil, nil
}

func printFunctionInsights(functions []FunctionInfo) {
	fmt.Printf("Functions (%d)\n", len(functions))
	if len(functions) == 0 {
		fmt.Println("  · No function definitions found.")
		return
	}

	for _, fn := range functions {
		marker := "def"
		if fn.IsAsync {
			marker = "async def"
		}
		fmt.Printf("  · %s %s(%s)\n", marker, fn.Name, strings.Join(fn.Parameters, ", "))
	}
}

func printPipelineInsights(pipelines []PipelineInfo) {
	fmt.Printf("Pipelines (%d)\n", len(pipelines))
	if len(pipelines) == 0 {
		fmt.Println("  · No pipelines detected. Use |> to chain transformations!")
		return
	}

	for i, pipe := range pipelines {
		fmt.Printf("  · Pipeline %d: %s\n", i+1, strings.Join(pipe.Chain, " |> "))
	}
}

func printVersion() {
	fmt.Printf("Flowa %s\n", version.Version)
	fmt.Printf("Build Date: %s\n", version.BuildDate)
	fmt.Printf("Git Commit: %s\n", version.GitCommit)
}

func printHelp() {
	fmt.Println("Flowa — pipeline-first programming language")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  flowa <file.flowa>      Run a Flowa script (shortcut for 'flowa run')")
	fmt.Println("  flowa run <file>        Execute a script")
	fmt.Println("  flowa repl              Start the interactive REPL")
	fmt.Println("  flowa inspect <file>    Summarize functions and pipelines")
	fmt.Println("  flowa pipelines <file>  Render pipeline chains")
	fmt.Println("  flowa ast <file>        Print the program AST")
	fmt.Println("  flowa version           Display build metadata")
	fmt.Println("  flowa help              Show this help message")
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  --help, -h              Show help")
	fmt.Println("  --version, -v           Show version")
}

func handleGlobalFlags(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--version", "-v":
			printVersion()
			return true
		case "--help", "-h":
			printHelp()
			return true
		}
	}
	return false
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
