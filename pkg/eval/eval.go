package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"flowa/pkg/ast"
	"flowa/pkg/lexer"
	"flowa/pkg/opcode"
	"flowa/pkg/parser"

	"github.com/gorilla/websocket"
	"gopkg.in/gomail.v2"
)

// Object is the interface that all Flowa values implement.
type Object interface {
	Kind() ObjectKind
	Inspect() string
}

type Integer struct {
	Value int64
	kind  ObjectKind
}

func (i *Integer) Kind() ObjectKind {
	if i.kind == 0 { // Lazy initialization for any Integer not created through NewInteger
		i.kind = KindInteger
	}
	return i.kind
}
func (i *Integer) Inspect() string { return fmt.Sprintf("%d", i.Value) }

type String struct {
	Value string
	kind  ObjectKind
}

func (s *String) Kind() ObjectKind { return KindString }
func (s *String) Inspect() string  { return s.Value }

type Native struct {
	Value interface{}
	kind  ObjectKind
}

func (n *Native) Kind() ObjectKind { return KindNative }
func (n *Native) Inspect() string  { return fmt.Sprintf("%v", n.Value) }

type Boolean struct {
	Value bool
	kind  ObjectKind
}

func (b *Boolean) Kind() ObjectKind { return KindBoolean }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Null struct {
	kind ObjectKind
}

func (n *Null) Kind() ObjectKind { return KindNull }
func (n *Null) Inspect() string  { return "null" }

type Array struct {
	Elements []Object
	kind     ObjectKind
}

func (a *Array) Kind() ObjectKind { return KindArray }
func (a *Array) Inspect() string {
	var out []string
	for _, e := range a.Elements {
		out = append(out, e.Inspect())
	}
	return "[" + strings.Join(out, ", ") + "]"
}

type ReturnValue struct {
	Value Object
	kind  ObjectKind
}

func (rv *ReturnValue) Kind() ObjectKind { return KindReturnValue }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type ErrorObj struct {
	Message string
	kind    ObjectKind
}

func (e *ErrorObj) Kind() ObjectKind { return KindError }
func (e *ErrorObj) Inspect() string  { return "ERROR: " + e.Message }

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
	kind       ObjectKind

	// Slot-based optimization: layout of local variables
	SlotCount    int                 // Number of slots allocated for this function's locals
	SlotNames    []string            // Names of parameters + local variables (indexed by slot)
	SlotIDs      []*InternedString   // Interned identifiers for fast lookups
	Instructions opcode.Instructions // Bytecode for this function
}

func (f *Function) Kind() ObjectKind { return KindFunction }
func (f *Function) Inspect() string  { return "function" }

type BuiltinFunction struct {
	Fn   func(args ...Object) Object
	kind ObjectKind
}

func (b *BuiltinFunction) Kind() ObjectKind { return KindBuiltin }
func (b *BuiltinFunction) Inspect() string  { return "builtin function" }

type Map struct {
	Pairs map[Object]Object
	kind  ObjectKind
}

func (m *Map) Kind() ObjectKind { return KindMap }
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
	Fn     *Function
	Args   []Object
	kind   ObjectKind
	Done   bool
	Result Object
}

func (t *Task) Kind() ObjectKind { return KindTask }
func (t *Task) Inspect() string  { return "task(" + t.Result.Inspect() + ")" }
func (t *Task) Await() Object {
	for !t.Done {
		time.Sleep(1 * time.Millisecond)
	}
	return t.Result
}

// StructInstance is a simple record-like value created via `type` declarations.
type StructInstance struct {
	Name   string
	Fields map[string]Object
	kind   ObjectKind
}

func (s *StructInstance) Kind() ObjectKind { return KindStructInstance }
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

func (m *Module) Kind() ObjectKind { return KindModule }
func (m *Module) Inspect() string {
	return "module " + m.Name
}

// Route configuration for HTTP server with path parameter support
type routeDef struct {
	Method      string
	Path        string         // Original path like "/users/:id"
	PathPattern string         // Regex pattern as string (for debugging / introspection)
	PathRegex   *regexp.Regexp // Precompiled regex for matching
	ParamNames  []string       // Names of path parameters
	Handler     *Function
	Middlewares []Object // Route-specific middleware
}

var registeredRoutes []routeDef
var globalMiddlewares []Object // Global middleware applied to all routes

