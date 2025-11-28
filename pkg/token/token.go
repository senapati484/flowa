package token

import "fmt"

type TokenType string

const (
	// Special
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"
	INDENT  = "INDENT"
	DEDENT  = "DEDENT"
	NEWLINE = "NEWLINE"

	// Identifiers & Literals
	IDENT  = "IDENT"  // main, x, y
	INT    = "INT"    // 123
	FLOAT  = "FLOAT"  // 123.45
	STRING = "STRING" // "hello"

	// Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	PIPE     = "|>" // Pipeline operator

	LT     = "<"
	GT     = ">"
	EQ     = "=="
	NOT_EQ = "!="
	LTE    = "<="
	GTE    = ">="

	// Delimiters
	COMMA     = ","
	COLON     = ":"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"
	ARROW     = "->"

	// Keywords
	DEF    = "DEF"
	ASYNC  = "ASYNC"
	RETURN = "RETURN"
	IF     = "IF"
	ELSE   = "ELSE"
	TRUE   = "TRUE"
	FALSE  = "FALSE"
	FOR    = "FOR"
	WHILE  = "WHILE"
	IN     = "IN"
	SPAWN  = "SPAWN"
	AWAIT  = "AWAIT"
	MODULE = "MODULE"
	IMPORT = "IMPORT"
	TYPE   = "TYPE"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Literal, t.Line, t.Column)
}

var keywords = map[string]TokenType{
	"def":    DEF,
	"async":  ASYNC,
	"return": RETURN,
	"if":     IF,
	"else":   ELSE,
	"true":   TRUE,
	"false":  FALSE,
	"for":    FOR,
	"while":  WHILE,
	"in":     IN,
	"spawn":  SPAWN,
	"await":  AWAIT,
	"module": MODULE,
	"import": IMPORT,
	"type":   TYPE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
