package main

import (
	"flowa/pkg/ast"
	"fmt"
	"strings"
)

type ProgramInsights struct {
	Functions []FunctionInfo
	Pipelines []PipelineInfo
}

type FunctionInfo struct {
	Name       string
	Parameters []string
	IsAsync    bool
}

type PipelineInfo struct {
	Chain []string
}

func analyzeProgram(program *ast.Program) ProgramInsights {
	insights := ProgramInsights{}
	walk(program, func(node ast.Node) {
		switch n := node.(type) {
		case *ast.FunctionStatement:
			params := make([]string, 0, len(n.Parameters))
			for _, p := range n.Parameters {
				params = append(params, p.Value)
			}
			insights.Functions = append(insights.Functions, FunctionInfo{
				Name:       n.Name.String(),
				Parameters: params,
				IsAsync:    n.IsAsync,
			})
		case *ast.PipelineExpression:
			if _, nested := n.Left.(*ast.PipelineExpression); nested {
				return
			}
			insights.Pipelines = append(insights.Pipelines, PipelineInfo{
				Chain: flattenPipeline(n),
			})
		}
	})
	return insights
}

func walk(node ast.Node, visitor func(ast.Node)) {
	if node == nil {
		return
	}

	visitor(node)

	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			walk(stmt, visitor)
		}
	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			walk(stmt, visitor)
		}
	case *ast.ExpressionStatement:
		walk(n.Expression, visitor)
	case *ast.ReturnStatement:
		walk(n.ReturnValue, visitor)
	case *ast.FunctionStatement:
		walk(n.Body, visitor)
	case *ast.AssignmentStatement:
		walk(n.Value, visitor)
	case *ast.PrefixExpression:
		walk(n.Right, visitor)
	case *ast.InfixExpression:
		walk(n.Left, visitor)
		walk(n.Right, visitor)
	case *ast.CallExpression:
		walk(n.Function, visitor)
		for _, arg := range n.Arguments {
			walk(arg, visitor)
		}
	case *ast.PipelineExpression:
		walk(n.Left, visitor)
		walk(n.Right, visitor)
	case *ast.IfExpression:
		walk(n.Condition, visitor)
		walk(n.Consequence, visitor)
		if n.Alternative != nil {
			walk(n.Alternative, visitor)
		}
	case *ast.SpawnExpression:
		walk(n.Call, visitor)
	case *ast.AwaitExpression:
		walk(n.Value, visitor)
	}
}

func flattenPipeline(expr ast.Expression) []string {
	switch n := expr.(type) {
	case *ast.PipelineExpression:
		chain := flattenPipeline(n.Left)
		chain = append(chain, describeExpression(n.Right))
		return chain
	default:
		return []string{describeExpression(expr)}
	}
}

func describeExpression(expr ast.Expression) string {
	switch n := expr.(type) {
	case *ast.Identifier:
		return n.Value
	case *ast.IntegerLiteral:
		return n.TokenLiteral()
	case *ast.CallExpression:
		args := make([]string, 0, len(n.Arguments))
		for _, arg := range n.Arguments {
			args = append(args, arg.String())
		}
		return fmt.Sprintf("%s(%s)", n.Function.String(), strings.Join(args, ", "))
	default:
		return expr.String()
	}
}