type Environment struct {
	store     map[string]Object // Map storage for globals and dynamic bindings
	outer     *Environment      // Parent environment for scope chain
	slots     []Object          // Pre-allocated slots for local variables (function scope optimization)
	slotIndex map[string]int    // Maps variable names to slot indices
	slotNames []string          // Names of variables in slots (for debugging)
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	env := &Environment{store: s, outer: nil}

	// -----------------------------
	// Core utility & I/O builtins
	// -----------------------------

	// Add built-in print function (debug / general I/O – relatively slow)
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

			switch obj := args[0].(type) {
			case *Array:
				return &Integer{Value: int64(len(obj.Elements))}
			case *String:
				return &Integer{Value: int64(len(obj.Value))}
			case *Map:
				return &Integer{Value: int64(len(obj.Pairs))}
			default:
				return newError("argument to `len` not supported, got %s", obj.Kind())
			}
		},
	}

	// Add built-in first function
	env.store["first"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			if args[0].Kind() != KindArray {
				return newError("argument to `first` must be ARRAY, got %s", args[0].Kind())
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
			if args[0].Kind() != KindArray {
				return newError("argument to `last` must be ARRAY, got %s", args[0].Kind())
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
			if args[0].Kind() != KindArray {
				return newError("argument to `rest` must be ARRAY, got %s", args[0].Kind())
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
			if args[0].Kind() != KindArray {
				return newError("first argument to `push` must be ARRAY, got %s", args[0].Kind())
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
			if args[0].Kind() != KindString {
				return newError("argument to `http_get` must be STRING, got %s", args[0].Kind())
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

	env.store["async_http_get"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			urlObj, ok := args[0].(*String)
			if !ok {
				return newError("argument to `async_http_get` must be STRING, got %s", args[0].Kind())
			}

			task := &Task{Done: false}

			go func() {
				resp, err := http.Get(urlObj.Value)
				if err != nil {
					task.Result = &ErrorObj{Message: fmt.Sprintf("http error: %s", err)}
					task.Done = true
					return
				}
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				task.Result = &String{Value: string(body)}
				task.Done = true
			}()

			return task
		},
	}

	// Add additional utility functions
	env.store["min"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			if args[0].Kind() != KindInteger || args[1].Kind() != KindInteger {
				return newError("arguments to `min` must be INTEGER, got %s and %s", args[0].Kind(), args[1].Kind())
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
			if args[0].Kind() != KindInteger || args[1].Kind() != KindInteger {
				return newError("arguments to `max` must be INTEGER, got %s and %s", args[0].Kind(), args[1].Kind())
			}
			a := args[0].(*Integer).Value
			b := args[1].(*Integer).Value
			if a > b {
				return &Integer{Value: a}
			}
			return &Integer{Value: b}
		},
	}

	// json module
	jsonModule := &StructInstance{
		Name: "JSON",
		Fields: map[string]Object{
			"encode": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) != 1 {
						return newError("wrong number of arguments. got=%d, want=1", len(args))
					}
					native := flowaToNative(args[0])
					bytes, err := json.Marshal(native)
					if err != nil {
						return newError("json encode error: %s", err)
					}
					return &String{Value: string(bytes)}
				},
			},
			"decode": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) != 1 {
						return newError("wrong number of arguments. got=%d, want=1", len(args))
					}
					if args[0].Kind() != KindString {
						return newError("argument to `json.decode` must be STRING, got %s", args[0].Kind())
					}
					var native interface{}
					err := json.Unmarshal([]byte(args[0].(*String).Value), &native)
					if err != nil {
						return newError("json decode error: %s", err)
					}
					return nativeToFlowa(native)
				},
			},
		},
	}
	env.store["json"] = jsonModule

	// response helpers
	responseModule := &StructInstance{
		Name: "ResponseHelpers",
		Fields: map[string]Object{
			"json": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) < 1 || len(args) > 2 {
						return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
					}
					// args[0] is data, args[1] is status (optional)
					native := flowaToNative(args[0])
					bytes, err := json.Marshal(native)
					if err != nil {
						return newError("json marshal error: %s", err)
					}
					status := int64(200)
					if len(args) == 2 {
						if s, ok := args[1].(*Integer); ok {
							status = s.Value
						}
					}

					fields := make(map[string]Object)
					fields["status"] = &Integer{Value: status}
					fields["body"] = &String{Value: string(bytes)}
					fields["headers"] = &Map{Pairs: map[Object]Object{
						&String{Value: "Content-Type"}: &String{Value: "application/json"},
					}}
					return &StructInstance{Name: "Response", Fields: fields}
				},
			},
			"text": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) < 1 || len(args) > 2 {
						return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
					}
					text := args[0].Inspect()
					if s, ok := args[0].(*String); ok {
						text = s.Value
					}
					status := int64(200)
					if len(args) == 2 {
						if s, ok := args[1].(*Integer); ok {
							status = s.Value
						}
					}
					fields := make(map[string]Object)
					fields["status"] = &Integer{Value: status}
					fields["body"] = &String{Value: text}
					fields["headers"] = &Map{Pairs: map[Object]Object{
						&String{Value: "Content-Type"}: &String{Value: "text/plain"},
					}}
					return &StructInstance{Name: "Response", Fields: fields}
				},
			},
			"html": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) < 1 || len(args) > 2 {
						return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
					}
					html := args[0].Inspect()
					if s, ok := args[0].(*String); ok {
						html = s.Value
					}
					status := int64(200)
					if len(args) == 2 {
						if s, ok := args[1].(*Integer); ok {
							status = s.Value
						}
					}
					fields := make(map[string]Object)
					fields["status"] = &Integer{Value: status}
					fields["body"] = &String{Value: html}
					fields["headers"] = &Map{Pairs: map[Object]Object{
						&String{Value: "Content-Type"}: &String{Value: "text/html"},
					}}
					return &StructInstance{Name: "Response", Fields: fields}
				},
			},
			"redirect": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) < 1 || len(args) > 2 {
						return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
					}
					url := ""
					if s, ok := args[0].(*String); ok {
						url = s.Value
					} else {
						return newError("redirect url must be STRING")
					}
					status := int64(302)
					if len(args) == 2 {
						if s, ok := args[1].(*Integer); ok {
							status = s.Value
						}
					}
					fields := make(map[string]Object)
					fields["status"] = &Integer{Value: status}
					fields["body"] = &String{Value: ""}
					fields["headers"] = &Map{Pairs: map[Object]Object{
						&String{Value: "Location"}: &String{Value: url},
					}}
					return &StructInstance{Name: "Response", Fields: fields}
				},
			},
		},
	}
	env.store["response"] = responseModule

	// config module
	configModule := &StructInstance{
		Name: "Config",
		Fields: map[string]Object{
			"env": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) < 1 || len(args) > 2 {
						return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
					}
					key := ""
					if s, ok := args[0].(*String); ok {
						key = s.Value
					} else {
						return newError("env key must be STRING")
					}
					// TODO: Implement os.Getenv
					// For now, return default or empty
					// We need to import "os"
					val := os.Getenv(key)
					if val == "" && len(args) == 2 {
						if s, ok := args[1].(*String); ok {
							val = s.Value
						}
					}
					return &String{Value: val}
				},
			},
		},
	}
	env.store["config"] = configModule

	// middleware module
	middlewareModule := &StructInstance{
		Name: "Middleware",
		Fields: map[string]Object{
			"logger": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					// Returns a middleware function: def(req, next)
					return &BuiltinFunction{
						Fn: func(mwArgs ...Object) Object {
							if len(mwArgs) != 2 {
								return newError("middleware must be called with (req, next)")
							}
							req := mwArgs[0]
							next := mwArgs[1]

							// Log request
							method := "UNKNOWN"
							path := "UNKNOWN"
							if m, ok := req.(*StructInstance).Fields["method"].(*String); ok {
								method = m.Value
							}
							if p, ok := req.(*StructInstance).Fields["path"].(*String); ok {
								path = p.Value
							}
							fmt.Printf("[LOG] %s %s\n", method, path)

							// Call next
							if nextFn, ok := next.(*BuiltinFunction); ok {
								return nextFn.Fn()
							}
							// If next is a Function (Flowa function)
							if nextFn, ok := next.(*Function); ok {
								return applyFunction(nextFn, []Object{})
							}
							return NULL
						},
					}
				},
			},
			"cors": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					// Returns a middleware function that adds CORS headers
					return &BuiltinFunction{
						Fn: func(mwArgs ...Object) Object {
							if len(mwArgs) != 2 {
								return newError("middleware must be called with (req, next)")
							}
							// req := mwArgs[0]
							next := mwArgs[1]

							// Call next to get response
							var result Object
							if nextFn, ok := next.(*BuiltinFunction); ok {
								result = nextFn.Fn()
							} else if nextFn, ok := next.(*Function); ok {
								result = applyFunction(nextFn, []Object{})
							} else {
								return NULL
							}

							// Add CORS headers to response
							if resp, ok := result.(*StructInstance); ok {
								if headers, ok := resp.Fields["headers"].(*Map); ok {
									headers.Pairs[&String{Value: "Access-Control-Allow-Origin"}] = &String{Value: "*"}
									headers.Pairs[&String{Value: "Access-Control-Allow-Methods"}] = &String{Value: "GET, POST, PUT, DELETE, OPTIONS"}
									headers.Pairs[&String{Value: "Access-Control-Allow-Headers"}] = &String{Value: "Content-Type, Authorization"}
								}
							}
							return result
						},
					}
				},
			},
		},
	}
	env.store["middleware"] = middlewareModule

	// time module - high resolution timing helpers for benchmarking
	timeModule := &StructInstance{
		Name: "Time",
		Fields: map[string]Object{
			// time.now_ms() -> current time in milliseconds since Unix epoch
			"now_ms": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) != 0 {
						return newError("time.now_ms expects 0 arguments")
					}
					ms := time.Now().UnixNano() / int64(time.Millisecond)
					return &Integer{Value: ms}
				},
			},
			// time.since_ms(start_ms) -> elapsed milliseconds since start_ms
			"since_ms": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) != 1 {
						return newError("time.since_ms expects 1 INTEGER argument (start_ms)")
					}
					if args[0].Kind() != KindInteger {
						return newError("time.since_ms argument must be INTEGER, got %s", args[0].Kind())
					}
					start := args[0].(*Integer).Value
					nowMs := time.Now().UnixNano() / int64(time.Millisecond)
					return &Integer{Value: nowMs - start}
				},
			},
			// time.since_s(start_ms[, precision]) -> formatted seconds string with fractional part
			"since_s": &BuiltinFunction{
				Fn: func(args ...Object) Object {
					if len(args) < 1 || len(args) > 2 {
						return newError("time.since_s expects 1 or 2 arguments: (start_ms[, precision])")
					}
					if args[0].Kind() != KindInteger {
						return newError("time.since_s first argument must be INTEGER (start_ms), got %s", args[0].Kind())
					}
					precision := 3
					if len(args) == 2 {
						if args[1].Kind() != KindInteger {
							return newError("time.since_s precision argument must be INTEGER, got %s", args[1].Kind())
						}
						precision = int(args[1].(*Integer).Value)
						if precision < 0 {
							precision = 0
						}
					}
					start := args[0].(*Integer).Value
					nowMs := time.Now().UnixNano() / int64(time.Millisecond)
					deltaMs := nowMs - start
					secs := float64(deltaMs) / 1000.0
					fmtStr := fmt.Sprintf("%%.%df", precision)
					return &String{Value: fmt.Sprintf(fmtStr, secs)}
				},
			},
		},
	}
	env.store["time"] = timeModule

	// mail module - SMTP email sending (similar to nodemailer)
	mailModule := &StructInstance{
		Name:   "Mail",
		Fields: make(map[string]Object),
	}

	// mail.send - send email via SMTP
	mailModule.Fields["send"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			// args[0] should be a map with: to, from, subject, body
			mailMap, ok := args[0].(*Map)
			if !ok {
				return newError("argument to mail.send must be a Map")
			}

			// Extract fields by iterating over map (since keys are pointers)
			to := ""
			from := ""
			subject := ""
			body := ""
			html := ""

			for k, v := range mailMap.Pairs {
				keyStr, ok := k.(*String)
				if !ok {
					continue
				}

				valStr, ok := v.(*String)
				if !ok {
					continue
				}

				switch keyStr.Value {
				case "to":
					to = valStr.Value
				case "from":
					from = valStr.Value
				case "subject":
					subject = valStr.Value
				case "body":
					body = valStr.Value
				case "html":
					html = valStr.Value
				}
			}

			// Read SMTP config from environment variables
			smtpHost := os.Getenv("SMTP_HOST")
			smtpPortStr := os.Getenv("SMTP_PORT")
			smtpUser := os.Getenv("SMTP_USER")
			smtpPass := os.Getenv("SMTP_PASS")

			if smtpHost == "" || smtpPortStr == "" {
				return newError("SMTP_HOST and SMTP_PORT environment variables must be set")
			}

			smtpPort, err := strconv.Atoi(smtpPortStr)
			if err != nil {
				return newError("SMTP_PORT must be an integer")
			}

			// Default from if not provided
			if from == "" {
				from = smtpUser
				if from == "" {
					from = "noreply@example.com"
				}
			}

			// Create message using gomail
			m := gomail.NewMessage()
			m.SetHeader("From", from)
			m.SetHeader("To", to)
			m.SetHeader("Subject", subject)

			// Use HTML if provided, otherwise plain text
			if html != "" {
				m.SetBody("text/html", html)
			} else {
				m.SetBody("text/plain", body)
			}

			// Send using gomail Dialer
			d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)

			// Send the email
			if err := d.DialAndSend(m); err != nil {
				return newError("failed to send email: %s", err)
			}

			return TRUE
		},
	}

	// mail.send_template - send email using template
	mailSendFn := mailModule.Fields["send"].(*BuiltinFunction)
	mailModule.Fields["send_template"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			// args[0] is template content, args[1] is data map
			template := ""
			if s, ok := args[0].(*String); ok {
				template = s.Value
			}

			// Simple template replacement
			dataMap, ok := args[1].(*Map)
			if !ok {
				return newError("second argument to mail.send_template must be a Map")
			}

			// Replace {{key}} with values from dataMap
			body := template
			for k, v := range dataMap.Pairs {
				keyStr := ""
				if s, ok := k.(*String); ok {
					keyStr = s.Value
				}
				valStr := v.Inspect()
				body = strings.ReplaceAll(body, "{{"+keyStr+"}}", valStr)
			}

			// Build send map by iterating to find fields
			sendMap := &Map{Pairs: make(map[Object]Object)}
			for k, v := range dataMap.Pairs {
				if keyStr, ok := k.(*String); ok {
					switch keyStr.Value {
					case "to", "from", "subject":
						sendMap.Pairs[k] = v
					}
				}
			}
			sendMap.Pairs[&String{Value: "body"}] = &String{Value: body}

			// Call mail.send
			return mailSendFn.Fn(sendMap)
		},
	}

	// mail.queue - send email in background
	mailModule.Fields["queue"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			// Send in background
			go func() {
				mailSendFn.Fn(args...)
			}()
			return TRUE
		},
	}

	env.store["mail"] = mailModule

	// auth module
	authModule := &StructInstance{
		Name:   "Auth",
		Fields: make(map[string]Object),
	}
	authModule.Fields["hash_password"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			pass, ok := args[0].(*String)
			if !ok {
				return newError("argument to auth.hash_password must be a String")
			}
			hash, err := hashPassword(pass.Value)
			if err != nil {
				return newError("failed to hash password: %s", err)
			}
			return &String{Value: hash}
		},
	}
	authModule.Fields["verify_password"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			hash, ok := args[0].(*String)
			if !ok {
				return newError("first argument to auth.verify_password must be a String")
			}
			pass, ok := args[1].(*String)
			if !ok {
				return newError("second argument to auth.verify_password must be a String")
			}
			valid := verifyPassword(hash.Value, pass.Value)
			if valid {
				return TRUE
			}
			return FALSE
		},
	}
	env.store["auth"] = authModule

	// jwt module
	jwtModule := &StructInstance{
		Name:   "JWT",
		Fields: make(map[string]Object),
	}
	jwtModule.Fields["sign"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("wrong number of arguments. got=%d, want=3", len(args))
			}
			payload, ok := args[0].(*Map)
			if !ok {
				return newError("first argument to jwt.sign must be a Map")
			}
			secret, ok := args[1].(*String)
			if !ok {
				return newError("second argument to jwt.sign must be a String")
			}
			expiresIn, ok := args[2].(*String)
			if !ok {
				return newError("third argument to jwt.sign must be a String")
			}

			// Convert Flowa Map to native map
			nativePayload := flowaToNative(payload).(map[string]interface{})
			token, err := signToken(nativePayload, secret.Value, expiresIn.Value)
			if err != nil {
				return newError("failed to sign token: %s", err)
			}
			return &String{Value: token}
		},
	}
	jwtModule.Fields["verify"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			token, ok := args[0].(*String)
			if !ok {
				return newError("first argument to jwt.verify must be a String")
			}
			secret, ok := args[1].(*String)
			if !ok {
				return newError("second argument to jwt.verify must be a String")
			}

			claims, err := verifyToken(token.Value, secret.Value)
			if err != nil {
				return NULL // Or error? usually null for invalid token
			}
			return nativeToFlowa(claims)
		},
	}
	env.store["jwt"] = jwtModule

	// ws module
	wsModule := &StructInstance{
		Name:   "WebSocket",
		Fields: make(map[string]Object),
	}
	wsModule.Fields["upgrade"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			reqObj, ok := args[0].(*StructInstance)
			if !ok || reqObj.Name != "Request" {
				return newError("argument to ws.upgrade must be a Request object")
			}

			// Extract native w and r
			nativeW, okW := reqObj.Fields["_native_writer"].(*Native)
			nativeR, okR := reqObj.Fields["_native_req"].(*Native)

			if !okW || !okR {
				return newError("invalid request object for websocket upgrade")
			}

			w := nativeW.Value.(http.ResponseWriter)
			r := nativeR.Value.(*http.Request)

			conn, err := upgradeToWebSocket(w, r)
			if err != nil {
				fmt.Printf("WebSocket upgrade error: %s\n", err)
				return NULL
			}
			return conn
		},
	}
	wsModule.Fields["send"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			conn, ok := args[0].(*WebSocketConnection)
			if !ok {
				return newError("first argument to ws.send must be a WebSocketConnection")
			}
			msg, ok := args[1].(*String)
			if !ok {
				return newError("second argument to ws.send must be a String")
			}

			err := conn.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Value))
			if err != nil {
				return newError("failed to send message: %s", err)
			}
			return TRUE
		},
	}
	wsModule.Fields["read"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			conn, ok := args[0].(*WebSocketConnection)
			if !ok {
				return newError("argument to ws.read must be a WebSocketConnection")
			}

			_, message, err := conn.Conn.ReadMessage()
			if err != nil {
				return NULL // Disconnected
			}
			return &String{Value: string(message)}
		},
	}
	wsModule.Fields["close"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			conn, ok := args[0].(*WebSocketConnection)
			if !ok {
				return newError("argument to ws.close must be a WebSocketConnection")
			}
			conn.Conn.Close()
			return TRUE
		},
	}
	env.store["websocket"] = wsModule

	// Add tap function for pipeline debugging
	env.store["tap"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			// For tap, we expect the function as argument
			if args[0].Kind() != KindFunction && args[0].Kind() != KindBuiltin {
				return newError("argument to `tap` must be FUNCTION, got %s", args[0].Kind())
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
			fmt.Printf("[DEBUG] Kind: %s, Value: %s\n", args[0].Kind(), args[0].Inspect())
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
				return newError("argument to `range` must be INTEGER, got %s", args[0].Kind())
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

	// -----------------------------------------------
	// Numeric performance-builtins (fast primitives)
	// -----------------------------------------------
	//
	// These are implemented fully in Go and operate on primitive int64
	// values without allocating per-iteration Integer objects. They are
	// intended for compute-heavy workloads (like benchmarks or analytics)
	// where you would otherwise write large numeric loops in Flowa.
	//
	// NOTE: They are explicit opt-in APIs – callers must choose to use
	// them, which keeps semantics obvious and avoids surprising behavior.

	// fast_sum_to(n) -> sum_{i=0}^{n-1} i
	// Example: fast_sum_to(10) == 45
	env.store["fast_sum_to"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("fast_sum_to expects 1 INTEGER argument (n)")
			}
			nInt, ok := args[0].(*Integer)
			if !ok {
				return newError("fast_sum_to argument must be INTEGER, got %s", args[0].Kind())
			}
			n := nInt.Value
			if n < 0 {
				return newError("fast_sum_to argument must be non-negative, got %d", n)
			}

			// Use pure Go int64 loop; only allocate final Integer.
			var sum int64
			for i := int64(0); i < n; i++ {
				sum += i
			}
			return &Integer{Value: sum}
		},
	}

	// fast_sum_range(start, end) -> sum_{i=start}^{end-1} i
	// Useful when you need an offset range; still avoids per-iteration
	// boxing and expression evaluation in the interpreter.
	env.store["fast_sum_range"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("fast_sum_range expects 2 INTEGER arguments (start, end)")
			}
			startInt, ok1 := args[0].(*Integer)
			endInt, ok2 := args[1].(*Integer)
			if !ok1 || !ok2 {
				return newError("fast_sum_range arguments must be INTEGER, got %s and %s", args[0].Kind(), args[1].Kind())
			}
			start := startInt.Value
			end := endInt.Value
			if end < start {
				return newError("fast_sum_range end must be >= start, got start=%d, end=%d", start, end)
			}

			var sum int64
			for i := start; i < end; i++ {
				sum += i
			}
			return &Integer{Value: sum}
		},
	}

	// fast_repeat(n, fn) -> calls fn(i) for i in [0, n), discarding results.
	// This still crosses the interpreter boundary once per iteration, but
	// avoids any extra allocation for the loop structure itself. It's a
	// middle ground between fully interpreted loops and fully-native
	// operations like fast_sum_to.
	env.store["fast_repeat"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("fast_repeat expects 2 arguments (n: INTEGER, fn: FUNCTION)")
			}
			nInt, ok := args[0].(*Integer)
			if !ok {
				return newError("fast_repeat first argument must be INTEGER, got %s", args[0].Kind())
			}
			fnObj := args[1]
			if fnObj.Kind() != KindFunction && fnObj.Kind() != KindBuiltin {
				return newError("fast_repeat second argument must be FUNCTION or BUILTIN, got %s", fnObj.Kind())
			}
			n := nInt.Value
			if n < 0 {
				return newError("fast_repeat n must be non-negative, got %d", n)
			}

			// Call fn(i) for each i; we only allocate one Integer per call.
			for i := int64(0); i < n; i++ {
				idx := &Integer{Value: i}
				switch f := fnObj.(type) {
				case *Function:
					applyFunction(f, []Object{idx})
				case *BuiltinFunction:
					_ = f.Fn(idx)
				default:
					return newError("fast_repeat encountered non-callable: %s", fnObj.Kind())
				}
			}
			return NULL
		},
	}

	// HTTP server helpers used by examples/server.flowa
	// Old response(status, body) function replaced by response module

	// route(method, path, handler) or route(method, path, handler, middlewares)

	// route(method, path, handler) or route(method, path, handler, middlewares)
	env.store["route"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) < 3 || len(args) > 4 {
				return newError("wrong number of arguments. got=%d, want=3 or 4", len(args))
			}
			methodStr, ok := args[0].(*String)
			if !ok {
				return newError("first argument to `route` must be STRING, got %s", args[0].Kind())
			}
			pathStr, ok := args[1].(*String)
			if !ok {
				return newError("second argument to `route` must be STRING, got %s", args[1].Kind())
			}
			handlerFn, ok := args[2].(*Function)
			if !ok {
				return newError("third argument to `route` must be FUNCTION, got %s", args[2].Kind())
			}

			// Parse path to extract parameter names and create regex pattern
			path := pathStr.Value
			paramNames := []string{}
			pattern := path

			// Find all :param patterns
			parts := strings.Split(path, "/")
			patternParts := make([]string, len(parts))
			for i, part := range parts {
				if strings.HasPrefix(part, ":") {
					paramName := part[1:]
					paramNames = append(paramNames, paramName)
					patternParts[i] = "([^/]+)" // Match any non-slash characters
				} else {
					patternParts[i] = part
				}
			}
			pattern = strings.Join(patternParts, "/")

			// Precompile regex once at route registration time so we don't
			// pay regexp.Compile on every HTTP request.
			var compiled *regexp.Regexp
			if len(paramNames) > 0 {
				var err error
				compiled, err = regexp.Compile("^" + pattern + "$")
				if err != nil {
					return newError("invalid route pattern %q: %s", path, err)
				}
			}

			// Optional middleware
			var middlewares []Object
			if len(args) == 4 {
				// Support array of middleware or single middleware
				if arr, ok := args[3].(*Array); ok {
					middlewares = arr.Elements
				} else {
					middlewares = []Object{args[3]}
				}
			}

			registeredRoutes = append(registeredRoutes, routeDef{
				Method:      strings.ToUpper(methodStr.Value),
				Path:        path,
				PathPattern: "^" + pattern + "$",
				PathRegex:   compiled,
				ParamNames:  paramNames,
				Handler:     handlerFn,
				Middlewares: middlewares,
			})
			return NULL
		},
	}

	// use_middleware(middleware_fn) - register global middleware
	env.store["use_middleware"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			middlewareFn, ok := args[0].(*Function)
			if !ok {
				return newError("argument to `use_middleware` must be FUNCTION, got %s", args[0].Kind())
			}
			globalMiddlewares = append(globalMiddlewares, middlewareFn)
			return NULL
		},
	}

	// listen(port)
	env.store["listen"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			var port string
			switch arg := args[0].(type) {
			case *Integer:
				port = fmt.Sprintf(":%d", arg.Value)
			case *String:
				if !strings.HasPrefix(arg.Value, ":") {
					port = ":" + arg.Value
				} else {
					port = arg.Value
				}
			default:
				return newError("listen() port must be INTEGER or STRING, got %s", args[0].Kind())
			}

			fmt.Printf("Starting server on %s\n", port)

			// Create a custom handler to match routes with regex
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// Find matching route
				var matchedRoute *routeDef
				var pathParams map[string]string

				for i := range registeredRoutes {
					route := &registeredRoutes[i]
					if strings.ToUpper(r.Method) != route.Method {
						continue
					}

					// Try regex match using precompiled pattern if available.
					if route.PathRegex != nil {
						matches := route.PathRegex.FindStringSubmatch(r.URL.Path)
						if matches != nil {
							matchedRoute = route
							pathParams = make(map[string]string)
							for j, paramName := range route.ParamNames {
								if j+1 < len(matches) {
									pathParams[paramName] = matches[j+1]
								}
							}
							break
						}
					} else if route.Path == r.URL.Path {
						// Exact match
						matchedRoute = route
						pathParams = make(map[string]string)
						break
					}
				}

				if matchedRoute == nil {
					http.NotFound(w, r)
					return
				}

				// Create enhanced Request object with path params
				reqObj := createRequestObjectWithParams(w, r, pathParams)

				// Execute middleware chain + handler
				finalHandler := func(req Object) Object {
					return applyFunction(matchedRoute.Handler, []Object{req})
				}

				// Wrap with route-specific middleware (in reverse order)
				handler := finalHandler
				for i := len(matchedRoute.Middlewares) - 1; i >= 0; i-- {
					mw := matchedRoute.Middlewares[i]
					currentHandler := handler
					handler = func(req Object) Object {
						// Call middleware with (req, next)
						nextFn := &BuiltinFunction{
							Fn: func(args ...Object) Object {
								return currentHandler(req)
							},
						}
						return applyFunction(mw.(*Function), []Object{req, nextFn})
					}
				}

				// Wrap with global middleware
				for i := len(globalMiddlewares) - 1; i >= 0; i-- {
					mw := globalMiddlewares[i]
					currentHandler := handler
					handler = func(req Object) Object {
						nextFn := &BuiltinFunction{
							Fn: func(args ...Object) Object {
								return currentHandler(req)
							},
						}
						return applyFunction(mw.(*Function), []Object{req, nextFn})
					}
				}

				// Execute the full chain
				result := handler(reqObj)

				// Handle response
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

			// Start server (this blocks)
			if err := http.ListenAndServe(port, nil); err != nil {
				return newError("listen error: %s", err)
			}
			return NULL
		},
	}

	// json_response(data, status=200) - helper for JSON responses
	env.store["json_response"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) < 1 || len(args) > 2 {
				return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
			}

			// Convert data to JSON string (simple serialization)
			data := args[0]
			jsonStr := data.Inspect() // Simple version, would need proper JSON encoding

			status := int64(200)
			if len(args) == 2 {
				statusObj, ok := args[1].(*Integer)
				if !ok {
					return newError("second argument to `json_response` must be INTEGER, got %s", args[1].Kind())
				}
				status = statusObj.Value
			}

			return &StructInstance{
				Name: "Response",
				Fields: map[string]Object{
					"status": &Integer{Value: status},
					"body":   &String{Value: jsonStr},
				},
			}
		},
	}

	// Add route() builtin - registers HTTP routes
	env.store["route"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) < 3 || len(args) > 4 {
				return newError("wrong number of arguments. got=%d, want=3 or 4", len(args))
			}

			// Parse method
			methodObj, ok := args[0].(*String)
			if !ok {
				return newError("route() method must be STRING, got %s", args[0].Kind())
			}
			method := strings.ToUpper(methodObj.Value)

			// Parse path
			pathObj, ok := args[1].(*String)
			if !ok {
				return newError("route() path must be STRING, got %s", args[1].Kind())
			}
			path := pathObj.Value

			// Parse handler
			handler, ok := args[2].(*Function)
			if !ok {
				return newError("route() handler must be FUNCTION, got %s", args[2].Kind())
			}

			// Optional middlewares (if 4th arg provided)
			var middlewares []Object
			if len(args) == 4 {
				middlewareArray, ok := args[3].(*Array)
				if !ok {
					return newError("route() middlewares must be ARRAY, got %s", args[3].Kind())
				}
				middlewares = middlewareArray.Elements
			}

			// Register the route
			registeredRoutes = append(registeredRoutes, routeDef{
				Method:      method,
				Path:        path,
				Handler:     handler,
				Middlewares: middlewares,
			})

			return NULL
		},
	}

	// Add listen() builtin - starts HTTP server
	env.store["listen"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}

			// Parse port
			var port string
			switch arg := args[0].(type) {
			case *Integer:
				port = fmt.Sprintf(":%d", arg.Value)
			case *String:
				if !strings.HasPrefix(arg.Value, ":") {
					port = ":" + arg.Value
				} else {
					port = arg.Value
				}
			default:
				return newError("listen() port must be INTEGER or STRING, got %s", args[0].Kind())
			}

			fmt.Printf("Starting HTTP server on %s\n", port)
			fmt.Printf("Registered %d route(s)\n", len(registeredRoutes))

			// Create HTTP server
			mux := http.NewServeMux()

			// Group routes by path to handle multiple methods per path
			routesByPath := make(map[string]map[string]routeDef)
			for _, route := range registeredRoutes {
				if routesByPath[route.Path] == nil {
					routesByPath[route.Path] = make(map[string]routeDef)
				}
				routesByPath[route.Path][route.Method] = route
				fmt.Printf("  %s %s\n", route.Method, route.Path)
			}

			// Register handlers for each path
			for path, methodsMap := range routesByPath {
				// Capture variables
				p := path
				methods := methodsMap

				mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
					// Find the route for this method
					route, found := methods[r.Method]
					if !found {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						return
					}

					// Create request object
					reqObj := createRequestObject(w, r)

					// Execute handler
					result := applyFunction(route.Handler, []Object{reqObj})

					// Handle NULL return (for WebSockets)
					if result == NULL {
						return
					}

					// Handle error
					if err, ok := result.(*ErrorObj); ok {
						http.Error(w, err.Message, http.StatusInternalServerError)
						return
					}

					// Handle response
					if resp, ok := result.(*StructInstance); ok {
						writeHTTPResponse(w, resp)
					} else {
						// Fallback: convert to string
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, result.Inspect())
					}
				})
			}

			// Start server (this blocks)
			if err := http.ListenAndServe(port, mux); err != nil {
				return newError("server error: %s", err)
			}

			return NULL
		},
	}

	// Add response() builtin - simple response helper
	// ... (removed in previous step, but ensuring we are in NewEnvironment)

	// FS Module
	fsModule := make(map[string]Object)

	// fs.read(path)
	fsModule["read"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("fs.read expects 1 argument (path)")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("fs.read argument must be STRING")
			}
			content, err := os.ReadFile(path.Value)
			if err != nil {
				return newError("fs.read failed: %s", err)
			}
			return &String{Value: string(content)}
		},
	}

	// fs.write(path, content)
	fsModule["write"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("fs.write expects 2 arguments (path, content)")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("fs.write path must be STRING")
			}
			content, ok := args[1].(*String)
			if !ok {
				return newError("fs.write content must be STRING")
			}
			err := os.WriteFile(path.Value, []byte(content.Value), 0644)
			if err != nil {
				return newError("fs.write failed: %s", err)
			}
			return TRUE
		},
	}

	// fs.append(path, content)
	fsModule["append"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("fs.append expects 2 arguments (path, content)")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("fs.append path must be STRING")
			}
			content, ok := args[1].(*String)
			if !ok {
				return newError("fs.append content must be STRING")
			}

			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return newError("fs.append failed: %s", err)
			}
			defer f.Close()

			if _, err := f.WriteString(content.Value); err != nil {
				return newError("fs.append failed: %s", err)
			}
			return TRUE
		},
	}

	// fs.exists(path)
	fsModule["exists"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("fs.exists expects 1 argument (path)")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("fs.exists argument must be STRING")
			}
			if _, err := os.Stat(path.Value); os.IsNotExist(err) {
				return FALSE
			}
			return TRUE
		},
	}

	// fs.remove(path)
	fsModule["remove"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("fs.remove expects 1 argument (path)")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("fs.remove argument must be STRING")
			}
			err := os.Remove(path.Value)
			if err != nil {
				return newError("fs.remove failed: %s", err)
			}
			return TRUE
		},
	}

	env.store["fs"] = &StructInstance{Name: "FS", Fields: fsModule}

	// HTTP Module
	httpModule := make(map[string]Object)

	// http.get(url)
	httpModule["get"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("http.get expects 1 argument (url)")
			}
			url, ok := args[0].(*String)
			if !ok {
				return newError("http.get url must be STRING")
			}

			resp, err := http.Get(url.Value)
			if err != nil {
				return newError("http.get failed: %s", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return newError("failed to read response body: %s", err)
			}

			// Return Response object
			fields := make(map[string]Object)
			fields["status"] = &Integer{Value: int64(resp.StatusCode)}
			fields["body"] = &String{Value: string(bodyBytes)}

			// Headers
			headerMap := &Map{Pairs: make(map[Object]Object)}
			for k, v := range resp.Header {
				if len(v) > 0 {
					headerMap.Pairs[&String{Value: k}] = &String{Value: v[0]}
				}
			}
			fields["headers"] = headerMap

			return &StructInstance{Name: "Response", Fields: fields}
		},
	}

	// http.post(url, body, headers)
	httpModule["post"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return newError("http.post expects at least 2 arguments (url, body)")
			}
			url, ok := args[0].(*String)
			if !ok {
				return newError("http.post url must be STRING")
			}

			var bodyReader io.Reader
			if args[1] != NULL {
				bodyStr, ok := args[1].(*String)
				if !ok {
					return newError("http.post body must be STRING or NULL")
				}
				bodyReader = strings.NewReader(bodyStr.Value)
			}

			req, err := http.NewRequest("POST", url.Value, bodyReader)
			if err != nil {
				return newError("failed to create request: %s", err)
			}

			// Optional headers
			if len(args) > 2 {
				headers, ok := args[2].(*Map)
				if ok {
					for k, v := range headers.Pairs {
						keyStr, ok1 := k.(*String)
						valStr, ok2 := v.(*String)
						if ok1 && ok2 {
							req.Header.Set(keyStr.Value, valStr.Value)
						}
					}
				}
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return newError("http.post failed: %s", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return newError("failed to read response body: %s", err)
			}

			// Return Response object
			fields := make(map[string]Object)
			fields["status"] = &Integer{Value: int64(resp.StatusCode)}
			fields["body"] = &String{Value: string(bodyBytes)}

			// Headers
			headerMap := &Map{Pairs: make(map[Object]Object)}
			for k, v := range resp.Header {
				if len(v) > 0 {
					headerMap.Pairs[&String{Value: k}] = &String{Value: v[0]}
				}
			}
			fields["headers"] = headerMap

			return &StructInstance{Name: "Response", Fields: fields}
		},
	}

	env.store["http"] = &StructInstance{Name: "HTTP", Fields: httpModule}

	return env
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	// Create a lightweight child environment that chains to the given outer
	// environment. We intentionally do NOT call NewEnvironment here because
	// that would re-register all builtins, HTTP helpers, etc. on every new
	// scope, which is extremely wasteful and slows down tight loops and
	// function calls.
	env := &Environment{
		store:     make(map[string]Object),
		outer:     outer,
		slotIndex: make(map[string]int),
	}
	return env
}

