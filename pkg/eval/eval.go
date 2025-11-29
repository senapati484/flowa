package eval

import (
	"fmt"
	"net/http"
	"strings"

	"flowa/pkg/ast"
)

type Object interface {
	Type() string
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() string    { return "INTEGER" }
func (i *Integer) Inspect() string { return fmt.Sprintf("%d", i.Value) }

type String struct {
	Value string
}

func (s *String) Type() string    { return "STRING" }
func (s *String) Inspect() string { return s.Value }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() string    { return "BOOLEAN" }
func (b *Boolean) Inspect() string { return fmt.Sprintf("%t", b.Value) }

type Null struct{}

func (n *Null) Type() string    { return "NULL" }
func (n *Null) Inspect() string { return "null" }

type Array struct {
	Elements []Object
}

func (a *Array) Type() string { return "ARRAY" }
func (a *Array) Inspect() string {
	var out []string
	for _, e := range a.Elements {
		out = append(out, e.Inspect())
	}
	return "[" + strings.Join(out, ", ") + "]"
}

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() string    { return "RETURN_VALUE" }
func (rv *ReturnValue) Inspect() string { return rv.Value.Inspect() }

type ErrorObj struct {
	Message string
}

func (e *ErrorObj) Type() string    { return "ERROR" }
func (e *ErrorObj) Inspect() string { return "ERROR: " + e.Message }

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() string    { return "FUNCTION" }
func (f *Function) Inspect() string { return "function" }

type BuiltinFunction struct {
	Fn func(args ...Object) Object
}

func (b *BuiltinFunction) Type() string    { return "BUILTIN" }
func (b *BuiltinFunction) Inspect() string { return "builtin function" }

type Map struct {
	Pairs map[Object]Object
}

func (m *Map) Type() string { return "MAP" }
func (m *Map) Inspect() string {
	var out []string
	for k, v := range m.Pairs {
		out = append(out, fmt.Sprintf("%s: %s", k.Inspect(), v.Inspect()))
	}
	return "{" + strings.Join(out, ", ") + "}"
}

// Task represents the result of a spawned computation.
// For now this is a simple wrapper around a value – evaluation is still synchronous.
type Task struct {
	Result Object
	Done   bool
}

func (t *Task) Type() string    { return "TASK" }
func (t *Task) Inspect() string { return "task(" + t.Result.Inspect() + ")" }

// StructInstance is a simple record-like value created via `type` declarations.
type StructInstance struct {
	Name   string
	Fields map[string]Object
}

func (s *StructInstance) Type() string { return "STRUCT_INSTANCE" }
func (s *StructInstance) Inspect() string {
	parts := make([]string, 0, len(s.Fields))
	for k, v := range s.Fields {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v.Inspect()))
	}
	return fmt.Sprintf("%s(%s)", s.Name, strings.Join(parts, ", "))
}

// Module is a simple container for values defined in a `module` block.
type Module struct {
	Name string
	Env  *Environment
}

func (m *Module) Type() string { return "MODULE" }
func (m *Module) Inspect() string {
	return "module " + m.Name
}

// Route configuration for the tiny HTTP server helpers.
type routeDef struct {
	Method  string
	Path    string
	Handler *Function
}

var registeredRoutes []routeDef

