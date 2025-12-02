package compiler

import (
	"flowa/pkg/ast"
	"flowa/pkg/eval"
	"flowa/pkg/opcode"
	"fmt"
)

type CompilerScope struct {
	instructions        opcode.Instructions
	lastInstruction     opcode.Opcode
	previousInstruction opcode.Opcode
}

type Compiler struct {
	instructions opcode.Instructions
	constants    []eval.Object
	symbolTable  *SymbolTable
	scopes       []CompilerScope
	scopeIndex   int
}

type Bytecode struct {
	Instructions  opcode.Instructions
	Constants     []eval.Object
	MainNumLocals int
}

type SymbolTable struct {
	Outer          *SymbolTable
	store          map[string]Symbol
	numDefinitions int
}

type Symbol struct {
	Name  string
	Scope string
	Index int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store: make(map[string]Symbol),
	}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions}
	if s.Outer == nil {
		symbol.Scope = "GLOBAL"
	} else {
		symbol.Scope = "LOCAL"
	}
	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		return obj, ok
	}
	return obj, ok
}

func New() *Compiler {
	mainScope := CompilerScope{
		instructions:        opcode.Instructions{},
		lastInstruction:     opcode.Opcode(0), // No instruction yet
		previousInstruction: opcode.Opcode(0),
	}

	// Create a symbol table with an outer scope so that variables
	// in the main program are treated as "LOCAL" instead of "GLOBAL".
	// This improves performance dramatically by using stack-based access
	// instead of hash map lookups for globals.
	mainSymbolTable := NewEnclosedSymbolTable(NewSymbolTable())

	return &Compiler{
		instructions: mainScope.instructions,
		constants:    []eval.Object{},
		symbolTable:  mainSymbolTable,
		scopes:       []CompilerScope{mainScope},
		scopeIndex:   0,
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.FunctionStatement:
		c.enterScope()

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		// Implicit return null if not present?
		if !c.lastInstructionIs(opcode.OpReturnValue) && !c.lastInstructionIs(opcode.OpReturn) {
			c.emit(opcode.OpReturn)
		}

		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope()

		compiledFn := &eval.Function{
			Instructions: instructions,
			SlotCount:    numLocals,
		}
		c.emit(opcode.OpConstant, c.addConstant(compiledFn))

		symbol := c.symbolTable.Define(node.Name.Value)
		if symbol.Scope == "GLOBAL" {
			c.emit(opcode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(opcode.OpSetLocal, symbol.Index)
		}

	case *ast.ReturnStatement:
		if node.ReturnValue != nil {
			err := c.Compile(node.ReturnValue)
			if err != nil {
				return err
			}
			c.emit(opcode.OpReturnValue)
		} else {
			c.emit(opcode.OpReturn)
		}

	case *ast.AssignmentStatement:
		// Optimization: Check for i = i + 1 pattern
		if infix, ok := node.Value.(*ast.InfixExpression); ok && infix.Operator == "+" {
			// Check if it's a local variable
			symbol, ok := c.symbolTable.Resolve(node.Name.Value)
			if ok && symbol.Scope == "LOCAL" {
				// Check for i + 1
				if leftIdent, ok := infix.Left.(*ast.Identifier); ok && leftIdent.Value == node.Name.Value {
					if rightInt, ok := infix.Right.(*ast.IntegerLiteral); ok && rightInt.Value == 1 {
						c.emit(opcode.OpIncLocal, symbol.Index)
						return nil
					}
				}
				// Check for 1 + i
				if rightIdent, ok := infix.Right.(*ast.Identifier); ok && rightIdent.Value == node.Name.Value {
					if leftInt, ok := infix.Left.(*ast.IntegerLiteral); ok && leftInt.Value == 1 {
						c.emit(opcode.OpIncLocal, symbol.Index)
						return nil
					}
				}

				// Optimization: Check for sum = sum + i (OpAddLocal)
				// Case 1: sum = sum + i
				if leftIdent, ok := infix.Left.(*ast.Identifier); ok && leftIdent.Value == node.Name.Value {
					if rightIdent, ok := infix.Right.(*ast.Identifier); ok {
						if sourceSymbol, ok := c.symbolTable.Resolve(rightIdent.Value); ok && sourceSymbol.Scope == "LOCAL" {
							c.emit(opcode.OpAddLocal, symbol.Index, sourceSymbol.Index)
							return nil
						}
					}
				}
				// Case 2: sum = i + sum
				if rightIdent, ok := infix.Right.(*ast.Identifier); ok && rightIdent.Value == node.Name.Value {
					if leftIdent, ok := infix.Left.(*ast.Identifier); ok {
						if sourceSymbol, ok := c.symbolTable.Resolve(leftIdent.Value); ok && sourceSymbol.Scope == "LOCAL" {
							c.emit(opcode.OpAddLocal, symbol.Index, sourceSymbol.Index)
							return nil
						}
					}
				}
			}
		}

		// Compile the value expression first
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}

		// Resolve the symbol first to check if it's already defined
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			symbol = c.symbolTable.Define(node.Name.Value)
		}
		if symbol.Scope == "GLOBAL" {
			c.emit(opcode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(opcode.OpSetLocal, symbol.Index)
		}

	case *ast.WhileStatement:
		// While loop structure:
		// <condition>
		// OpJumpNotTruth <end>
		// <body>
		// OpJump <condition>
		// <end>

		loopStart := len(c.instructions)

		// Optimization: Check for while i < N (OpJumpIfLocalGreaterEqualConst)
		var optimizedJumpPos int = -1

		if infix, ok := node.Condition.(*ast.InfixExpression); ok && infix.Operator == "<" {
			// Case: i < 10000000
			if leftIdent, ok := infix.Left.(*ast.Identifier); ok {
				if rightInt, ok := infix.Right.(*ast.IntegerLiteral); ok {
					if symbol, ok := c.symbolTable.Resolve(leftIdent.Value); ok && symbol.Scope == "LOCAL" {
						// Add constant
						constObj := eval.NewInteger(rightInt.Value)
						constIdx := c.addConstant(constObj)

						// Emit optimized jump: if i >= 10000000 goto end
						optimizedJumpPos = c.emit(opcode.OpJumpIfLocalGreaterEqualConst, symbol.Index, constIdx, 9999)
					}
				}
			}
		}

		var jumpNotTruthPos int
		if optimizedJumpPos != -1 {
			jumpNotTruthPos = optimizedJumpPos
		} else {
			// Compile condition normally
			err := c.Compile(node.Condition)
			if err != nil {
				return err
			}

			// Emit conditional jump (placeholder)
			jumpNotTruthPos = c.emit(opcode.OpJumpNotTruth, 9999)
		}

		// Compile body
		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		// Jump back to condition
		c.emit(opcode.OpJump, loopStart)

		// Backpatch the conditional jump to point here (after the loop)
		afterLoopPos := len(c.instructions)

		// For OpJumpIfLocalGreaterEqualConst, we need to update only the last 2 bytes (jump position)
		// without touching the first 3 bytes (local index + const index)
		if optimizedJumpPos != -1 && c.instructions[optimizedJumpPos] == byte(opcode.OpJumpIfLocalGreaterEqualConst) {
			// Manual backpatch: update bytes at opPos+4 and opPos+5 (the jump position)
			c.instructions[optimizedJumpPos+4] = byte(afterLoopPos >> 8)
			c.instructions[optimizedJumpPos+5] = byte(afterLoopPos)
		} else {
			c.changeOperand(jumpNotTruthPos, afterLoopPos)
		}

	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(opcode.OpPop)

	case *ast.InfixExpression:
		// Constant folding optimization
		if folded := c.foldConstants(node); folded != nil {
			return c.Compile(folded)
		}

		// Special case for < operator: compile right then left, then use >
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			c.emit(opcode.OpGreaterThan)
			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(opcode.OpAdd)
		case "-":
			c.emit(opcode.OpSub)
		case "*":
			c.emit(opcode.OpMul)
		case "/":
			c.emit(opcode.OpDiv)
		case ">":
			c.emit(opcode.OpGreaterThan)
		case "==":
			c.emit(opcode.OpEqual)
		case "!=":
			c.emit(opcode.OpNotEqual)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IntegerLiteral:
		integer := eval.NewInteger(node.Value)
		c.emit(opcode.OpConstant, c.addConstant(integer))

	case *ast.StringLiteral:
		str := &eval.String{Value: node.Value}
		c.emit(opcode.OpConstant, c.addConstant(str))

	case *ast.Identifier:
		// Try to resolve as a variable first
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if ok {
			if symbol.Scope == "GLOBAL" {
				c.emit(opcode.OpGetGlobal, symbol.Index)
			} else {
				c.emit(opcode.OpGetLocal, symbol.Index)
			}
			return nil
		}

		// Handle built-in boolean and null identifiers
		switch node.Value {
		case "true":
			c.emit(opcode.OpTrue)
		case "false":
			c.emit(opcode.OpFalse)
		case "nil":
			c.emit(opcode.OpNull)
		case "print":
			c.emit(opcode.OpGetBuiltin, 0) // 0 = print
		case "len":
			c.emit(opcode.OpGetBuiltin, 1) // 1 = len
		case "time":
			c.emit(opcode.OpGetBuiltin, 2) // 2 = time module
		default:
			return fmt.Errorf("undefined identifier: %s", node.Value)
		}

	case *ast.MemberExpression:
		err := c.Compile(node.Object)
		if err != nil {
			return err
		}

		// Property is treated as a string index
		propName := &eval.String{Value: node.Property.Value}
		c.emit(opcode.OpConstant, c.addConstant(propName))
		c.emit(opcode.OpIndex)

	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}

		c.emit(opcode.OpCall, len(node.Arguments))

	case *ast.Boolean:
		if node.Value {
			c.emit(opcode.OpTrue)
		} else {
			c.emit(opcode.OpFalse)
		}
	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions:  c.instructions,
		Constants:     c.constants,
		MainNumLocals: c.symbolTable.numDefinitions,
	}
}