// NewEnvironmentWithSlots creates a child environment pre-configured with slot-based locals.
// This is used for function calls where we know the exact local variables ahead of time.
func NewEnvironmentWithSlots(outer *Environment, slotCount int, slotNames []string, slotIDs []*InternedString) *Environment {
	env := &Environment{
		store:     make(map[string]Object),
		outer:     outer,
		slots:     make([]Object, slotCount),
		slotIndex: make(map[string]int, slotCount),
		slotNames: slotNames,
	}

	// Build the slotIndex map for O(1) lookups
	for i, name := range slotNames {
		env.slotIndex[name] = i
	}

	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	// Fast path: check slots first if available
	if e.slotIndex != nil {
		if idx, ok := e.slotIndex[name]; ok && idx < len(e.slots) {
			return e.slots[idx], true
		}
	}

	// Standard path: check local store
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	// Fast path: set in slots if available
	if e.slotIndex != nil {
		if idx, ok := e.slotIndex[name]; ok && idx < len(e.slots) {
			e.slots[idx] = val
			return val
		}
	}

	// Standard path: store in map
	e.store[name] = val
	return val
}

func Eval(node ast.Node, env *Environment) Object {
	// PHASE 3 OPTIMIZATION: Fast-path specialization for most common operations
	// These represent ~90% of evaluations in typical programs
	switch node := node.(type) {
	// Hot paths: expressions evaluated in tight loops (prioritize performance)
	case *ast.Identifier:
		// Variable lookups are extremely common - inline fast path
		val, ok := env.Get(node.Value)
		if !ok {
			return newError("identifier not found: %s", node.Value)
		}
		return val

	case *ast.IntegerLiteral:
		// Integer literals are very common - avoid allocation when possible
		return &Integer{Value: node.Value, kind: KindInteger}

	case *ast.InfixExpression:
		// Binary operations are core hot path
		// Try constant folding first (compile-time optimization)
		if isBinaryOperatorConstantFoldable(node) {
			if result := tryConstantFold(node); result != nil {
				return result
			}
		}

		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.IndexExpression:
		// Array/map indexing is common in loops - inline fast path
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		// Inline type switch with minimal overhead
		switch {
		case left.Kind() == KindMap:
			return evalMapIndexExpression(left, index)
		case left.Kind() == KindArray:
			return evalArrayIndexExpression(left, index)
		default:
			return newError("index operator not supported: %s", left.Kind())
		}

	case *ast.IfExpression:
		// Control flow - inline to avoid extra dispatch
		condition := Eval(node.Condition, env)
		if isError(condition) {
			return condition
		}
		if isTruthy(condition) {
			return Eval(node.Consequence, env)
		} else if node.Alternative != nil {
			return Eval(node.Alternative, env)
		} else {
			return NULL
		}

	case *ast.CallExpression:
		// Function calls - very common
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)

	// Less common but still significant
	case *ast.StringLiteral:
		return &String{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.NullLiteral:
		return NULL

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.ArrayLiteral:
		return evalArrayLiteral(node, env)

	case *ast.MapLiteral:
		return evalMapLiteral(node, env)

	// Statement execution
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

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
		// Build slot layout for fast local variable access
		buildFunctionSlotLayout(fn)
		env.Set(node.Name.Value, fn)
		return fn

	case *ast.AssignmentStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return val

	case *ast.WhileStatement:
		return evalWhileStatement(node, env)

	case *ast.ForStatement:
		return evalForStatement(node, env)

	// Less common statements
	case *ast.ModuleStatement:
		return evalModuleStatement(node, env)

	case *ast.ImportStatement:
		return evalImportStatement(node, env)

	case *ast.FromImportStatement:
		return evalFromImportStatement(node, env)

	case *ast.TypeStatement:
		return evalTypeStatement(node, env)

	case *ast.PipelineExpression:
		return evalPipelineExpression(node, env)

	case *ast.SpawnExpression:
		return evalSpawnExpression(node, env)

	case *ast.AwaitExpression:
		return evalAwaitExpression(node, env)

	case *ast.MemberExpression:
		return evalMemberExpression(node, env)

	case *ast.ServiceStatement:
		return evalServiceStatement(node, env)

	case *ast.RouteStatement:
		return evalRouteStatement(node, env)

	case *ast.MiddlewareStatement:
		return evalMiddlewareStatement(node, env)
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
			rt := result.Kind()
			if rt == KindReturnValue || rt == KindError {
				return result
			}
		}
	}
	return result
}