type Environment struct {
	store map[string]Object
	outer *Environment
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	env := &Environment{store: s, outer: nil}

	// Add built-in print function
	env.store["print"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(arg.Inspect())
			}
			fmt.Println()
			return NULL
		},
	}

	// Add built-in len function
	env.store["len"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *String:
				return &Integer{Value: int64(len(arg.Value))}
			case *Map:
				return &Integer{Value: int64(len(arg.Pairs))}
			case *Array:
				return &Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("argument to `len` not supported, got %s", args[0].Type())
			}
		},
	}

	// Add built-in first function
	env.store["first"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			if args[0].Type() != "ARRAY" {
				return newError("argument to `first` must be ARRAY, got %s", args[0].Type())
			}
			array := args[0].(*Array)
			if len(array.Elements) > 0 {
				return array.Elements[0]
			}
			return NULL
		},
	}

	// Add built-in last function
	env.store["last"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			if args[0].Type() != "ARRAY" {
				return newError("argument to `last` must be ARRAY, got %s", args[0].Type())
			}
			array := args[0].(*Array)
			if len(array.Elements) > 0 {
				return array.Elements[len(array.Elements)-1]
			}
			return NULL
		},
	}

	// Add built-in rest function
	env.store["rest"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			if args[0].Type() != "ARRAY" {
				return newError("argument to `rest` must be ARRAY, got %s", args[0].Type())
			}
			array := args[0].(*Array)
			length := len(array.Elements)
			if length > 0 {
				newElements := make([]Object, length-1)
				copy(newElements, array.Elements[1:length])
				return &Array{Elements: newElements}
			}
			return NULL
		},
	}

	// Add built-in push function
	env.store["push"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			if args[0].Type() != "ARRAY" {
				return newError("first argument to `push` must be ARRAY, got %s", args[0].Type())
			}
			array := args[0].(*Array)
			length := len(array.Elements)
			newElements := make([]Object, length+1)
			copy(newElements, array.Elements)
			newElements[length] = args[1]
			return &Array{Elements: newElements}
		},
	}

	// Add built-in puts function
	env.store["puts"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}
			return NULL
		},
	}

	// Add built-in http_get function
	env.store["http_get"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			if args[0].Type() != "STRING" {
				return newError("argument to `http_get` must be STRING, got %s", args[0].Type())
			}
			url := args[0].(*String).Value
			resp, err := http.Get(url)
			if err != nil {
				return newError("http get error: %s", err)
			}
			defer resp.Body.Close()
			return &String{Value: resp.Status}
		},
	}

	// Add additional utility functions
	env.store["min"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			if args[0].Type() != "INTEGER" || args[1].Type() != "INTEGER" {
				return newError("arguments to `min` must be INTEGER, got %s and %s", args[0].Type(), args[1].Type())
			}
			a := args[0].(*Integer).Value
			b := args[1].(*Integer).Value
			if a < b {
				return &Integer{Value: a}
			}
			return &Integer{Value: b}
		},
	}

	env.store["max"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			if args[0].Type() != "INTEGER" || args[1].Type() != "INTEGER" {
				return newError("arguments to `max` must be INTEGER, got %s and %s", args[0].Type(), args[1].Type())
			}
			a := args[0].(*Integer).Value
			b := args[1].(*Integer).Value
			if a > b {
				return &Integer{Value: a}
			}
			return &Integer{Value: b}
		},
	}

	// Add tap function for pipeline debugging
	env.store["tap"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			// For tap, we expect the function as argument
			if args[0].Type() != "FUNCTION" && args[0].Type() != "BUILTIN" {
				return newError("argument to `tap` must be FUNCTION, got %s", args[0].Type())
			}
			// Note: tap is meant to be used in pipelines, so the actual value
			// being tapped will come from the pipeline context, not here
			return args[0]
		},
	}

	// Add inspect function for debugging
	env.store["inspect"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			fmt.Printf("[DEBUG] Type: %s, Value: %s\n", args[0].Type(), args[0].Inspect())
			return args[0]
		},
	}

	// range(n) -> [0, 1, ..., n-1]
	env.store["range"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			intArg, ok := args[0].(*Integer)
			if !ok {
				return newError("argument to `range` must be INTEGER, got %s", args[0].Type())
			}
			n := intArg.Value
			if n < 0 {
				return newError("argument to `range` must be non-negative, got %d", n)
			}
			elements := make([]Object, 0, n)
			for i := int64(0); i < n; i++ {
				elements = append(elements, &Integer{Value: i})
			}
			return &Array{Elements: elements}
		},
	}

	// HTTP server helpers used by examples/server.flowa
	// response(status, body) -> simple struct describing an HTTP response.
	env.store["response"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			statusInt, ok := args[0].(*Integer)
			if !ok {
				return newError("first argument to `response` must be INTEGER, got %s", args[0].Type())
			}
			bodyStr, ok := args[1].(*String)
			if !ok {
				return newError("second argument to `response` must be STRING, got %s", args[1].Type())
			}
			// We re-use StructInstance to avoid another dedicated type.
			return &StructInstance{
				Name: "Response",
				Fields: map[string]Object{
					"status": &Integer{Value: statusInt.Value},
					"body":   &String{Value: bodyStr.Value},
				},
			}
		},
	}

	// route(method, path, handler)
	env.store["route"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("wrong number of arguments. got=%d, want=3", len(args))
			}
			methodStr, ok := args[0].(*String)
			if !ok {
				return newError("first argument to `route` must be STRING, got %s", args[0].Type())
			}
			pathStr, ok := args[1].(*String)
			if !ok {
				return newError("second argument to `route` must be STRING, got %s", args[1].Type())
			}
			handlerFn, ok := args[2].(*Function)
			if !ok {
				return newError("third argument to `route` must be FUNCTION, got %s", args[2].Type())
			}
			registeredRoutes = append(registeredRoutes, routeDef{
				Method:  strings.ToUpper(methodStr.Value),
				Path:    pathStr.Value,
				Handler: handlerFn,
			})
			return NULL
		},
	}

	// listen(port)
	env.store["listen"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			portInt, ok := args[0].(*Integer)
			if !ok {
				return newError("argument to `listen` must be INTEGER, got %s", args[0].Type())
			}
			addr := fmt.Sprintf(":%d", portInt.Value)

			// Register handlers for all configured routes.
			for _, rt := range registeredRoutes {
				routeCopy := rt // capture
				http.HandleFunc(routeCopy.Path, func(w http.ResponseWriter, r *http.Request) {
					if strings.ToUpper(r.Method) != routeCopy.Method {
						http.NotFound(w, r)
						return
					}
					// Call user handler with a very small request placeholder (not exposed yet).
					result := applyFunction(routeCopy.Handler, []Object{NULL})
					// Expecting a Response-like StructInstance.
					resp, ok := result.(*StructInstance)
					if !ok {
						http.Error(w, "handler did not return response()", http.StatusInternalServerError)
						return
					}
					statusObj, okStatus := resp.Fields["status"].(*Integer)
					bodyObj, okBody := resp.Fields["body"].(*String)
					if !okStatus || !okBody {
						http.Error(w, "invalid response() object", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(int(statusObj.Value))
					_, _ = w.Write([]byte(bodyObj.Value))
				})
			}

			// Start blocking HTTP server.
			if err := http.ListenAndServe(addr, nil); err != nil {
				return newError("listen error: %s", err)
			}
			return NULL
		},
	}

	return env
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

var (
	NULL  = &Null{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

func Eval(node ast.Node, env *Environment) Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}
	case *ast.FunctionStatement:
		fn := &Function{
			Parameters: node.Parameters,
			Body:       node.Body,
			Env:        env,
		}
		env.Set(node.Name.Value, fn)
		return fn
	case *ast.AssignmentStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return val
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)
	case *ast.WhileStatement:
		return evalWhileStatement(node, env)
	case *ast.ForStatement:
		return evalForStatement(node, env)
	case *ast.ModuleStatement:
		return evalModuleStatement(node, env)
	case *ast.TypeStatement:
		return evalTypeStatement(node, env)
	case *ast.IntegerLiteral:
		return &Integer{Value: node.Value}
	case *ast.StringLiteral:
		return &String{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.NullLiteral:
		return NULL
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)
	case *ast.PipelineExpression:
		return evalPipelineExpression(node, env)
	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)
	case *ast.SpawnExpression:
		return evalSpawnExpression(node, env)
	case *ast.AwaitExpression:
		return evalAwaitExpression(node, env)
	case *ast.MapLiteral:
		return evalMapLiteral(node, env)
	case *ast.MemberExpression:
		return evalMemberExpression(node, env)
	// case *ast.MapIndexExpression:
	// 	mapObj := Eval(node.Map, env)
	// 	if isError(mapObj) {
	// 		return mapObj
	// 	}
	// 	index := Eval(node.Index, env)
	// 	if isError(index) {
	// 		return index
	// 	}
	// 	return evalMapIndexExpression(mapObj, index)
	// case *ast.ArrayLiteral:
	// 	elements := evalExpressions(node.Elements, env)
	// 	if len(elements) == 1 && isError(elements[0]) {
	// 		return elements[0]
	// 	}
	// 	return &Array{Elements: elements}
	// case *ast.IndexExpression:
	// 	left := Eval(node.Left, env)
	// 	if isError(left) {
	// 		return left
	// 	}
	// 	index := Eval(node.Index, env)
	// 	if isError(index) {
	// 		return index
	// 	}
	// 	return evalIndexExpression(left, index)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	}
	return NULL
}

