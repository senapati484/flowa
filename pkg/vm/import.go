package vm

import (
	"flowa/pkg/compiler"
	"flowa/pkg/eval"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"fmt"
	"os"
	"path/filepath"
)

// loadModule loads and executes a Flowa module file, returning a module object
func (vm *VM) loadModule(path string) (eval.Object, error) {
	// Check cache first
	if cached, ok := vm.importCache[path]; ok {
		return cached, nil
	}

	// Read the module file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read module file: %v", err)
	}

	// Parse the module
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s: %v", path, p.Errors())
	}

	// Compile the module
	comp := compiler.New()
	err = comp.Compile(program)
	if err != nil {
		return nil, fmt.Errorf("compile error in %s: %v", path, err)
	}

	// Create a new VM for the module
	bytecode := comp.Bytecode()
	moduleVM := New(bytecode)

	// Execute the module
	err = moduleVM.Run()
	if err != nil {
		return nil, fmt.Errorf("runtime error in %s: %v", path, err)
	}

	// Create a module object containing all exported symbols
	moduleEnv := eval.NewEnvironment()

	// The compiler uses nested symbol tables
	// Functions are defined in the OUTER (truly global) scope
	// Variables are LOCAL and stored on the stack

	symbolTable := comp.SymbolTable()

	// Helper function to export symbols from a symbol table
	exportFromTable := func(table *compiler.SymbolTable) {
		for name, symbol := range table.Store() {
			// Export GLOBAL scope symbols (skip BUILTIN)
			if symbol.Scope == "GLOBAL" && symbol.Index < len(moduleVM.globals) {
				obj := moduleVM.globals[symbol.Index]
				if obj != nil {
					moduleEnv.Set(name, obj)
				}
			}
			// Export LOCAL scope from stack (module variables)
			if symbol.Scope == "LOCAL" && symbol.Index < moduleVM.sp {
				obj := moduleVM.stack[symbol.Index]
				if obj != nil {
					moduleEnv.Set(name, obj)
				}
			}
		}
	}

	// Export from current scope
	exportFromTable(symbolTable)

	// Export from outer scope (where functions are stored)
	if symbolTable.Outer != nil {
		exportFromTable(symbolTable.Outer)
	}

	moduleObj := &eval.Module{
		Name: filepath.Base(path),
		Env:  moduleEnv,
	}

	// Cache the module
	vm.importCache[path] = moduleObj

	return moduleObj, nil
}
