package compiler

import (
	"flowa/pkg/eval"
	"sync"
)

// Compiler pool for reusing compiler instances to avoid allocations
var compilerPool = sync.Pool{
	New: func() interface{} {
		return NewCompilerForPool()
	},
}

// NewCompilerForPool creates a new compiler with pre-allocated structures
// for use with the compiler pool
func NewCompilerForPool() *Compiler {
	mainScope := CompilerScope{
		instructions:        []byte{}, // Don't pre-allocate - causes test issues
		lastInstruction:     0,
		previousInstruction: 0,
	}

	// Create a symbol table with an outer scope
	mainSymbolTable := NewEnclosedSymbolTable(NewSymbolTable())

	return &Compiler{
		instructions: mainScope.instructions,
		constants:    []eval.Object{}, // Don't pre-allocate
		symbolTable:  mainSymbolTable,
		scopes:       []CompilerScope{mainScope},
		scopeIndex:   0,
	}
}

// GetCompiler retrieves a compiler from the pool
func GetCompiler() *Compiler {
	return compilerPool.Get().(*Compiler)
}

// PutCompiler returns a compiler to the pool after use
func PutCompiler(c *Compiler) {
	// Reset compiler state
	c.instructions = c.instructions[:0] // Keep capacity, reset length
	c.constants = c.constants[:0]       // Keep capacity, reset length
	c.scopes = c.scopes[:1]             // Keep main scope only
	c.scopeIndex = 0

	// Reset main scope
	c.scopes[0].instructions = c.scopes[0].instructions[:0]
	c.scopes[0].lastInstruction = 0
	c.scopes[0].previousInstruction = 0

	// Create fresh symbol table
	c.symbolTable = NewEnclosedSymbolTable(NewSymbolTable())

	compilerPool.Put(c)
}
