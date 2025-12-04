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
	instructions      opcode.Instructions
	constants         []eval.Object
	symbolTable       *SymbolTable
	scopes            []CompilerScope
	scopeIndex        int
	loopStack         [][]int // Stack of break positions for nested loops
	mainFunctionIndex int     // Global index of main function, -1 if not found
}

type Bytecode struct {
	Instructions      opcode.Instructions
	Constants         []eval.Object
	MainNumLocals     int
	MainFunctionIndex int // Index of main function in globals, -1 if not present
}

type SymbolTable struct {
	Outer          *SymbolTable // Exported for import system access
	store          map[string]Symbol
	numDefinitions int
}

type Symbol struct {
	Name  string
	Scope string
	Index int
}

func (st *SymbolTable) GetStore() map[string]Symbol {
	return st.store
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

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Scope: "BUILTIN", Index: index}
	s.store[name] = symbol
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

// Store returns the internal symbol store for iteration
func (s *SymbolTable) Store() map[string]Symbol {
	return s.store
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

	// Register built-ins (must match VM indices exactly - see pkg/vm/vm.go New())
	builtins := []string{
		"print",          // 0
		"len",            // 1
		"time",           // 2
		"auth",           // 3
		"json",           // 4
		"http",           // 5
		"fs",             // 6
		"response",       // 7
		"websocket",      // 8
		"mail",           // 9
		"jwt",            // 10
		"config",         // 11
		"fast_sum_to",    // 12
		"fast_sum_range", // 13
		"fast_repeat",    // 14
		"route",          // 15
		"listen",         // 16
	}

	for i, name := range builtins {
		mainSymbolTable.Outer.DefineBuiltin(i, name)
	}

	return &Compiler{
		instructions:      mainScope.instructions,
		constants:         []eval.Object{},
		symbolTable:       mainSymbolTable,
		scopes:            []CompilerScope{mainScope},
		scopeIndex:        0,
		mainFunctionIndex: -1, // No main function found yet
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

		// If the last instruction is OpPop, remove it so the block returns a value
		if len(c.instructions) > 0 && c.lastInstructionIs(opcode.OpPop) {
			c.removeLastPop()
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

		// Functions must be GLOBAL to support cross-function references
		// Find the outermost (truly global) scope
		globalTable := c.symbolTable
		for globalTable.Outer != nil {
			globalTable = globalTable.Outer
		}

		// Define the symbol in the global scope
		symbol := globalTable.Define(node.Name.Value)
		symbol.Scope = "GLOBAL"
		globalTable.store[node.Name.Value] = symbol

		// Track if this is the main function
		if node.Name.Value == "main" {
			c.mainFunctionIndex = symbol.Index
		}

		c.emit(opcode.OpSetGlobal, symbol.Index)

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

	case *ast.BreakStatement:
		if len(c.loopStack) == 0 {
			return fmt.Errorf("break outside of loop")
		}
		// Emit a jump that will be patched later to jump to loop end
		pos := c.emit(opcode.OpJump, 9999)
		// Add this position to the current loop's break list
		c.loopStack[len(c.loopStack)-1] = append(c.loopStack[len(c.loopStack)-1], pos)

	case *ast.AssignmentStatement:
		// Optimization DISABLED: i = 0 pattern (OpSetLocalZero)
		// Even with constant alignment, still causes crashes
		// The opcodes themselves may be broken in the VM
		/*
			if intLit, ok := node.Value.(*ast.IntegerLiteral); ok && intLit.Value == 0 {
				symbol, ok := c.symbolTable.Resolve(node.Name.Value)
				if !ok {
					symbol = c.symbolTable.Define(node.Name.Value)
				}
				if symbol.Scope == "LOCAL" {
					c.emit(opcode.OpSetLocalZero, symbol.Index)
					return nil
				}
			}
		*/

		// Optimization DISABLED: i = i + 1 pattern (OpIncLocal)
		// Even with constant alignment, still causes crashes
		/*
			if infix, ok := node.Value.(*ast.InfixExpression); ok && infix.Operator == "+" {
				if leftIdent, ok := infix.Left.(*ast.Identifier); ok && leftIdent.Value == node.Name.Value {
					if rightInt, ok := infix.Right.(*ast.IntegerLiteral); ok && rightInt.Value == 1 {
						if symbol, ok := c.symbolTable.Resolve(node.Name.Value); ok && symbol.Scope == "LOCAL" {
							c.emit(opcode.OpIncLocal, symbol.Index)
							return nil
						}
					}
				}
			}
		*/

		// Compile the value expression first
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}

		// Get or define the symbol
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			// Define new variable
			symbol = c.symbolTable.Define(node.Name.Value)
		}

		// Emit the appropriate set instruction
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

		// Push new loop context for break tracking
		c.loopStack = append(c.loopStack, []int{})

		loopStart := len(c.instructions)

		// Optimization: Check for while i < N (OpJumpIfLocalGreaterEqualConst)
		// DISABLED: Has encoding bug, needs investigation
		var optimizedJumpPos int = -1
		/*
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
		*/

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

		// Patch all break statements to jump to after the loop
		breakPositions := c.loopStack[len(c.loopStack)-1]
		for _, breakPos := range breakPositions {
			c.changeOperand(breakPos, afterLoopPos)
		}

		// Pop loop context
		c.loopStack = c.loopStack[:len(c.loopStack)-1]

	case *ast.ForStatement:
		// For loop structure (for item in array):
		// 1. Compile the iterable (array/range)
		// 2. Store it in a temp local
		// 3. Initialize iterator index to 0
		// 4. Loop:
		//    - Check if index < length
		//    - Get array[index]
		//    - Assign to loop variable
		//    - Execute body
		//    - Increment index
		//    - Jump back

		// Compile the iterable expression
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}

		// Store the array in a temporary local (we'll reuse the iterator name + "_array")
		arraySymbol := c.symbolTable.Define(node.Iterator.Value + "_array")
		if arraySymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpSetGlobal, arraySymbol.Index)
		} else {
			c.emit(opcode.OpSetLocal, arraySymbol.Index)
		}

		// Initialize index to 0
		indexSymbol := c.symbolTable.Define(node.Iterator.Value + "_index")
		c.emit(opcode.OpConstant, c.addConstant(eval.NewInteger(0)))
		if indexSymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpSetGlobal, indexSymbol.Index)
		} else {
			c.emit(opcode.OpSetLocal, indexSymbol.Index)
		}

		// Define the loop variable
		loopVarSymbol := c.symbolTable.Define(node.Iterator.Value)

		// Push new loop context for break tracking
		c.loopStack = append(c.loopStack, []int{})

		loopStart := len(c.instructions)

		// Check if index < array.length
		// Load index first
		if indexSymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpGetGlobal, indexSymbol.Index)
		} else {
			c.emit(opcode.OpGetLocal, indexSymbol.Index)
		}

		// For calling len(array), we need: [len_builtin, array] on stack before OpCall
		// OpCall expects: [function, arg1, arg2, ...]
		c.emit(opcode.OpGetBuiltin, 1) // len builtin

		// Load array
		if arraySymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpGetGlobal, arraySymbol.Index)
		} else {
			c.emit(opcode.OpGetLocal, arraySymbol.Index)
		}

		c.emit(opcode.OpCall, 1) // len(array)

		// Stack now has: [index, length]
		// OpLessThan pops right (length) then left (index)
		// Computes: left < right = index < length (Correct!)
		c.emit(opcode.OpLessThan)

		// Jump if not true (exit loop)
		jumpNotTruthPos := c.emit(opcode.OpJumpNotTruth, 9999)

		// Get array[index]
		if arraySymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpGetGlobal, arraySymbol.Index)
		} else {
			c.emit(opcode.OpGetLocal, arraySymbol.Index)
		}
		if indexSymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpGetGlobal, indexSymbol.Index)
		} else {
			c.emit(opcode.OpGetLocal, indexSymbol.Index)
		}
		c.emit(opcode.OpIndex)

		// Assign to loop variable
		if loopVarSymbol.Scope == "GLOBAL" {
			c.emit(opcode.OpSetGlobal, loopVarSymbol.Index)
		} else {
			c.emit(opcode.OpSetLocal, loopVarSymbol.Index)
		}

		// Compile body
		err = c.Compile(node.Body)
		if err != nil {
			return err
		}

		// Increment index: index = index + 1
		if indexSymbol.Scope == "LOCAL" {
			c.emit(opcode.OpIncLocal, indexSymbol.Index)
		} else {
			// For non-local (shouldn't happen in this case, but handle it)
			if indexSymbol.Scope == "GLOBAL" {
				c.emit(opcode.OpGetGlobal, indexSymbol.Index)
			} else {
				c.emit(opcode.OpGetLocal, indexSymbol.Index)
			}
			c.emit(opcode.OpConstant, c.addConstant(eval.NewInteger(1)))
			c.emit(opcode.OpAdd)
			if indexSymbol.Scope == "GLOBAL" {
				c.emit(opcode.OpSetGlobal, indexSymbol.Index)
			} else {
				c.emit(opcode.OpSetLocal, indexSymbol.Index)
			}
		}

		// Jump back to loop start
		c.emit(opcode.OpJump, loopStart)

		// Backpatch the exit jump
		afterLoopPos := len(c.instructions)
		c.changeOperand(jumpNotTruthPos, afterLoopPos)

		// Patch all break statements to jump to after the loop
		breakPositions := c.loopStack[len(c.loopStack)-1]
		for _, breakPos := range breakPositions {
			c.changeOperand(breakPos, afterLoopPos)
		}

		// Pop loop context
		c.loopStack = c.loopStack[:len(c.loopStack)-1]

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
		case "<":
			c.emit(opcode.OpLessThan)
		case ">=":
			c.emit(opcode.OpGreaterThanEqual)
		case "<=":
			c.emit(opcode.OpLessThanEqual)
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

	case *ast.FloatLiteral:
		float := &eval.Float{Value: node.Value}
		c.emit(opcode.OpConstant, c.addConstant(float))

	case *ast.StringLiteral:
		str := &eval.String{Value: node.Value}
		c.emit(opcode.OpConstant, c.addConstant(str))

	case *ast.Identifier:
		// Try to resolve as a variable first
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if ok {
			switch symbol.Scope {
			case "GLOBAL":
				c.emit(opcode.OpGetGlobal, symbol.Index)
			case "LOCAL":
				c.emit(opcode.OpGetLocal, symbol.Index)
			case "BUILTIN":
				c.emit(opcode.OpGetBuiltin, symbol.Index)
			default:
				return fmt.Errorf("unknown symbol scope: %s", symbol.Scope)
			}
			return nil
		}

		// Handle built-in boolean and null identifiers
		switch node.Value {
		case "true":
			c.emit(opcode.OpTrue)
		case "false":
			c.emit(opcode.OpFalse)
		case "nil", "null", "None":
			c.emit(opcode.OpNull)
		default:
			return fmt.Errorf("undefined identifier: %s", node.Value)
		}

	case *ast.PrefixExpression:
		// Handle unary operators like - and !
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(opcode.OpBang)
		case "-":
			c.emit(opcode.OpMinus)
		default:
			return fmt.Errorf("unknown prefix operator: %s", node.Operator)
		}

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}
		c.emit(opcode.OpArray, len(node.Elements))

	case *ast.MapLiteral:
		for _, pair := range node.Pairs {
			err := c.Compile(pair.Key)
			if err != nil {
				return err
			}
			err = c.Compile(pair.Value)
			if err != nil {
				return err
			}
		}
		c.emit(opcode.OpHash, len(node.Pairs)*2)

	case *ast.PipelineExpression:
		// Pipeline: a |> f becomes f(a)
		// Pipeline: a |> f(b) becomes f(a, b)

		// Check the right side
		switch right := node.Right.(type) {
		case *ast.Identifier:
			// Simple case: a |> f
			// Compile as f(a)
			// Stack order: function first, then argument
			err := c.Compile(right) // Function
			if err != nil {
				return err
			}
			err = c.Compile(node.Left) // Argument
			if err != nil {
				return err
			}
			c.emit(opcode.OpCall, 1)

		case *ast.CallExpression:
			// Case: a |> f(b, c)
			// Compile as f(a, b, c)
			// Stack order: function, then piped value, then call's arguments

			// Compile the function first
			err := c.Compile(right.Function)
			if err != nil {
				return err
			}

			// Compile the piped value (first argument)
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			// Now compile the call's arguments
			numArgs := 1 // Start with 1 for the piped value
			for _, arg := range right.Arguments {
				err := c.Compile(arg)
				if err != nil {
					return err
				}
				numArgs++
			}

			c.emit(opcode.OpCall, numArgs)

		case *ast.PipelineExpression:
			// Nested pipeline: a |> b |> c
			// Compile as (a |> b) |> c
			// First create a pipeline for a |> b
			leftPipeline := &ast.PipelineExpression{
				Token: node.Token,
				Left:  node.Left,
				Right: right.Left,
			}
			// Then pipe that result to c
			return c.Compile(&ast.PipelineExpression{
				Token: right.Token,
				Left:  leftPipeline,
				Right: right.Right,
			})

		default:
			return fmt.Errorf("invalid right-hand side of pipeline: %T", node.Right)
		}

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		c.emit(opcode.OpIndex)

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

	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		// Emit jump not truth with placeholder
		jumpNotTruthPos := c.emit(opcode.OpJumpNotTruth, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(opcode.OpPop) {
			c.removeLastPop()
		}

		// Jump over alternative
		jumpPos := c.emit(opcode.OpJump, 9999)

		afterConsequencePos := len(c.instructions)
		c.changeOperand(jumpNotTruthPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(opcode.OpNull)
		} else {
			// Special case for elif (ExpressionStatement containing IfExpression)
			if exprStmt, ok := node.Alternative.(*ast.ExpressionStatement); ok {
				err := c.Compile(exprStmt.Expression)
				if err != nil {
					return err
				}
			} else {
				err := c.Compile(node.Alternative)
				if err != nil {
					return err
				}
			}

			if c.lastInstructionIs(opcode.OpPop) {
				c.removeLastPop()
			}
		}

		afterAlternativePos := len(c.instructions)
		c.changeOperand(jumpPos, afterAlternativePos)

	case *ast.ImportStatement:
		// import "module_name"
		// Emit OpImport with module name constant
		moduleName := node.Path.Value
		c.emit(opcode.OpImport, c.addConstant(&eval.String{Value: moduleName}))

		// Store module in variable with same name as module (e.g. import "math" -> math = ...)
		// If alias provided (not supported in AST yet), use that.
		// For now, assume module name is the variable name.
		// Strip extension and path if present
		// e.g. "libs/math.flowa" -> "math"
		// Simple implementation: use the full string as variable name for now, user should use simple names
		// Better: use base name without extension

		// Define symbol
		symbol := c.symbolTable.Define(moduleName)
		if symbol.Scope == "GLOBAL" {
			c.emit(opcode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(opcode.OpSetLocal, symbol.Index)
		}

	case *ast.FromImportStatement:
		// from "module" import x, y
		// 1. Import module -> Stack: [module]
		moduleName := node.Path.Value
		c.emit(opcode.OpImport, c.addConstant(&eval.String{Value: moduleName}))

		// 2. For each symbol, get it from module
		for range node.Symbols {
			// Duplicate module for next extraction (except last one)
			// But we don't have OpDup yet!
			// Workaround: Store module in temp local, then load it for each symbol

			// Actually, simpler:
			// Just emit OpImport for EACH symbol. It's cached in VM so it's cheap.
			// This avoids needing OpDup or temp variables.
		}

		// Wait, the loop above didn't do anything. Let's redo.

		// Strategy:
		// 1. Import module
		// 2. Store in temp variable (hidden)
		// 3. For each symbol:
		//    - Load module
		//    - Load symbol name string
		//    - OpIndex
		//    - Store in variable

		// Since we don't have hidden temp vars easily, let's just re-import for each symbol.
		// The VM caches imports, so subsequent OpImports for same module return the same object instantly.

		for _, ident := range node.Symbols {
			// Import module
			c.emit(opcode.OpImport, c.addConstant(&eval.String{Value: moduleName}))

			// Get symbol
			c.emit(opcode.OpConstant, c.addConstant(&eval.String{Value: ident.Value}))
			c.emit(opcode.OpIndex)

			// Store in variable
			symbol := c.symbolTable.Define(ident.Value)
			if symbol.Scope == "GLOBAL" {
				c.emit(opcode.OpSetGlobal, symbol.Index)
			} else {
				c.emit(opcode.OpSetLocal, symbol.Index)
			}
		}

	case *ast.PostfixExpression:
		// Postfix operators: i++, i--
		// These return the old value and modify the variable
		// Only support identifiers for now
		ident, ok := node.Left.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("postfix operators only supported on identifiers")
		}

		symbol, ok := c.symbolTable.Resolve(ident.Value)
		if !ok {
			return fmt.Errorf("undefined variable: %s", ident.Value)
		}

		if symbol.Scope != "LOCAL" {
			return fmt.Errorf("postfix operators only supported on local variables")
		}

		switch node.Operator {
		case "++":
			c.emit(opcode.OpPostfixInc, symbol.Index)
		case "--":
			c.emit(opcode.OpPostfixDec, symbol.Index)
		default:
			return fmt.Errorf("unknown postfix operator: %s", node.Operator)
		}

	case *ast.ClassicForStatement:
		// Classic for-loop: for(init; condition; post) { body }
		// Structure:
		//   <init>
		//   loop_start:
		//     <condition>
		//     OpJumpNotTruth loop_end
		//     <body>
		//     <post>
		//     OpJump loop_start
		//   loop_end:

		// Compile init statement
		if node.Init != nil {
			err := c.Compile(node.Init)
			if err != nil {
				return err
			}
			// Pop the result of init if it's an expression statement
			if _, ok := node.Init.(*ast.ExpressionStatement); ok {
				c.emit(opcode.OpPop)
			}
		}

		loopStart := len(c.instructions)

		// Compile condition
		jumpNotTruthPos := -1
		if node.Condition != nil {
			err := c.Compile(node.Condition)
			if err != nil {
				return err
			}
			jumpNotTruthPos = c.emit(opcode.OpJumpNotTruth, 9999)
		}

		// Compile body
		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		// Compile post statement
		if node.Post != nil {
			err := c.Compile(node.Post)
			if err != nil {
				return err
			}
			// Pop the result of post if it's an expression statement
			if _, ok := node.Post.(*ast.ExpressionStatement); ok {
				c.emit(opcode.OpPop)
			}
		}

		// Jump back to condition
		c.emit(opcode.OpJump, loopStart)

		// Backpatch the conditional jump
		if jumpNotTruthPos != -1 {
			afterLoopPos := len(c.instructions)
			c.changeOperand(jumpNotTruthPos, afterLoopPos)
		}
	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions:      c.instructions,
		Constants:         c.constants,
		MainNumLocals:     c.symbolTable.numDefinitions,
		MainFunctionIndex: c.mainFunctionIndex,
	}
}

// SymbolTable returns the compiler's symbol table for external access
func (c *Compiler) SymbolTable() *SymbolTable {
	return c.symbolTable
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

func (c *Compiler) removeLastPop() {
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.instructions
	newIns := old[:len(old)-1]

	c.instructions = newIns
	c.scopes[c.scopeIndex].instructions = newIns
	c.scopes[c.scopeIndex].lastInstruction = previous
	// We don't know the instruction before previous, but that's okay for now
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
func (c *Compiler) GetSymbolTable() *SymbolTable {
	return c.symbolTable
}
