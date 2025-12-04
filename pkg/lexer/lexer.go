package lexer

import (
	"flowa/pkg/token"
	"strings"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int

	indentStack []int // Stack of indentation levels (column numbers)
	tokenQueue  []token.Token
}

func New(input string) *Lexer {
	l := &Lexer{
		input:       input,
		line:        1,
		column:      0,
		indentStack: []int{0},
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
	l.column += 1
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	// If we have queued tokens (e.g. DEDENTs), return them first
	if len(l.tokenQueue) > 0 {
		tok := l.tokenQueue[0]
		l.tokenQueue = l.tokenQueue[1:]
		return tok
	}

	var tok token.Token

	// Skip whitespace but NOT newlines
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}

	// Skip comments
	if l.ch == '#' {
		l.skipComment()
		// After skipping comment, we're at newline or EOF
		// Continue to next token
		return l.NextToken()
	}

	switch l.ch {
	case '\n':
		return l.handleNewline()
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.ASSIGN, l.ch, l.line, l.column)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.PLUS_PLUS, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.PLUS, l.ch, l.line, l.column)
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.ARROW, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MINUS_MINUS, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.MINUS, l.ch, l.line, l.column)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.BANG, l.ch, l.line, l.column)
		}
	case '/':
		tok = newToken(token.SLASH, l.ch, l.line, l.column)
	case '*':
		tok = newToken(token.ASTERISK, l.ch, l.line, l.column)
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.LT, l.ch, l.line, l.column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.GT, l.ch, l.line, l.column)
		}
	case '|':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.PIPE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, l.line, l.column)
		}
	case ',':
		tok = newToken(token.COMMA, l.ch, l.line, l.column)
	case ':':
		tok = newToken(token.COLON, l.ch, l.line, l.column)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, l.line, l.column)
	case '(':
		tok = newToken(token.LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(token.RPAREN, l.ch, l.line, l.column)
	case '.':
		tok = newToken(token.DOT, l.ch, l.line, l.column)
	case '{':
		tok = newToken(token.LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(token.RBRACE, l.ch, l.line, l.column)
	case '[':
		tok = newToken(token.LBRACKET, l.ch, l.line, l.column)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, l.line, l.column)
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
		tok.Line = l.line
		tok.Column = l.column
	case 0:
		// Handle EOF: dedent remaining
		if len(l.indentStack) > 1 {
			for i := 0; i < len(l.indentStack)-1; i++ {
				l.tokenQueue = append(l.tokenQueue, token.Token{Type: token.DEDENT, Literal: "", Line: l.line, Column: l.column})
			}
			l.indentStack = []int{0}
			tok = l.tokenQueue[0]
			l.tokenQueue = l.tokenQueue[1:]
		} else {
			tok.Literal = ""
			tok.Type = token.EOF
		}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			// fmt.Printf("DEBUG: Ident=%q Type=%q\n", tok.Literal, tok.Type)
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) handleNewline() token.Token {
	// 1. Emit NEWLINE
	tok := token.Token{Type: token.NEWLINE, Literal: "\n", Line: l.line, Column: l.column}

	l.readChar() // Consume \n
	l.line++
	l.column = 0 // Reset column for new line (will be 1 after next readChar if we didn't do it here, but we are consuming manually)
	// Actually readChar increments column. So if we just consumed \n, column is 1?
	// Let's reset column to 0, so next char is 1.
	l.column = 0

	// 2. Check indentation
	// Peek ahead to count spaces
	indentLen := 0
	for l.ch == ' ' || l.ch == '\t' {
		if l.ch == '\t' {
			indentLen += 4 // Assume tab is 4 spaces for now
		} else {
			indentLen++
		}
		l.readChar()
	}

	// If line is empty or comment, ignore indentation change
	if l.ch == '\n' || l.ch == 0 {
		// Just a blank line, maybe emit another newline or ignore?
		// Usually we ignore blank lines in indentation logic.
		// But we already emitted NEWLINE.
		// Let's just return the NEWLINE we created.
		return tok
	}

	currentIndent := l.indentStack[len(l.indentStack)-1]

	if indentLen > currentIndent {
		l.indentStack = append(l.indentStack, indentLen)
		l.tokenQueue = append(l.tokenQueue, token.Token{Type: token.INDENT, Literal: "", Line: l.line, Column: l.column})
	} else if indentLen < currentIndent {
		for len(l.indentStack) > 1 && indentLen < l.indentStack[len(l.indentStack)-1] {
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			l.tokenQueue = append(l.tokenQueue, token.Token{Type: token.DEDENT, Literal: "", Line: l.line, Column: l.column})
		}
		if indentLen != l.indentStack[len(l.indentStack)-1] {
			// Indentation error
			return token.Token{Type: token.ILLEGAL, Literal: "Indentation Error", Line: l.line, Column: l.column}
		}
	}

	return tok
}

func newToken(tokenType token.TokenType, ch byte, line, col int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Line: line, Column: col}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	// Support float?
	if l.ch == '.' {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) readString() string {
	var result strings.Builder
	l.readChar() // Skip opening quote

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			// Handle escape sequences
			l.readChar()
			switch l.ch {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '0':
				result.WriteByte('\x00')
			default:
				// Unknown escape, just include the backslash and character
				result.WriteByte('\\')
				result.WriteByte(l.ch)
			}
		} else {
			result.WriteByte(l.ch)
		}
		l.readChar()
	}

	return result.String()
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}