func (c *Compiler) addConstant(obj eval.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op opcode.Opcode, operands ...int) int {
	ins := opcode.Make(op, operands...)
	pos := c.addInstruction(ins)
	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.instructions)
	c.instructions = append(c.instructions, ins...)
	return posNewInstruction
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := opcode.Opcode(c.instructions[opPos])
	newInstruction := opcode.Make(op, operand)

	for i := 0; i < len(newInstruction); i++ {
		c.instructions[opPos+i] = newInstruction[i]
	}
}

func (c *Compiler) enterScope() {
	scope := CompilerScope{
		instructions:        opcode.Instructions{},
		lastInstruction:     opcode.Opcode(0),
		previousInstruction: opcode.Opcode(0),
	}
	// Save current instructions to current scope
	c.scopes[c.scopeIndex].instructions = c.instructions

	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
	c.instructions = scope.instructions
}

func (c *Compiler) leaveScope() opcode.Instructions {
	instructions := c.instructions

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer
	c.instructions = c.scopes[c.scopeIndex].instructions

	return instructions
}

func (c *Compiler) lastInstructionIs(op opcode.Opcode) bool {
	if len(c.instructions) == 0 {
		return false
	}
	return opcode.Opcode(c.instructions[len(c.instructions)-1]) == op
}
