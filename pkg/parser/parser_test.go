package parser

import (
	"flowa/pkg/ast"
	"flowa/pkg/lexer"
	"testing"
)

func TestReturnStatements(t *testing.T) {
	input := `
return 5
return 10
return 993322
`
	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d",
			len(program.Statements))
	}

	for _, stmt := range program.Statements {
		returnStmt, ok := stmt.(*ast.ReturnStatement)
		if !ok {
			t.Errorf("stmt not *ast.ReturnStatement. got=%T", stmt)
			continue
		}
		if returnStmt.TokenLiteral() != "return" {
			t.Errorf("returnStmt.TokenLiteral not 'return', got %q",
				returnStmt.TokenLiteral())
		}
	}
}

func TestFunctionStatement(t *testing.T) {
	input := `func add(x, y){
return x + y
}
`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statements. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.FunctionStatement. got=%T",
			program.Statements[0])
	}

	if stmt.Name.Value != "add" {
		t.Fatalf("function name not 'add'. got=%q", stmt.Name.Value)
	}

	if len(stmt.Parameters) != 2 {
		t.Fatalf("function literal has wrong parameters count. got=%d",
			len(stmt.Parameters))
	}

	if stmt.Parameters[0].Value != "x" {
		t.Fatalf("parameter 0 is not 'x'. got=%q", stmt.Parameters[0].Value)
	}

	if stmt.Parameters[1].Value != "y" {
		t.Fatalf("parameter 1 is not 'y'. got=%q", stmt.Parameters[1].Value)
	}

	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("function body has wrong statements count. got=%d",
			len(stmt.Body.Statements))
	}

	returnStmt, ok := stmt.Body.Statements[0].(*ast.ReturnStatement)
	if !ok {
		t.Fatalf("function body stmt is not ast.ReturnStatement. got=%T",
			stmt.Body.Statements[0])
	}

	// Check return value expression structure
	infix, ok := returnStmt.ReturnValue.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("return value is not ast.InfixExpression. got=%T", returnStmt.ReturnValue)
	}
	if infix.Operator != "+" {
		t.Fatalf("operator is not '+'. got=%q", infix.Operator)
	}
}

func TestPipelineExpression(t *testing.T) {
	input := `data |> map(f)`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statements. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	pipe, ok := stmt.Expression.(*ast.PipelineExpression)
	if !ok {
		t.Fatalf("exp not *ast.PipelineExpression. got=%T", stmt.Expression)
	}

	if pipe.Left.String() != "data" {
		t.Errorf("pipe.Left not 'data'. got=%q", pipe.Left.String())
	}

	call, ok := pipe.Right.(*ast.CallExpression)
	if !ok {
		t.Fatalf("pipe.Right not *ast.CallExpression. got=%T", pipe.Right)
	}

	if call.Function.String() != "map" {
		t.Errorf("call.Function not 'map'. got=%q", call.Function.String())
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}
