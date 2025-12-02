package compiler

import "flowa/pkg/ast"

// foldConstants performs constant folding optimization on InfixExpressions.
// It evaluates constant expressions at compile time and returns a simplified AST node.
// Returns nil if the expression cannot be folded.
func (c *Compiler) foldConstants(node *ast.InfixExpression) ast.Node {
	// First, try to fold the operands recursively
	left := node.Left
	right := node.Right

	// If left is an infix expression, try to fold it
	if leftInfix, ok := left.(*ast.InfixExpression); ok {
		if folded := c.foldConstants(leftInfix); folded != nil {
			// Type assertion to ast.Expression
			if expr, ok := folded.(ast.Expression); ok {
				left = expr
			}
		}
	}

	// If right is an infix expression, try to fold it
	if rightInfix, ok := right.(*ast.InfixExpression); ok {
		if folded := c.foldConstants(rightInfix); folded != nil {
			// Type assertion to ast.Expression
			if expr, ok := folded.(ast.Expression); ok {
				right = expr
			}
		}
	}

	// Check if both operands are integer literals
	leftInt, leftOk := left.(*ast.IntegerLiteral)
	rightInt, rightOk := right.(*ast.IntegerLiteral)

	if leftOk && rightOk {
		// Fold integer arithmetic operations
		var result int64
		switch node.Operator {
		case "+":
			result = leftInt.Value + rightInt.Value
		case "-":
			result = leftInt.Value - rightInt.Value
		case "*":
			result = leftInt.Value * rightInt.Value
		case "/":
			if rightInt.Value == 0 {
				return nil // Cannot fold division by zero
			}
			result = leftInt.Value / rightInt.Value
		case ">":
			// For comparisons, return a boolean
			return &ast.Boolean{Value: leftInt.Value > rightInt.Value}
		case "<":
			return &ast.Boolean{Value: leftInt.Value < rightInt.Value}
		case "==":
			return &ast.Boolean{Value: leftInt.Value == rightInt.Value}
		case "!=":
			return &ast.Boolean{Value: leftInt.Value != rightInt.Value}
		default:
			return nil
		}
		return &ast.IntegerLiteral{Value: result}
	}

	// Check if both operands are boolean literals
	leftBool, leftOk := left.(*ast.Boolean)
	rightBool, rightOk := right.(*ast.Boolean)

	if leftOk && rightOk {
		// Fold boolean operations
		switch node.Operator {
		case "==":
			return &ast.Boolean{Value: leftBool.Value == rightBool.Value}
		case "!=":
			return &ast.Boolean{Value: leftBool.Value != rightBool.Value}
		}
	}

	return nil // Cannot fold
}