func evalPrefixExpression(operator string, right Object) Object {
	switch operator {
	case "-":
		if right.Kind() != KindInteger {
			return newError("unknown operator: -%s", right.Kind())
		}
		value := right.(*Integer).Value
		return &Integer{Value: -value}
	case "!":
		return evalBangOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Kind())
	}
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
	if left.Kind() == KindInteger && right.Kind() == KindInteger {
		return evalIntegerInfixExpression(operator, left, right)
	}
	if left.Kind() == KindString && right.Kind() == KindString {
		return evalStringInfixExpression(operator, left, right)
	}
	if operator == "==" {
		return evalEqualInfixExpression(left, right)
	}
	if operator == "!=" {
		return evalNotEqualInfixExpression(left, right)
	}
	if left.Kind() != right.Kind() {
		return newError("type mismatch: %s %s %s", left.Kind(), operator, right.Kind())
	}
	return newError("unknown operator: %s %s %s", left.Kind(), operator, right.Kind())
}

func evalIntegerInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	// Fast path: optimize for common operators using direct comparison
	switch operator[0] {
	case '+':
		return NewInteger(leftVal + rightVal)
	case '-':
		return NewInteger(leftVal - rightVal)
	case '*':
		return NewInteger(leftVal * rightVal)
	case '/':
		if rightVal == 0 {
			return newError("division by zero")
		}
		return NewInteger(leftVal / rightVal)
	case '<':
		if len(operator) == 1 {
			return nativeBoolToBooleanObject(leftVal < rightVal)
		}
		// Handle "<=" (length 2)
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case '>':
		if len(operator) == 1 {
			return nativeBoolToBooleanObject(leftVal > rightVal)
		}
		// Handle ">=" (length 2)
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case '=':
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case '!':
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Kind(), operator, right.Kind())
	}
}

func evalStringInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Kind(), operator, right.Kind())
	}
}

func evalEqualInfixExpression(left, right Object) Object {
	if left.Kind() == KindInteger && right.Kind() == KindInteger {
		leftVal := left.(*Integer).Value
		rightVal := right.(*Integer).Value
		return nativeBoolToBooleanObject(leftVal == rightVal)
	}
	if left.Kind() == KindBoolean && right.Kind() == KindBoolean {
		leftVal := left.(*Boolean).Value
		rightVal := right.(*Boolean).Value
		return nativeBoolToBooleanObject(leftVal == rightVal)
	}
	if left.Kind() == KindString && right.Kind() == KindString {
		leftVal := left.(*String).Value
		rightVal := right.(*String).Value
		return nativeBoolToBooleanObject(leftVal == rightVal)
	}
	return FALSE
}

func evalNotEqualInfixExpression(left, right Object) Object {
	if left.Kind() == KindInteger && right.Kind() == KindInteger {
		leftVal := left.(*Integer).Value
		rightVal := right.(*Integer).Value
		return nativeBoolToBooleanObject(leftVal != rightVal)
	}
	if left.Kind() == KindBoolean && right.Kind() == KindBoolean {
		leftVal := left.(*Boolean).Value
		rightVal := right.(*Boolean).Value
		return nativeBoolToBooleanObject(leftVal != rightVal)
	}
	if left.Kind() == KindString && right.Kind() == KindString {
		leftVal := left.(*String).Value
		rightVal := right.(*String).Value
		return nativeBoolToBooleanObject(leftVal != rightVal)
	}
	return TRUE
}

