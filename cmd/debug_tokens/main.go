package main

import (
	"flowa/pkg/lexer"
	"flowa/pkg/token"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run debug_tokens.go '<code>'")
		os.Exit(1)
	}

	input := os.Args[1]
	l := lexer.New(input)

	fmt.Printf("Input: %s\n\n", input)
	fmt.Println("Tokens:")
	fmt.Println("-------")

	for {
		tok := l.NextToken()
		fmt.Printf("%-15s %-20s (line %d, col %d)\n", tok.Type, fmt.Sprintf("'%s'", tok.Literal), tok.Line, tok.Column)

		if tok.Type == token.EOF {
			break
		}
	}
}