func evalProgram(program *ast.Program, env *Environment) Object {
	var result Object
	for _, statement := range program.Statements {
		result = Eval(statement, env)
		if rv, ok := result.(*ReturnValue); ok {
			return rv.Value
		}
		if errObj, ok := result.(*ErrorObj); ok {
			return errObj
		}
	}
	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
	var result Object
	for _, statement := range block.Statements {
		result = Eval(statement, env)
		if result != nil {
			rt := result.Type()
			if rt == "RETURN_VALUE" || rt == "ERROR" {
				return result
			}
		}
	}
	return result
}

func evalPrefixExpression(operator string, right Object) Object {
	switch operator {
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	case "!":
		return evalBangOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalMinusPrefixOperatorExpression(right Object) Object {
	if right.Type() != "INTEGER" {
		return newError("unknown operator: -%s", right.Type())
	}
	value := right.(*Integer).Value
	return &Integer{Value: -value}
}

func evalBangOperatorExpression(right Object) Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalInfixExpression(operator string, left, right Object) Object {
	if left.Type() == "INTEGER" && right.Type() == "INTEGER" {
		return evalIntegerInfixExpression(operator, left, right)
	}
	if left.Type() == "STRING" && right.Type() == "STRING" {
		return evalStringInfixExpression(operator, left, right)
	}
	if operator == "==" {
		return evalEqualInfixExpression(left, right)
	}
	if operator == "!=" {
		return evalNotEqualInfixExpression(left, right)
	}
	if left.Type() != right.Type() {
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	}
	return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

func evalIntegerInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value
	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newError("division by zero")
		}
		return &Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(operator string, left, right Object) Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value
	return &String{Value: leftVal + rightVal}
}

func evalEqualInfixExpression(left, right Object) Object {
	if left.Type() == "INTEGER" && right.Type() == "INTEGER" {
		leftVal := left.(*Integer).Value
		rightVal := right.(*Integer).Value
		return nativeBoolToBooleanObject(leftVal == rightVal)
	}
	if left.Type() == "BOOLEAN" && right.Type() == "BOOLEAN" {
		leftVal := left.(*Boolean).Value
		rightVal := right.(*Boolean).Value
		return nativeBoolToBooleanObject(leftVal == rightVal)
	}
	if left.Type() == "STRING" && right.Type() == "STRING" {
		leftVal := left.(*String).Value
		rightVal := right.(*String).Value
		return nativeBoolToBooleanObject(leftVal == rightVal)
	}
	return FALSE
}

func evalNotEqualInfixExpression(left, right Object) Object {
	if left.Type() == "INTEGER" && right.Type() == "INTEGER" {
		leftVal := left.(*Integer).Value
		rightVal := right.(*Integer).Value
		return nativeBoolToBooleanObject(leftVal != rightVal)
	}
	if left.Type() == "BOOLEAN" && right.Type() == "BOOLEAN" {
		leftVal := left.(*Boolean).Value
		rightVal := right.(*Boolean).Value
		return nativeBoolToBooleanObject(leftVal != rightVal)
	}
	if left.Type() == "STRING" && right.Type() == "STRING" {
		leftVal := left.(*String).Value
		rightVal := right.(*String).Value
		return nativeBoolToBooleanObject(leftVal != rightVal)
	}
	return TRUE
}

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	val, ok := env.Get(node.Value)
	if !ok {
		return newError("identifier not found: %s", node.Value)
	}
	return val
}