// ============================================================================
// Constant Folding & Visitor Pattern Optimization
// ============================================================================

// isBinaryOperatorConstantFoldable checks if an infix expression can be folded at parse-time
func isBinaryOperatorConstantFoldable(node *ast.InfixExpression) bool {
	// Check if both operands are constants (literals)
	_, leftIsConst := node.Left.(*ast.IntegerLiteral)
	_, rightIsConst := node.Right.(*ast.IntegerLiteral)

	if leftIsConst && rightIsConst {
		return true
	}

	// String concatenation/comparison
	_, leftIsStr := node.Left.(*ast.StringLiteral)
	_, rightIsStr := node.Right.(*ast.StringLiteral)

	if leftIsStr && rightIsStr {
		return node.Operator == "+" || node.Operator == "==" || node.Operator == "!="
	}

	return false
}

// tryConstantFold attempts to evaluate an expression with constant operands at parse time
func tryConstantFold(node *ast.InfixExpression) Object {
	switch left := node.Left.(type) {
	case *ast.IntegerLiteral:
		if right, ok := node.Right.(*ast.IntegerLiteral); ok {
			return foldIntegerOperation(node.Operator, left.Value, right.Value)
		}
	case *ast.StringLiteral:
		if right, ok := node.Right.(*ast.StringLiteral); ok {
			return foldStringOperation(node.Operator, left.Value, right.Value)
		}
	}
	return nil
}

// foldIntegerOperation evaluates integer arithmetic at compile time
func foldIntegerOperation(operator string, left, right int64) Object {
	switch operator {
	case "+":
		return NewInteger(left + right)
	case "-":
		return NewInteger(left - right)
	case "*":
		return NewInteger(left * right)
	case "/":
		if right == 0 {
			return newError("division by zero")
		}
		return NewInteger(left / right)
	case "<":
		return nativeBoolToBooleanObject(left < right)
	case ">":
		return nativeBoolToBooleanObject(left > right)
	case "<=":
		return nativeBoolToBooleanObject(left <= right)
	case ">=":
		return nativeBoolToBooleanObject(left >= right)
	case "==":
		return nativeBoolToBooleanObject(left == right)
	case "!=":
		return nativeBoolToBooleanObject(left != right)
	default:
		return nil
	}
}

