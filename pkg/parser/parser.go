package parser

import (
	"flowa/pkg/ast"
	"flowa/pkg/lexer"
	"flowa/pkg/token"
	"fmt"
	"strconv"
	"strings"
)

const (
	_ int = iota
	LOWEST
	PIPELINE    // |>
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	MEMBER      // object.property
	POSTFIX     // i++
)

var precedences = map[token.TokenType]int{
	token.EQ:          EQUALS,
	token.NOT_EQ:      EQUALS,
	token.LT:          LESSGREATER,
	token.GT:          LESSGREATER,
	token.LTE:         LESSGREATER, // <=
	token.GTE:         LESSGREATER, // >=
	token.PLUS:        SUM,
	token.MINUS:       SUM,
	token.SLASH:       PRODUCT,
	token.ASTERISK:    PRODUCT,
	token.LPAREN:      CALL,
	token.DOT:         MEMBER,
	token.LBRACKET:    MEMBER, // Bracket access has same precedence as member access
	token.PIPE:        PIPELINE,
	token.PLUS_PLUS:   POSTFIX,
	token.MINUS_MINUS: POSTFIX,
}

func (p *Parser) parseForClauseStatement() ast.Statement {
	if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.ASSIGN) {
		return p.parseAssignmentStatement()
	}

	startToken := p.curToken
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	return &ast.ExpressionStatement{Token: startToken, Expression: expr}
}

func (p *Parser) parsePostfixExpression(left ast.Expression) ast.Expression {
	return &ast.PostfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
}

func (p *Parser) parseClassicForStatement() *ast.ClassicForStatement {
	stmt := &ast.ClassicForStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// Move to first token inside parentheses
	p.nextToken()

	// Initialization statement (optional)
	if !p.curTokenIs(token.SEMICOLON) {
		stmt.Init = p.parseForClauseStatement()
		if stmt.Init == nil {
			return nil
		}
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}

	// Move to condition
	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		stmt.Condition = p.parseExpression(LOWEST)
		if stmt.Condition == nil {
			return nil
		}
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}

	// Move to post statement or closing paren
	p.nextToken()
	if !p.curTokenIs(token.RPAREN) {
		stmt.Post = p.parseForClauseStatement()
		if stmt.Post == nil {
			return nil
		}
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}

	if !p.curTokenIs(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBraceBlockStatement()
	return stmt
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.LBRACE, p.parseMapLiteral)     // Map literals
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral) // Array literals
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.SPAWN, p.parseSpawnExpression)
	p.registerPrefix(token.SPAWN, p.parseSpawnExpression)
	p.registerPrefix(token.AWAIT, p.parseAwaitExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.NONE, p.parseNull)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LTE, p.parseInfixExpression) // <=
	p.registerInfix(token.GTE, p.parseInfixExpression) // >=
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.PIPE, p.parsePipelineExpression)
	p.registerInfix(token.DOT, p.parseMemberExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression) // NEW: bracket access
	p.registerInfix(token.PLUS_PLUS, p.parsePostfixExpression)
	p.registerInfix(token.MINUS_MINUS, p.parsePostfixExpression)

	// Server keywords as identifiers
	p.registerPrefix(token.GET, p.parseIdentifier)
	p.registerPrefix(token.POST, p.parseIdentifier)
	p.registerPrefix(token.PUT, p.parseIdentifier)
	p.registerPrefix(token.DELETE, p.parseIdentifier)
	p.registerPrefix(token.WS, p.parseIdentifier)
	p.registerPrefix(token.USE, p.parseIdentifier)
	p.registerPrefix(token.SERVICE, p.parseIdentifier)
	p.registerPrefix(token.ON, p.parseIdentifier)
	p.registerPrefix(token.ON, p.parseIdentifier)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.RETURN:
		return p.parseReturnStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.FUNC:
		return p.parseFunctionStatement()
	case token.ASYNC:
		if p.peekTokenIs(token.FUNC) {
			return p.parseFunctionStatement()
		}
		return nil
	case token.WHILE:
		return p.parseWhileStatement()
	case token.FOR:
		if p.peekTokenIs(token.LPAREN) {
			return p.parseClassicForStatement()
		}
		return p.parseForStatement()
	case token.MODULE:
		return p.parseModuleStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.FROM:
		return p.parseFromImportStatement()
	case token.TYPE:
		return p.parseTypeStatement()
	case token.SERVICE:
		return p.parseServiceStatement()
	case token.GET, token.POST, token.PUT, token.DELETE, token.WS:
		return p.parseRouteStatement()
	case token.USE:
		return p.parseMiddlewareStatement()
	case token.DEFER:
		return p.parseDeferStatement()
	case token.NEWLINE:
		return nil
	case token.IDENT:
		// Check if this is an assignment
		if p.peekTokenIs(token.ASSIGN) {
			return p.parseAssignmentStatement()
		}
		fallthrough
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{Token: p.curToken}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken() // move to value
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	if !p.curTokenIs(token.NEWLINE) && !p.curTokenIs(token.EOF) {
		stmt.ReturnValue = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.curToken}

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {
	stmt := &ast.FunctionStatement{Token: p.curToken}
	if p.curTokenIs(token.ASYNC) {
		stmt.IsAsync = true
		if !p.expectPeek(token.FUNC) {
			return nil
		}
	}
	// now curToken is FUNC
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseFunctionParameters()

	// Expect opening brace for function body
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBraceBlockStatement()

	return stmt
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	if !p.expectPeek(token.INDENT) {
		return nil
	}
	p.nextToken() // consume INDENT

	for !p.curTokenIs(token.DEDENT) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseBraceBlockStatement parses a brace-delimited block (for functions)
func (p *Parser) parseBraceBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken() // consume LBRACE

	// Skip any newlines after opening brace
	for p.curTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.NEWLINE) || p.curTokenIs(token.INDENT) || p.curTokenIs(token.DEDENT) {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.curToken}

	// Check if parentheses are used (optional)
	hasParens := p.peekTokenIs(token.LPAREN)

	if hasParens {
		// Parse with parentheses: while(condition){}
		p.nextToken() // consume LPAREN
		p.nextToken()
		stmt.Condition = p.parseExpression(LOWEST)

		// Expect closing paren
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	} else {
		// Parse without parentheses: while condition{}
		p.nextToken()
		stmt.Condition = p.parseExpression(LOWEST)
	}

	// Expect opening brace
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBraceBlockStatement()

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Iterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.IN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Expect opening brace
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBraceBlockStatement()

	return stmt
}