func evalExpressions(exps []ast.Expression, env *Environment) []Object {
	var result []Object
	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *BuiltinFunction:
		return fn.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)
	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}
	return env
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func evalMapLiteral(node *ast.MapLiteral, env *Environment) Object {
	pairs := make(map[Object]Object)
	for _, pair := range node.Pairs {
		key := Eval(pair.Key, env)
		if isError(key) {
			return key
		}
		value := Eval(pair.Value, env)
		if isError(value) {
			return value
		}
		pairs[key] = value
	}
	return &Map{Pairs: pairs}
}

func evalIfExpression(ie *ast.IfExpression, env *Environment) Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func evalWhileStatement(ws *ast.WhileStatement, env *Environment) Object {
	var result Object = NULL
	for {
		condition := Eval(ws.Condition, env)
		if isError(condition) {
			return condition
		}
		if !isTruthy(condition) {
			break
		}
		result = Eval(ws.Body, env)
		if result != nil {
			if result.Type() == "RETURN_VALUE" || result.Type() == "ERROR" {
				return result
			}
		}
	}
	return result
}

func evalForStatement(fs *ast.ForStatement, env *Environment) Object {
	iterable := Eval(fs.Value, env)
	if isError(iterable) {
		return iterable
	}
	array, ok := iterable.(*Array)
	if !ok {
		return newError("for-loop value must be ARRAY, got %s", iterable.Type())
	}

	var result Object = NULL
	for _, elem := range array.Elements {
		// New inner scope for each iteration
		iterEnv := NewEnclosedEnvironment(env)
		iterEnv.Set(fs.Iterator.Value, elem)
		result = Eval(fs.Body, iterEnv)
		if result != nil {
			if result.Type() == "RETURN_VALUE" || result.Type() == "ERROR" {
				return result
			}
		}
	}
	return result
}