// foldStringOperation evaluates string operations at compile time
func foldStringOperation(operator string, left, right string) Object {
	switch operator {
	case "+":
		return &String{Value: left + right}
	case "==":
		return nativeBoolToBooleanObject(left == right)
	case "!=":
		return nativeBoolToBooleanObject(left != right)
	default:
		return nil
	}
}

// Server Implementation

type ServiceContext struct {
	Mux         *http.ServeMux
	Middlewares []Object
	kind        ObjectKind
}

func (sc *ServiceContext) Kind() ObjectKind { return KindNative }
func (sc *ServiceContext) Inspect() string  { return "service context" }

func evalServiceStatement(node *ast.ServiceStatement, env *Environment) Object {
	addr := node.Address.Value
	mux := http.NewServeMux()

	serviceCtx := &ServiceContext{
		Mux:         mux,
		Middlewares: []Object{},
	}

	// Create a new environment for the service block
	serviceEnv := NewEnclosedEnvironment(env)
	serviceEnv.Set("__service_ctx__", serviceCtx)

	// Evaluate body
	res := evalBlockStatement(node.Body, serviceEnv)
	if res != nil && res.Kind() == KindError {
		fmt.Printf("Error evaluating service body: %s\n", res.Inspect())
	}

	// Start server blocking
	fmt.Printf("Starting service %s on %s\n", node.Name.Value, addr)

	// Apply middlewares
	// This is tricky because http.ServeMux doesn't support middleware wrapping easily AFTER routes are added if we want global middleware.
	// But we can wrap the mux itself.
	var handler http.Handler = mux

	// Middlewares are applied in reverse order to wrap the handler
	for i := len(serviceCtx.Middlewares) - 1; i >= 0; i-- {
		mwObj := serviceCtx.Middlewares[i]
		// mwObj should be a function: func(req, ctx, next) -> response
		// We need to wrap 'handler' with this.

		// Capture current handler
		nextHandler := handler

		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create Flowa Request and Context
			reqObj := createRequestObject(w, r)
			ctxObj := createContextObject(r) // TODO: Implement context

			// 'next' function to pass to middleware
			nextFn := &BuiltinFunction{
				Fn: func(args ...Object) Object {
					// This is called by middleware to proceed
					// We need to call nextHandler.ServeHTTP
					// But nextHandler.ServeHTTP expects w and r.
					// We can't easily pause execution here and resume.
					// Flowa is synchronous.
					// If middleware calls await next(req, ctx), it expects a response.

					// For now, let's assume middleware calls next() and we execute the next handler.
					// But we need to capture the response from nextHandler.
					// This requires a ResponseRecorder if we want to inspect it.

					// Simplified: just call nextHandler
					nextHandler.ServeHTTP(w, r)
					return NULL // We don't have the response object easily unless we record it.
				},
			}

			// Silence unused variables for now
			_ = mwObj
			_ = reqObj
			_ = ctxObj
			_ = nextFn

			// Call middleware: mw(req, ctx, next)
			// Wait, if middleware is async?
			// For now assume sync or simple async.

			// We need to call the middleware function.
			// applyFunction(mwObj, []Object{reqObj, ctxObj, nextFn})
			// But we are in Go http handler.
			// We can't easily bridge this without more complex logic.

			// Fallback: Just run the mux for now, ignore middleware implementation details for this step
			// or implement a simple version.
			nextHandler.ServeHTTP(w, r)
		})
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("Service %s error: %s\n", node.Name.Value, err)
	}

	return NULL
}

func evalRouteStatement(node *ast.RouteStatement, env *Environment) Object {
	ctxObj, ok := env.Get("__service_ctx__")
	if !ok {
		return newError("route statement outside service block")
	}
	serviceCtx := ctxObj.(*ServiceContext)

	handlerObj, ok := env.Get(node.Handler.Value)
	if !ok {
		return newError("handler not found: %s", node.Handler.Value)
	}

	// Convert :id to {id}
	path := node.Path.Value
	path = strings.ReplaceAll(path, ":", "{")
	path = strings.ReplaceAll(path, "}", "}") // Just to be safe? No, :id -> {id} is tricky.
	// :id -> {id}
	// Regex would be better but simple replacement might work for simple cases.
	// /users/:id -> /users/{id}
	// /users/:id/posts -> /users/{id}/posts
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	path = strings.Join(parts, "/")

	method := strings.ToUpper(node.Method)

	pattern := method + " " + path
	fmt.Printf("Registering route: %s\n", pattern)

	serviceCtx.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Handling request: %s %s\n", r.Method, r.URL.Path)
		// Create Request object
		reqObj := createRequestObject(w, r)

		// Call handler
		// Handler signature: def handler(req)
		args := []Object{reqObj}

		result := applyFunction(handlerObj, args)

		// Handle result
		if isError(result) {
			fmt.Println("DEBUG: Handler returned error:", result.Inspect())
			http.Error(w, result.Inspect(), http.StatusInternalServerError)
			return
		}

		// If result is NULL, assume response was handled (e.g. WebSocket upgrade)
		if result == NULL {
			return
		}

		// Expect Response object (StructInstance)
		resp, ok := result.(*StructInstance)
		if !ok {
			// If it returns NULL or something else, maybe default 200 OK?
			// User said "def get_user(req) -> Response"
			http.Error(w, "Invalid response from handler", http.StatusInternalServerError)
			return
		}

		statusObj, _ := resp.Fields["status"].(*Integer)
		bodyObj, _ := resp.Fields["body"].(*String)

		status := 200
		if statusObj != nil {
			status = int(statusObj.Value)
		}

		body := ""
		if bodyObj != nil {
			body = bodyObj.Value
		}

		w.WriteHeader(status)
		w.Write([]byte(body))
	})

	return NULL
}

func evalMiddlewareStatement(node *ast.MiddlewareStatement, env *Environment) Object {
	ctxObj, ok := env.Get("__service_ctx__")
	if !ok {
		return newError("use statement outside service block")
	}
	serviceCtx := ctxObj.(*ServiceContext)

	mwObj, ok := env.Get(node.Middleware.Value)
	if !ok {
		return newError("middleware not found: %s", node.Middleware.Value)
	}

	serviceCtx.Middlewares = append(serviceCtx.Middlewares, mwObj)
	return NULL
}

func createRequestObject(w http.ResponseWriter, r *http.Request) Object {
	return createRequestObjectWithParams(w, r, nil)
}

func createRequestObjectWithParams(w http.ResponseWriter, r *http.Request, pathParams map[string]string) Object {
	// Read body
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()
	bodyStr := string(bodyBytes)

	fields := make(map[string]Object)
	fields["method"] = &String{Value: r.Method}
	fields["path"] = &String{Value: r.URL.Path}
	fields["body"] = &String{Value: bodyStr} // Raw body as string

	// IP address
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}
	fields["ip"] = &String{Value: strings.TrimSpace(ip)}

	// Headers (case-insensitive via Get)
	headers := make(map[Object]Object)
	for k, v := range r.Header {
		if len(v) > 0 {
			// Store with lowercase keys for case-insensitivity
			headers[&String{Value: strings.ToLower(k)}] = &String{Value: v[0]}
		}
	}
	fields["headers"] = &Map{Pairs: headers}

	// Cookies
	cookies := make(map[Object]Object)
	for _, cookie := range r.Cookies() {
		cookies[&String{Value: cookie.Name}] = &String{Value: cookie.Value}
	}
	fields["cookies"] = &Map{Pairs: cookies}

	// Query params
	query := make(map[Object]Object)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[&String{Value: k}] = &String{Value: v[0]}
		}
	}
	fields["query"] = &Map{Pairs: query}

	// Form data (for application/x-www-form-urlencoded or multipart/form-data)
	formDataMap := make(map[Object]Object)
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") ||
		strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Re-parse body for form
		r.ParseForm()
		for k, v := range r.Form {
			if len(v) > 0 {
				formDataMap[&String{Value: k}] = &String{Value: v[0]}
			}
		}
	}

	// Path params
	params := make(map[Object]Object)
	for k, v := range pathParams {
		params[&String{Value: k}] = &String{Value: v}
	}
	fields["params"] = &Map{Pairs: params}

	// Callable methods - return functions that can be called
	// req.text() - returns body as string
	fields["text"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			return &String{Value: bodyStr}
		},
	}

	// req.json() - returns parsed JSON as map (simplified - just returns raw for now)
	fields["json"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			// In a real implementation, would parse JSON here
			// For now, just return the raw body
			return &String{Value: bodyStr}
		},
	}

	// req.form() - returns form data as map
	fields["form"] = &BuiltinFunction{
		Fn: func(args ...Object) Object {
			return &Map{Pairs: formDataMap}
		},
	}

	// TODO: req.files - file uploads (would need multipart parsing)
	fields["files"] = &Map{Pairs: make(map[Object]Object)}

	// TODO: req.ctx - Context object (would need context implementation)
	fields["ctx"] = &StructInstance{Name: "Context", Fields: map[string]Object{}}

	// Native objects for WebSocket upgrade
	fields["_native_req"] = &Native{Value: r}
	fields["_native_writer"] = &Native{Value: w}

	return &StructInstance{Name: "Request", Fields: fields}
}