func (p *Parser) parseModuleStatement() *ast.ModuleStatement {
	stmt := &ast.ModuleStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.NEWLINE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseImportStatement() ast.Statement {
	// Check if it's `import { ... } from "path"`
	if p.peekTokenIs(token.LBRACE) {
		stmt := &ast.FromImportStatement{Token: p.curToken} // Reuse FromImportStatement
		p.nextToken()                                       // consume import

		// Parse symbols inside { ... }
		p.nextToken() // consume {

		if !p.curTokenIs(token.IDENT) {
			return nil
		}

		stmt.Symbols = append(stmt.Symbols, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

		for p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			stmt.Symbols = append(stmt.Symbols, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		}

		if !p.expectPeek(token.RBRACE) {
			return nil
		}

		if !p.expectPeek(token.FROM) {
			return nil
		}

		if !p.expectPeek(token.STRING) {
			return nil
		}

		stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		return stmt
	}

	// Normal `import "path"`
	stmt := &ast.ImportStatement{Token: p.curToken}

	if !p.expectPeek(token.STRING) {
		return nil
	}

	stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	return stmt
}

func (p *Parser) parseFromImportStatement() ast.Statement {
	stmt := &ast.FromImportStatement{Token: p.curToken}

	// from "path"
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	// import
	if !p.expectPeek(token.IMPORT) {
		return nil
	}

	// Check for wildcard *
	if p.peekTokenIs(token.ASTERISK) {
		p.nextToken()
		stmt.ImportAll = true
		return stmt
	}

	// symbol, symbol, ...
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Symbols = append(stmt.Symbols, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		stmt.Symbols = append(stmt.Symbols, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	return stmt
}

func (p *Parser) parseTypeStatement() *ast.TypeStatement {
	stmt := &ast.TypeStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.NEWLINE) {
		return nil
	}

	if !p.expectPeek(token.INDENT) {
		return nil
	}
	p.nextToken() // consume INDENT

	// Parse field names
	stmt.Fields = []*ast.Identifier{}
	for !p.curTokenIs(token.DEDENT) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			field := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			stmt.Fields = append(stmt.Fields, field)
		}
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseServiceStatement() *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ON) {
		return nil
	}

	if !p.expectPeek(token.STRING) {
		return nil
	}

	stmt.Address = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.NEWLINE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseRouteStatement() *ast.RouteStatement {
	stmt := &ast.RouteStatement{Token: p.curToken, Method: p.curToken.Literal}

	if !p.expectPeek(token.STRING) {
		return nil
	}

	stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ARROW) {
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Handler = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseMiddlewareStatement() *ast.MiddlewareStatement {
	stmt := &ast.MiddlewareStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Middleware = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	// Try to parse as possible assignment
	if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.ASSIGN) {
		// This is an assignment
		return nil // We'll handle this in parseStatement
	}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.NEWLINE) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	// Check if this is actually a float (contains a dot)
	if strings.Contains(p.curToken.Literal, ".") {
		lit := &ast.FloatLiteral{Token: p.curToken}
		value, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
			p.errors = append(p.errors, msg)
			return nil
		}
		lit.Value = value
		return lit
	}

	// Parse as integer
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNull() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseCallArguments()
	return exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := []ast.Expression{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseDeferStatement() *ast.DeferStatement {
	stmt := &ast.DeferStatement{Token: p.curToken}

	p.nextToken()
	stmt.Call = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	// Check if parentheses are used (optional)
	hasParens := p.peekTokenIs(token.LPAREN)

	if hasParens {
		// Parse with parentheses: if(condition){}
		p.nextToken() // consume LPAREN
		p.nextToken()
		expression.Condition = p.parseExpression(LOWEST)

		// Expect closing paren
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	} else {
		// Parse without parentheses: if condition{}
		p.nextToken()
		expression.Condition = p.parseExpression(LOWEST)
	}

	// Expect opening brace
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBraceBlockStatement()

	// Handle ELIF and ELSE
	// Skip any newlines/whitespace between blocks
	for p.peekTokenIs(token.NEWLINE) || p.peekTokenIs(token.INDENT) || p.peekTokenIs(token.DEDENT) {
		p.nextToken()
	}

	if p.peekTokenIs(token.ELIF) {
		p.nextToken() // consume ELIF
		// Parse as a new IfExpression (recursively)
		elifExpr := p.parseIfExpression()
		expression.Alternative = &ast.ExpressionStatement{
			Token:      elifExpr.(*ast.IfExpression).Token,
			Expression: elifExpr,
		}
		return expression
	}

	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // consume ELSE

		// Expect opening brace
		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBraceBlockStatement()
	}

	return expression
}

func (p *Parser) parsePipelineExpression(left ast.Expression) ast.Expression {
	expression := &ast.PipelineExpression{
		Token: p.curToken,
		Left:  left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseSpawnExpression() ast.Expression {
	expression := &ast.SpawnExpression{Token: p.curToken}
	p.nextToken()
	expression.Call = p.parseExpression(LOWEST)
	return expression
}

func (p *Parser) parseAwaitExpression() ast.Expression {
	expression := &ast.AwaitExpression{Token: p.curToken}
	p.nextToken()
	expression.Value = p.parseExpression(LOWEST)
	return expression
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseMapLiteral() ast.Expression {
	mapLiteral := &ast.MapLiteral{Token: p.curToken, Pairs: []ast.MapPair{}}

	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return mapLiteral
	}

	for {
		p.nextToken()
		key := p.parseExpression(LOWEST)
		if key == nil {
			return nil
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}

		mapLiteral.Pairs = append(mapLiteral.Pairs, ast.MapPair{Key: key, Value: value})

		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return mapLiteral
}

func (p *Parser) parseMemberExpression(object ast.Expression) ast.Expression {
	expression := &ast.MemberExpression{Token: p.curToken, Object: object}

	p.nextToken()

	// Allow keywords as property names (e.g., http.get, http.post)
	// This includes GET, POST, PUT, DELETE, etc.
	expression.Property = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return expression
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	expression := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	expression.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return expression
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