func evalPipelineExpression(pe *ast.PipelineExpression, env *Environment) Object {
	leftVal := Eval(pe.Left, env)
	if isError(leftVal) {
		return leftVal
	}

	switch right := pe.Right.(type) {
	case *ast.Identifier:
		fn := evalIdentifier(right, env)
		if isError(fn) {
			return fn
		}
		return applyFunction(fn, []Object{leftVal})
	case *ast.CallExpression:
		// Evaluate function part and arguments separately
		fn := Eval(right.Function, env)
		if isError(fn) {
			return fn
		}
		args := evalExpressions(right.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		// Prepend pipeline value
		allArgs := append([]Object{leftVal}, args...)
		return applyFunction(fn, allArgs)
	case *ast.PipelineExpression:
		// Allow chaining inside the right-hand side
		rightWithLeft := &ast.PipelineExpression{
			Token: pe.Token,
			Left:  &ast.PipelineExpression{Token: pe.Token, Left: pe.Left, Right: right.Left},
			Right: right.Right,
		}
		return evalPipelineExpression(rightWithLeft, env)
	default:
		return newError("invalid right-hand side of pipeline: %T", pe.Right)
	}
}

func evalSpawnExpression(se *ast.SpawnExpression, env *Environment) Object {
	// Synchronous "spawn": evaluate the expression immediately and wrap in a Task.
	val := Eval(se.Call, env)
	if isError(val) {
		return val
	}
	return &Task{Result: val, Done: true}
}

func evalAwaitExpression(ae *ast.AwaitExpression, env *Environment) Object {
	val := Eval(ae.Value, env)
	if isError(val) {
		return val
	}
	task, ok := val.(*Task)
	if !ok {
		return newError("await can only be used on tasks, got %s", val.Type())
	}
	if !task.Done {
		// For now everything is eager, so Done should always be true.
		return NULL
	}
	return task.Result
}

func evalModuleStatement(ms *ast.ModuleStatement, env *Environment) Object {
	moduleEnv := NewEnclosedEnvironment(env)
	// Evaluate body inside the module environment
	bodyResult := Eval(ms.Body, moduleEnv)
	if isError(bodyResult) {
		return bodyResult
	}
	mod := &Module{
		Name: ms.Name.Value,
		Env:  moduleEnv,
	}
	env.Set(ms.Name.Value, mod)
	return mod
}

func evalTypeStatement(ts *ast.TypeStatement, env *Environment) Object {
	// Create a constructor function that builds StructInstance values.
	constructor := &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != len(ts.Fields) {
				return newError("wrong number of arguments to constructor %s. got=%d, want=%d",
					ts.Name.Value, len(args), len(ts.Fields))
			}
			fields := make(map[string]Object, len(ts.Fields))
			for i, field := range ts.Fields {
				fields[field.Value] = args[i]
			}
			return &StructInstance{
				Name:   ts.Name.Value,
				Fields: fields,
			}
		},
	}
	env.Set(ts.Name.Value, constructor)
	return constructor
}

func evalMemberExpression(me *ast.MemberExpression, env *Environment) Object {
	obj := Eval(me.Object, env)
	if isError(obj) {
		return obj
	}
	propName := me.Property.Value

	switch v := obj.(type) {
	case *StructInstance:
		if val, ok := v.Fields[propName]; ok {
			return val
		}
		return NULL
	case *Module:
		if val, ok := v.Env.Get(propName); ok {
			return val
		}
		return NULL
	case *Map:
		// Allow map["key"] style via member for string-like keys
		key := &String{Value: propName}
		if val, ok := v.Pairs[key]; ok {
			return val
		}
		return NULL
	default:
		return newError("type %s does not support member access", obj.Type())
	}
}

func isTruthy(obj Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == "ERROR"
	}
	return false
}

func newError(format string, a ...interface{}) *ErrorObj {
	return &ErrorObj{Message: fmt.Sprintf(format, a...)}
}

func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}