// writeHTTPResponse writes a Flowa Response object to http.ResponseWriter
func writeHTTPResponse(w http.ResponseWriter, resp *StructInstance) {
	// Extract status
	status := 200
	if statusObj, ok := resp.Fields["status"]; ok {
		if s, ok := statusObj.(*Integer); ok {
			status = int(s.Value)
		}
	}

	// Extract headers
	if headersObj, ok := resp.Fields["headers"]; ok {
		if headers, ok := headersObj.(*Map); ok {
			for k, v := range headers.Pairs {
				keyStr, ok1 := k.(*String)
				valStr, ok2 := v.(*String)
				if ok1 && ok2 {
					w.Header().Set(keyStr.Value, valStr.Value)
				}
			}
		}
	}

	// Write status
	w.WriteHeader(status)

	// Extract and write body
	if bodyObj, ok := resp.Fields["body"]; ok {
		if body, ok := bodyObj.(*String); ok {
			fmt.Fprint(w, body.Value)
		} else {
			fmt.Fprint(w, bodyObj.Inspect())
		}
	}
}

func createContextObject(r *http.Request) Object {
	return &StructInstance{Name: "Context", Fields: map[string]Object{}}
}

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	val, ok := env.Get(node.Value)
	if !ok {
		return newError("identifier not found: %s", node.Value)
	}
	return val
}

func evalExpressions(exps []ast.Expression, env *Environment) []Object {
	result := make([]Object, 0, len(exps))
	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

// buildFunctionSlotLayout creates the slot layout for a function based on its parameters.
// This pre-allocates a slot array and builds the index map for O(1) variable lookups.
func buildFunctionSlotLayout(fn *Function) {
	if fn == nil || len(fn.Parameters) == 0 {
		// No parameters, no slots needed for now (can be extended for let bindings)
		fn.SlotCount = 0
		fn.SlotNames = []string{}
		fn.SlotIDs = []*InternedString{}
		return
	}

	slotCount := len(fn.Parameters)
	slotNames := make([]string, slotCount)
	slotIDs := make([]*InternedString, slotCount)

	// Build slot layout from parameters
	for i, param := range fn.Parameters {
		slotNames[i] = param.Value
		slotIDs[i] = InternIdentifier(param.Value)
	}

	fn.SlotCount = slotCount
	fn.SlotNames = slotNames
	fn.SlotIDs = slotIDs
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
		return newError("not a function: %s", fn.Kind())
	}
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	// If the function has pre-computed slot layout, use it for fast local variable access
	if fn.SlotCount > 0 {
		env := NewEnvironmentWithSlots(fn.Env, fn.SlotCount, fn.SlotNames, fn.SlotIDs)

		// Bind parameters directly to slots by index (much faster than map Set)
		for paramIdx := range fn.Parameters {
			if paramIdx < len(args) && paramIdx < len(env.slots) {
				env.slots[paramIdx] = args[paramIdx]
			}
		}
		return env
	}

	// Fallback: use traditional map-based binding (for backward compatibility)
	env := NewEnclosedEnvironment(fn.Env)
	for paramIdx, param := range fn.Parameters {
		if paramIdx < len(args) {
			env.Set(param.Value, args[paramIdx])
		}
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
	pairs := make(map[Object]Object, len(node.Pairs))
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
			if result.Kind() == KindReturnValue || result.Kind() == KindError {
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
		return newError("for-loop value must be ARRAY, got %s", iterable.Kind())
	}

	var result Object = NULL
	// Create a single enclosed environment for the loop and just update the
	// iterator binding on each iteration. This avoids allocating a brand new
	// Environment (and backing map) for every element, which is a significant
	// cost in large loops, while still giving the body its own scope that
	// chains to the outer environment.
	iterEnv := NewEnclosedEnvironment(env)
	for _, elem := range array.Elements {
		iterEnv.Set(fs.Iterator.Value, elem)
		result = Eval(fs.Body, iterEnv)
		if result != nil {
			if result.Kind() == KindReturnValue || result.Kind() == KindError {
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
		return newError("await can only be used on tasks, got %s", val.Kind())
	}
	// Block until the task has completed and return its result.
	return task.Await()
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

func evalImportStatement(node *ast.ImportStatement, env *Environment) Object {
	path := node.Path.Value
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return newError("failed to read import file %s: %s", path, err)
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return newError("parse errors in %s: %s", path, strings.Join(p.Errors(), ", "))
	}

	// Evaluate in new env
	newEnv := NewEnvironment()
	evalProgram(program, newEnv)

	// For `import "path"`, we could bind the module to a variable derived from path
	// But for now, let's just execute it (like include) to keep it simple,
	// or we can implement `import x` later where x is an identifier.
	// Given the string literal, let's treat it as "execute in current scope" (mixin) for now?
	// OR better: execute in new scope and return NULL (just side effects? no that's useless).
	// Let's make it execute in CURRENT scope (mixin) for `import "file"`.
	// This allows splitting code easily.
	evalProgram(program, env)

	return NULL
}

func evalFromImportStatement(node *ast.FromImportStatement, env *Environment) Object {
	path := node.Path.Value
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return newError("failed to read import file %s: %s", path, err)
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return newError("parse errors in %s: %s", path, strings.Join(p.Errors(), ", "))
	}

	// Evaluate in new env
	newEnv := NewEnvironment()
	evalProgram(program, newEnv)

	// Import all symbols
	if node.ImportAll {
		for k, v := range newEnv.store {
			env.Set(k, v)
		}
		return NULL
	}

	// Extract symbols
	for _, ident := range node.Symbols {
		val, ok := newEnv.Get(ident.Value)
		if !ok {
			return newError("symbol %s not found in %s", ident.Value, path)
		}
		env.Set(ident.Value, val)
	}

	return NULL
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
		return newError("type %s does not support member access", obj.Kind())
	}
}

func evalArrayLiteral(node *ast.ArrayLiteral, env *Environment) Object {
	// Pre-allocate array with exact capacity (cache-friendly)
	elements := make([]Object, 0, len(node.Elements))

	for _, el := range node.Elements {
		evaluated := Eval(el, env)
		if isError(evaluated) {
			return evaluated
		}
		elements = append(elements, evaluated)
	}

	return &Array{Elements: elements}
}

func evalMapIndexExpression(mapObj, index Object) Object {
	mapObject := mapObj.(*Map)

	// Optimize for common string keys (fast path)
	if strIdx, ok := index.(*String); ok {
		for k, v := range mapObject.Pairs {
			if strK, ok := k.(*String); ok && strK.Value == strIdx.Value {
				return v
			}
		}
		return NULL
	}

	// Optimize for integer keys
	if intIdx, ok := index.(*Integer); ok {
		for k, v := range mapObject.Pairs {
			if intK, ok := k.(*Integer); ok && intK.Value == intIdx.Value {
				return v
			}
		}
		return NULL
	}

	// Fallback: generic comparison
	for k, v := range mapObject.Pairs {
		if compareKeys(k, index) {
			return v
		}
	}

	return NULL
}

func evalArrayIndexExpression(arrayObj, index Object) Object {
	arrayObject := arrayObj.(*Array)

	// Fast path: expect Integer index (most common case)
	if idx, ok := index.(*Integer); ok {
		// Direct bounds check (no allocation, cache-friendly)
		if idx.Value >= 0 && idx.Value < int64(len(arrayObject.Elements)) {
			return arrayObject.Elements[idx.Value]
		}
		return NULL
	}

	return newError("array index must be INTEGER, got %s", index.Kind())
}

func compareKeys(key1, key2 Object) bool {
	// Handle String keys specially
	if key1.Kind() == KindString && key2.Kind() == KindString {
		return key1.(*String).Value == key2.(*String).Value
	}
	// For other types, use Inspect() comparison (simple but works)
	return key1.Inspect() == key2.Inspect()
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
		return obj.Kind() == KindError
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
