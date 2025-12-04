package vm

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flowa/pkg/ast"
	"flowa/pkg/compiler"
	"flowa/pkg/eval"
	"flowa/pkg/opcode"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const StackSize = 256   // Reduced from 2048 for better cache locality
const GlobalsSize = 256 // Reduced from 65536 - 256 globals should be sufficient
const MaxFrames = 64    // Reduced from 1024 - supports 64 levels of function calls

// Frame represents a call frame
type Frame struct {
	fn          *eval.Function
	ip          int // instruction pointer for this frame
	basePointer int // base pointer for local variables in this frame
}

func NewFrame(fn *eval.Function, basePointer int) *Frame {
	return &Frame{
		fn:          fn,
		ip:          -1,
		basePointer: basePointer,
	}
}

type VM struct {
	constants []eval.Object
	globals   []eval.Object

	stack []eval.Object
	sp    int // Always points to the next value. Top of stack is stack[sp-1]

	builtins []eval.Object

	frames      []*Frame
	framesIndex int

	// HTTP server state
	httpRoutes map[string]map[string]*eval.Function // method -> path -> handler function

	// Import cache
	importCache map[string]eval.Object // path -> loaded module
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &eval.Function{Body: &ast.BlockStatement{}, Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	// Initialize built-ins - must match compiler indices
	builtins := make([]eval.Object, 18)

	// 0: print
	builtins[0] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				if arg == nil || arg == eval.NULL {
					fmt.Print("null")
				} else {
					fmt.Print(arg.Inspect())
				}
			}
			fmt.Println()
			return eval.NULL
		},
	}

	// 1: len
	builtins[1] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			switch arg := args[0].(type) {
			case *eval.String:
				return &eval.Integer{Value: int64(len(arg.Value))}
			case *eval.Array:
				return &eval.Integer{Value: int64(len(arg.Elements))}
			default:
				return &eval.ErrorObj{Message: fmt.Sprintf("argument to `len` not supported, got %s", args[0].Kind())}
			}
		},
	}

	// 2: time module
	timeEnv := eval.NewEnvironment()
	timeEnv.Set("now_ms", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			return &eval.Integer{Value: time.Now().UnixMilli()}
		},
	})
	timeEnv.Set("since_ms", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "time.since_ms expects 1 argument (start_time_ms)"}
			}
			start, ok := args[0].(*eval.Integer)
			if !ok {
				return &eval.ErrorObj{Message: "time.since_ms argument must be INTEGER"}
			}
			now := time.Now().UnixMilli()
			return &eval.Integer{Value: now - start.Value}
		},
	})
	timeEnv.Set("since_s", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "wrong number of arguments: expected 1 or 2"}
			}
			start, ok := args[0].(*eval.Integer)
			if !ok {
				return &eval.ErrorObj{Message: "first argument must be integer"}
			}

			// Default precision is 3
			precision := 3
			if len(args) == 2 {
				precisionArg, ok := args[1].(*eval.Integer)
				if !ok {
					return &eval.ErrorObj{Message: "second argument (precision) must be integer"}
				}
				precision = int(precisionArg.Value)
			}

			now := time.Now().UnixMilli()
			diff := now - start.Value
			formatStr := fmt.Sprintf("%%.%df", precision)
			return &eval.String{Value: fmt.Sprintf(formatStr, float64(diff)/1000.0)}
		},
	})
	builtins[2] = &eval.Module{Name: "time", Env: timeEnv}

	// 3: auth module
	authEnv := eval.NewEnvironment()
	authEnv.Set("hash_password", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "auth.hash_password expects 1 argument"}
			}
			pass, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "argument must be STRING"}
			}
			// Use real bcrypt hashing
			hash, err := eval.HashPassword(pass.Value)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("failed to hash password: %s", err)}
			}
			return &eval.String{Value: hash}
		},
	})
	authEnv.Set("verify_password", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "auth.verify_password expects 2 arguments (hash, password)"}
			}
			hash, ok1 := args[0].(*eval.String)
			pass, ok2 := args[1].(*eval.String)
			if !ok1 || !ok2 {
				return &eval.ErrorObj{Message: "arguments must be STRING"}
			}
			// Use real bcrypt verification
			if eval.VerifyPassword(hash.Value, pass.Value) {
				return eval.TRUE
			}
			return eval.FALSE
		},
	})
	builtins[3] = &eval.Module{Name: "auth", Env: authEnv}

	// 4: json module
	jsonEnv := eval.NewEnvironment()
	jsonEnv.Set("encode", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "json.encode expects 1 argument"}
			}
			native := eval.FlowaToNative(args[0])
			bytes, err := json.Marshal(native)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("json encode error: %s", err)}
			}
			return &eval.String{Value: string(bytes)}
		},
	})
	jsonEnv.Set("decode", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "json.decode expects 1 argument"}
			}
			strArg, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "json.decode argument must be STRING"}
			}
			var native interface{}
			err := json.Unmarshal([]byte(strArg.Value), &native)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("json decode error: %s", err)}
			}
			return eval.NativeToFlowa(native)
		},
	})
	builtins[4] = &eval.Module{Name: "json", Env: jsonEnv}

	// 5: http module (client)
	httpEnv := eval.NewEnvironment()
	httpEnv.Set("get", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "http.get expects 1 argument (url)"}
			}
			url, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "http.get url must be STRING"}
			}

			resp, err := http.Get(url.Value)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("http.get failed: %s", err)}
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("failed to read response body: %s", err)}
			}

			responseMap := make(map[eval.Object]eval.Object)
			responseMap[&eval.String{Value: "status"}] = &eval.Integer{Value: int64(resp.StatusCode)}
			responseMap[&eval.String{Value: "body"}] = &eval.String{Value: string(body)}
			return &eval.Map{Pairs: responseMap}
		},
	})
	httpEnv.Set("post", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 2 {
				return &eval.ErrorObj{Message: "http.post expects at least 2 arguments (url, body)"}
			}
			url, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "http.post url must be STRING"}
			}

			bodyStr, ok := args[1].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "http.post body must be STRING"}
			}

			resp, err := http.Post(url.Value, "application/json", bytes.NewBuffer([]byte(bodyStr.Value)))
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("http.post failed: %s", err)}
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("failed to read response body: %s", err)}
			}

			responseMap := make(map[eval.Object]eval.Object)
			responseMap[&eval.String{Value: "status"}] = &eval.Integer{Value: int64(resp.StatusCode)}
			responseMap[&eval.String{Value: "body"}] = &eval.String{Value: string(respBody)}
			return &eval.Map{Pairs: responseMap}
		},
	})
	// http.use(middleware_fn) - register global middleware (placeholder for VM)
	httpEnv.Set("use", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "http.use expects 1 argument (middleware function)"}
			}
			// In VM mode, http.use is acknowledged but middleware execution
			// happens in the bytecode interpreter context
			// For now, just return success
			return eval.NULL
		},
	})
	builtins[5] = &eval.Module{Name: "http", Env: httpEnv}

	// 6: fs module
	fsEnv := eval.NewEnvironment()
	fsEnv.Set("read", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "fs.read expects 1 argument (path)"}
			}
			path, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.read path must be STRING"}
			}
			content, err := os.ReadFile(path.Value)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("fs.read failed: %s", err)}
			}
			return &eval.String{Value: string(content)}
		},
	})
	fsEnv.Set("write", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "fs.write expects 2 arguments (path, content)"}
			}
			path, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.write path must be STRING"}
			}
			content, ok := args[1].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.write content must be STRING"}
			}
			err := os.WriteFile(path.Value, []byte(content.Value), 0644)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("fs.write failed: %s", err)}
			}
			return eval.TRUE
		},
	})
	fsEnv.Set("exists", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "fs.exists expects 1 argument"}
			}
			path, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.exists path must be STRING"}
			}
			if _, err := os.Stat(path.Value); err == nil {
				return eval.TRUE
			}
			return eval.FALSE
		},
	})
	fsEnv.Set("remove", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "fs.remove expects 1 argument"}
			}
			path, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.remove path must be STRING"}
			}
			err := os.Remove(path.Value)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("fs.remove failed: %s", err)}
			}
			return eval.TRUE
		},
	})
	fsEnv.Set("append", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "fs.append expects 2 arguments"}
			}
			path, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.append path must be STRING"}
			}
			content, ok := args[1].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "fs.append content must be STRING"}
			}
			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("fs.append failed: %s", err)}
			}
			defer f.Close()
			if _, err := f.WriteString(content.Value); err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("fs.append write failed: %s", err)}
			}
			return eval.TRUE
		},
	})
	builtins[6] = &eval.Module{Name: "fs", Env: fsEnv}

	// 7: response module
	responseEnv := eval.NewEnvironment()
	responseEnv.Set("json", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "response.json expects 1-2 arguments (data, status_code)"}
			}
			// Return a response map
			response := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
			response.Pairs[&eval.String{Value: "type"}] = &eval.String{Value: "json"}
			response.Pairs[&eval.String{Value: "data"}] = args[0]
			status := int64(200)
			if len(args) == 2 {
				if statusArg, ok := args[1].(*eval.Integer); ok {
					status = statusArg.Value
				}
			}
			response.Pairs[&eval.String{Value: "status"}] = &eval.Integer{Value: status}
			return response
		},
	})
	responseEnv.Set("text", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "response.text expects 1-2 arguments (text, status_code)"}
			}
			response := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
			response.Pairs[&eval.String{Value: "type"}] = &eval.String{Value: "text"}
			response.Pairs[&eval.String{Value: "data"}] = args[0]
			status := int64(200)
			if len(args) == 2 {
				if statusArg, ok := args[1].(*eval.Integer); ok {
					status = statusArg.Value
				}
			}
			response.Pairs[&eval.String{Value: "status"}] = &eval.Integer{Value: status}
			return response
		},
	})
	responseEnv.Set("html", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "response.html expects 1-2 arguments (html, status_code)"}
			}
			response := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
			response.Pairs[&eval.String{Value: "type"}] = &eval.String{Value: "html"}
			response.Pairs[&eval.String{Value: "data"}] = args[0]
			status := int64(200)
			if len(args) == 2 {
				if statusArg, ok := args[1].(*eval.Integer); ok {
					status = statusArg.Value
				}
			}
			response.Pairs[&eval.String{Value: "status"}] = &eval.Integer{Value: status}
			return response
		},
	})
	responseEnv.Set("redirect", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "response.redirect expects 1-2 arguments (url, status_code)"}
			}
			response := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
			response.Pairs[&eval.String{Value: "type"}] = &eval.String{Value: "redirect"}
			response.Pairs[&eval.String{Value: "url"}] = args[0]
			status := int64(302)
			if len(args) == 2 {
				if statusArg, ok := args[1].(*eval.Integer); ok {
					status = statusArg.Value
				}
			}
			response.Pairs[&eval.String{Value: "status"}] = &eval.Integer{Value: status}
			return response
		},
	})
	builtins[7] = &eval.Module{Name: "response", Env: responseEnv}

	// 8: websocket module
	// NOTE: WebSocket functions in VM mode return simulated responses because
	// real WebSocket connections require HTTP context (http.ResponseWriter) which
	// is only available when running an actual HTTP server, not in standalone VM execution.
	// This is expected behavior - for real WebSocket functionality, use with http server.
	websocketEnv := eval.NewEnvironment()

	// websocket.upgrade(req) - Upgrade HTTP connection to WebSocket
	websocketEnv.Set("upgrade", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 {
				return &eval.ErrorObj{Message: "websocket.upgrade expects request object"}
			}

			// For now, return a simulated connection object
			// Real upgrade happens in HTTP handler context
			conn := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
			conn.Pairs[&eval.String{Value: "connected"}] = &eval.Boolean{Value: true}
			conn.Pairs[&eval.String{Value: "id"}] = &eval.Integer{Value: 1}
			conn.Pairs[&eval.String{Value: "type"}] = &eval.String{Value: "websocket"}
			return conn
		},
	})

	// websocket.send(conn, message) - Send message over WebSocket
	websocketEnv.Set("send", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 2 {
				return &eval.ErrorObj{Message: "websocket.send expects (conn, message)"}
			}

			// Check connection type
			_, ok := args[0].(*eval.Map)
			if !ok {
				return &eval.ErrorObj{Message: "websocket.send: first argument must be connection"}
			}

			// Get message to send
			var message string
			switch msg := args[1].(type) {
			case *eval.String:
				message = msg.Value
			case *eval.Integer:
				message = fmt.Sprintf("%d", msg.Value)
			case *eval.Boolean:
				message = fmt.Sprintf("%t", msg.Value)
			default:
				message = msg.Inspect()
			}

			// In real implementation with http.ResponseWriter, this would send via WebSocket
			// For now, log and return success
			fmt.Printf("[WebSocket] Sending: %s\n", message)
			return &eval.Boolean{Value: true}
		},
	})

	// websocket.receive(conn) / websocket.read(conn) - Receive message
	receiveFunc := &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 {
				return &eval.ErrorObj{Message: "websocket.receive expects connection"}
			}

			// Simulate receiving a message
			// In real implementation, this would block and read from WebSocket
			return &eval.String{Value: "Simulated incoming message"}
		},
	}
	websocketEnv.Set("receive", receiveFunc)
	websocketEnv.Set("read", receiveFunc) // Alias

	// websocket.close(conn) - Close WebSocket connection
	websocketEnv.Set("close", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 {
				return &eval.ErrorObj{Message: "websocket.close expects connection"}
			}

			// Simulate closing connection
			fmt.Println("[WebSocket] Connection closed")
			return &eval.Boolean{Value: true}
		},
	})

	// websocket.broadcast(message) - Broadcast to all connected clients
	websocketEnv.Set("broadcast", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 {
				return &eval.ErrorObj{Message: "websocket.broadcast expects message"}
			}

			message, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "websocket.broadcast: message must be string"}
			}

			fmt.Printf("[WebSocket] Broadcasting: %s\n", message.Value)
			return &eval.Integer{Value: 0} // Number of clients reached
		},
	})

	builtins[8] = &eval.Module{Name: "websocket", Env: websocketEnv}

	// 9: mail module
	mailEnv := eval.NewEnvironment()
	mailEnv.Set("send", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "mail.send expects 1 argument (mail_data map)"}
			}
			mailMap, ok := args[0].(*eval.Map)
			if !ok {
				return &eval.ErrorObj{Message: "mail.send argument must be MAP"}
			}

			// Extract mail data
			var to, subject, body, html string
			for k, v := range mailMap.Pairs {
				keyStr, ok := k.(*eval.String)
				if !ok {
					continue
				}
				valStr, ok := v.(*eval.String)
				if !ok {
					continue
				}
				switch keyStr.Value {
				case "to":
					to = valStr.Value
				case "from":
					// from is optional, ignore for now
				case "subject":
					subject = valStr.Value
				case "body":
					body = valStr.Value
				case "html":
					html = valStr.Value
				}
			}

			if to == "" || subject == "" {
				return &eval.ErrorObj{Message: "mail.send requires 'to' and 'subject' fields"}
			}

			// Get SMTP settings from environment
			smtpHost := os.Getenv("SMTP_HOST")
			smtpPort := os.Getenv("SMTP_PORT")
			smtpUser := os.Getenv("SMTP_USER")
			smtpPass := os.Getenv("SMTP_PASS")

			if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
				msg := fmt.Sprintf("SMTP not configured. Set SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS env vars. Would send to: %s", to)
				return &eval.String{Value: msg}
			}

			// Actually send the email using SMTP
			err := sendSMTPEmail(smtpHost, smtpPort, smtpUser, smtpPass, smtpUser, to, subject, body, html)
			if err != nil {
				return &eval.ErrorObj{Message: fmt.Sprintf("Failed to send email: %v", err)}
			}

			// Return success message
			msg := fmt.Sprintf("✅ Email sent successfully to %s", to)
			return &eval.String{Value: msg}
		},
	})
	mailEnv.Set("send_template", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "mail.send_template expects 2 arguments (template, data)"}
			}
			template, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "mail.send_template template must be STRING"}
			}
			dataMap, ok := args[1].(*eval.Map)
			if !ok {
				return &eval.ErrorObj{Message: "mail.send_template data must be MAP"}
			}

			// Simple template replacement
			body := template.Value
			for k, v := range dataMap.Pairs {
				keyStr, ok := k.(*eval.String)
				if !ok {
					continue
				}
				valStr, ok := v.(*eval.String)
				if !ok {
					continue
				}
				placeholder := "{{" + keyStr.Value + "}}"
				body = strings.Replace(body, placeholder, valStr.Value, -1)
			}

			return &eval.String{Value: "Template email prepared: " + body[:50] + "..."}
		},
	})
	builtins[9] = &eval.Module{Name: "mail", Env: mailEnv}

	// 10: jwt module
	jwtEnv := eval.NewEnvironment()
	jwtEnv.Set("sign", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 3 {
				return &eval.ErrorObj{Message: "jwt.sign expects 3 arguments (payload, secret, expiry)"}
			}
			payload, ok := args[0].(*eval.Map)
			if !ok {
				return &eval.ErrorObj{Message: "jwt.sign payload must be MAP"}
			}
			secret, ok := args[1].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "jwt.sign secret must be STRING"}
			}

			// Simplified JWT (HS256)
			header := `{"alg":"HS256","typ":"JWT"}`

			// Convert payload to JSON
			payloadJSON := "{"
			first := true
			for k, v := range payload.Pairs {
				if !first {
					payloadJSON += ","
				}
				first = false
				keyStr := k.(*eval.String)
				payloadJSON += `"` + keyStr.Value + `":`
				if valStr, ok := v.(*eval.String); ok {
					payloadJSON += `"` + valStr.Value + `"`
				} else if valInt, ok := v.(*eval.Integer); ok {
					payloadJSON += fmt.Sprintf("%d", valInt.Value)
				}
			}
			payloadJSON += "}"

			encodeB64 := func(s string) string {
				return strings.TrimRight(base64.URLEncoding.EncodeToString([]byte(s)), "=")
			}

			headerB64 := encodeB64(header)
			payloadB64 := encodeB64(payloadJSON)

			h := hmac.New(sha256.New, []byte(secret.Value))
			h.Write([]byte(headerB64 + "." + payloadB64))
			signatureB64 := encodeB64(string(h.Sum(nil)))

			token := headerB64 + "." + payloadB64 + "." + signatureB64
			return &eval.String{Value: token}
		},
	})
	jwtEnv.Set("verify", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "jwt.verify expects 2 arguments (token, secret)"}
			}
			token, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "jwt.verify token must be STRING"}
			}
			secret, ok := args[1].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "jwt.verify secret must be STRING"}
			}

			parts := strings.Split(token.Value, ".")
			if len(parts) != 3 {
				return eval.NULL
			}

			h := hmac.New(sha256.New, []byte(secret.Value))
			h.Write([]byte(parts[0] + "." + parts[1]))
			expectedSig := strings.TrimRight(base64.URLEncoding.EncodeToString(h.Sum(nil)), "=")

			if parts[2] != expectedSig {
				return eval.NULL
			}

			// Decode payload
			payloadB64 := parts[1]
			switch len(payloadB64) % 4 {
			case 2:
				payloadB64 += "=="
			case 3:
				payloadB64 += "="
			}

			payloadBytes, err := base64.URLEncoding.DecodeString(payloadB64)
			if err != nil {
				return eval.NULL
			}

			var native interface{}
			json.Unmarshal(payloadBytes, &native)
			return eval.NativeToFlowa(native)
		},
	})
	builtins[10] = &eval.Module{Name: "jwt", Env: jwtEnv}

	// 11: config module
	configEnv := eval.NewEnvironment()
	configEnv.Set("env", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) < 1 || len(args) > 2 {
				return &eval.ErrorObj{Message: "config.env expects 1 or 2 arguments"}
			}
			key, ok := args[0].(*eval.String)
			if !ok {
				return &eval.ErrorObj{Message: "config.env key must be STRING"}
			}

			value := os.Getenv(key.Value)
			if value == "" && len(args) == 2 {
				return args[1]
			}
			return &eval.String{Value: value}
		},
	})
	builtins[11] = &eval.Module{Name: "config", Env: configEnv}

	// 12-14: Performance built-ins
	builtins[12] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "fast_sum_to expects 1 INTEGER argument"}
			}
			nInt, ok := args[0].(*eval.Integer)
			if !ok {
				return &eval.ErrorObj{Message: "fast_sum_to argument must be INTEGER"}
			}
			n := nInt.Value
			if n < 0 {
				return &eval.ErrorObj{Message: "fast_sum_to argument must be non-negative"}
			}

			var sum int64
			for i := int64(0); i < n; i++ {
				sum += i
			}
			return eval.NewInteger(sum)
		},
	}

	builtins[13] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "fast_sum_range expects 2 INTEGER arguments"}
			}
			startInt, ok1 := args[0].(*eval.Integer)
			endInt, ok2 := args[1].(*eval.Integer)
			if !ok1 || !ok2 {
				return &eval.ErrorObj{Message: "fast_sum_range arguments must be INTEGER"}
			}
			start := startInt.Value
			end := endInt.Value
			if end < start {
				return &eval.ErrorObj{Message: "fast_sum_range end must be >= start"}
			}

			var sum int64
			for i := start; i < end; i++ {
				sum += i
			}
			return eval.NewInteger(sum)
		},
	}

	// 14: fast_repeat(n, fn)
	builtins[14] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			if len(args) != 2 {
				return &eval.ErrorObj{Message: "fast_repeat expects 2 arguments (n, fn)"}
			}
			n, ok := args[0].(*eval.Integer)
			if !ok {
				return &eval.ErrorObj{Message: "fast_repeat n must be INTEGER"}
			}
			fn, ok := args[1].(*eval.BuiltinFunction)
			if !ok {
				return &eval.ErrorObj{Message: "fast_repeat fn must be built-in function (user functions not supported yet)"}
			}
			for i := int64(0); i < n.Value; i++ {
				fn.Fn(&eval.Integer{Value: i})
			}
			return eval.NULL
		},
	}

	// 15: route(method, path, handler) - Store route for later
	// Note: This needs access to VM to store routes, but builtins don't have VM reference
	// Workaround: We'll handle route() as a special case in the VM
	builtins[15] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			// This is a placeholder - actual route storage happens in VM
			if len(args) != 3 {
				return &eval.ErrorObj{Message: "route expects 3 arguments (method, path, handler)"}
			}

			method, ok1 := args[0].(*eval.String)
			path, ok2 := args[1].(*eval.String)
			if !ok1 || !ok2 {
				return &eval.ErrorObj{Message: "route: method and path must be strings"}
			}

			// Return success message
			msg := fmt.Sprintf("Route registered: %s %s", method.Value, path.Value)
			return &eval.String{Value: msg}
		},
	}

	// 16: listen(port) - Start HTTP server
	builtins[16] = &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			// This is a placeholder - actual server start happens in VM
			if len(args) != 1 {
				return &eval.ErrorObj{Message: "listen expects 1 argument (port)"}
			}

			port, ok := args[0].(*eval.Integer)
			if !ok {
				return &eval.ErrorObj{Message: "listen: port must be an integer"}
			}

			// Return success message (server doesn't actually start in test mode)
			msg := fmt.Sprintf("Server ready on port %d (test mode - not actually listening)", port.Value)
			return &eval.String{Value: msg}
		},
	}

	// 17: middleware module
	middlewareEnv := eval.NewEnvironment()

	// middleware.logger() - Returns logging middleware function
	middlewareEnv.Set("logger", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			// Returns a middleware function: func(req, next)
			return &eval.BuiltinFunction{
				Fn: func(mwArgs ...eval.Object) eval.Object {
					if len(mwArgs) != 2 {
						return &eval.ErrorObj{Message: "middleware must be called with (req, next)"}
					}
					req := mwArgs[0]
					next := mwArgs[1]

					// Log request
					method := "UNKNOWN"
					path := "UNKNOWN"
					if reqMap, ok := req.(*eval.Map); ok {
						if m, ok := reqMap.Pairs[&eval.String{Value: "method"}].(*eval.String); ok {
							method = m.Value
						}
						if p, ok := reqMap.Pairs[&eval.String{Value: "path"}].(*eval.String); ok {
							path = p.Value
						}
					}
					fmt.Printf("[LOG] %s %s\n", method, path)

					// Call next
					if nextBuiltin, ok := next.(*eval.BuiltinFunction); ok {
						return nextBuiltin.Fn()
					}
					if _, ok := next.(*eval.Function); ok {
						// Would need to apply function but we don't have context here
						// Return NULL as placeholder
						return eval.NULL
					}
					return eval.NULL
				},
			}
		},
	})

	// middleware.cors() - Returns CORS middleware function
	middlewareEnv.Set("cors", &eval.BuiltinFunction{
		Fn: func(args ...eval.Object) eval.Object {
			// Returns a middleware function: func(req, next)
			return &eval.BuiltinFunction{
				Fn: func(mwArgs ...eval.Object) eval.Object {
					if len(mwArgs) != 2 {
						return &eval.ErrorObj{Message: "middleware must be called with (req, next)"}
					}
					next := mwArgs[1]

					// Call next to get response
					var result eval.Object
					if nextBuiltin, ok := next.(*eval.BuiltinFunction); ok {
						result = nextBuiltin.Fn()
					} else if _, ok := next.(*eval.Function); ok {
						// Would need to apply function
						result = eval.NULL
					} else {
						return eval.NULL
					}

					// Add CORS headers to response (if it's a map)
					if respMap, ok := result.(*eval.Map); ok {
						if headers, ok := respMap.Pairs[&eval.String{Value: "headers"}].(*eval.Map); ok {
							headers.Pairs[&eval.String{Value: "Access-Control-Allow-Origin"}] = &eval.String{Value: "*"}
							headers.Pairs[&eval.String{Value: "Access-Control-Allow-Methods"}] = &eval.String{Value: "GET, POST, PUT, DELETE, OPTIONS"}
							headers.Pairs[&eval.String{Value: "Access-Control-Allow-Headers"}] = &eval.String{Value: "Content-Type, Authorization"}
						}
					}

					return result
				},
			}
		},
	})

	builtins[17] = &eval.Module{Name: "middleware", Env: middlewareEnv}

	return &VM{
		constants: bytecode.Constants,
		globals:   make([]eval.Object, GlobalsSize),

		stack: make([]eval.Object, StackSize),
		sp:    bytecode.MainNumLocals,

		builtins: builtins,

		frames:      frames,
		framesIndex: 1,

		httpRoutes:  make(map[string]map[string]*eval.Function),
		importCache: make(map[string]eval.Object),
	}
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

func (vm *VM) StackTop() eval.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) Run() error {
	// Cache frame pointer to avoid repeated function calls
	frame := vm.currentFrame()
	ip := frame.ip
	ins := frame.fn.Instructions

	// Cache frequently accessed fields
	stack := vm.stack
	sp := vm.sp
	constants := vm.constants

	// Main execution loop
	for ip < len(ins)-1 {
		ip++
		op := opcode.Opcode(ins[ip])

		switch op {
		case opcode.OpConstant:
			constIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2
			stack[sp] = constants[constIndex]
			sp++
			// fmt.Printf("OpConstant: sp=%d\n", sp)

		case opcode.OpGetBuiltin:
			builtinIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1
			stack[sp] = vm.builtins[builtinIndex]
			sp++

		case opcode.OpIncLocal:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1

			obj := stack[frame.basePointer+localIndex]
			if intObj, ok := obj.(*eval.Integer); ok {
				// Create new integer to preserve immutability
				// TODO: Use object pooling here for further optimization
				stack[frame.basePointer+localIndex] = eval.NewInteger(intObj.Value + 1)
			} else {
				return fmt.Errorf("operand to OpIncLocal must be an integer, got %s", obj.Kind())
			}

		case opcode.OpAddLocal:
			destIndex := int(opcode.ReadUint8(ins[ip+1:]))
			sourceIndex := int(opcode.ReadUint8(ins[ip+2:]))
			ip += 2

			dest := stack[frame.basePointer+destIndex]
			source := stack[frame.basePointer+sourceIndex]

			// Check for nil
			if dest == nil || source == nil {
				return fmt.Errorf("OpAddLocal: nil operand")
			}

			// Only works with integers
			destInt, ok1 := dest.(*eval.Integer)
			sourceInt, ok2 := source.(*eval.Integer)

			if !ok1 || !ok2 {
				return fmt.Errorf("OpAddLocal requires integer operands, got %s and %s", dest.Kind(), source.Kind())
			}

			// Fast path: integer addition
			result := destInt.Value + sourceInt.Value
			stack[frame.basePointer+destIndex] = eval.NewInteger(result)

		case opcode.OpArray:
			numElements := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2

			array := &eval.Array{Elements: make([]eval.Object, numElements)}
			for i := numElements - 1; i >= 0; i-- {
				sp--
				array.Elements[i] = stack[sp]
			}

			stack[sp] = array
			sp++

		case opcode.OpHash:
			numElements := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2

			hash := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
			for i := 0; i < numElements; i += 2 {
				sp--
				value := stack[sp]
				sp--
				key := stack[sp]
				hash.Pairs[key] = value
			}

			stack[sp] = hash
			sp++

		case opcode.OpIndex:
			index := stack[sp-1]
			left := stack[sp-2]
			sp -= 2

			// Sync and delegate to helper
			vm.sp = sp
			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}
			// Sync local sp back
			sp = vm.sp

		case opcode.OpAdd:
			right := stack[sp-1]
			left := stack[sp-2]
			sp -= 2

			leftType := left.Kind()
			rightType := right.Kind()

			// String concatenation
			if leftType == eval.KindString || rightType == eval.KindString {
				var leftVal, rightVal string

				if leftStr, ok := left.(*eval.String); ok {
					leftVal = leftStr.Value
				} else if leftInt, ok := left.(*eval.Integer); ok {
					leftVal = fmt.Sprintf("%d", leftInt.Value)
				} else if leftFloat, ok := left.(*eval.Float); ok {
					leftVal = fmt.Sprintf("%g", leftFloat.Value)
				} else {
					return fmt.Errorf("cannot concatenate %s with string", leftType)
				}

				if rightStr, ok := right.(*eval.String); ok {
					rightVal = rightStr.Value
				} else if rightInt, ok := right.(*eval.Integer); ok {
					rightVal = fmt.Sprintf("%d", rightInt.Value)
				} else if rightFloat, ok := right.(*eval.Float); ok {
					rightVal = fmt.Sprintf("%g", rightFloat.Value)
				} else {
					return fmt.Errorf("cannot concatenate string with %s", rightType)
				}

				stack[sp] = &eval.String{Value: leftVal + rightVal}
				sp++
			} else if leftType == eval.KindInteger && rightType == eval.KindInteger {
				// Fast path: integer addition
				leftInt := left.(*eval.Integer)
				rightInt := right.(*eval.Integer)
				stack[sp] = eval.NewInteger(leftInt.Value + rightInt.Value)
				sp++
			} else if leftType == eval.KindFloat || rightType == eval.KindFloat {
				// Float addition with type promotion
				var leftVal, rightVal float64

				if leftType == eval.KindFloat {
					leftVal = left.(*eval.Float).Value
				} else {
					leftVal = float64(left.(*eval.Integer).Value)
				}

				if rightType == eval.KindFloat {
					rightVal = right.(*eval.Float).Value
				} else {
					rightVal = float64(right.(*eval.Integer).Value)
				}

				stack[sp] = &eval.Float{Value: leftVal + rightVal}
				sp++
			} else {
				return fmt.Errorf("unsupported types for addition: %s + %s", leftType, rightType)
			}

		case opcode.OpSub, opcode.OpMul, opcode.OpDiv:
			right := stack[sp-1]
			left := stack[sp-2]
			sp -= 2

			leftType := left.Kind()
			rightType := right.Kind()

			// Integer fast path
			if leftType == eval.KindInteger && rightType == eval.KindInteger {
				leftInt := left.(*eval.Integer)
				rightInt := right.(*eval.Integer)

				var result int64
				switch op {
				case opcode.OpSub:
					result = leftInt.Value - rightInt.Value
				case opcode.OpMul:
					result = leftInt.Value * rightInt.Value
				case opcode.OpDiv:
					result = leftInt.Value / rightInt.Value
				}

				stack[sp] = eval.NewInteger(result)
				sp++
			} else if leftType == eval.KindFloat || rightType == eval.KindFloat {
				// Float operations with type promotion
				var leftVal, rightVal float64

				if leftType == eval.KindFloat {
					leftVal = left.(*eval.Float).Value
				} else if leftType == eval.KindInteger {
					leftVal = float64(left.(*eval.Integer).Value)
				} else {
					return fmt.Errorf("operand must be number, got %s", leftType)
				}

				if rightType == eval.KindFloat {
					rightVal = right.(*eval.Float).Value
				} else if rightType == eval.KindInteger {
					rightVal = float64(right.(*eval.Integer).Value)
				} else {
					return fmt.Errorf("operand must be number, got %s", rightType)
				}

				var result float64
				switch op {
				case opcode.OpSub:
					result = leftVal - rightVal
				case opcode.OpMul:
					result = leftVal * rightVal
				case opcode.OpDiv:
					result = leftVal / rightVal
				}

				stack[sp] = &eval.Float{Value: result}
				sp++
			} else {
				return fmt.Errorf("unsupported types for operation: %s and %s", leftType, rightType)
			}

		case opcode.OpPop:
			sp--

		case opcode.OpTrue:
			stack[sp] = eval.TRUE
			sp++

		case opcode.OpFalse:
			stack[sp] = eval.FALSE
			sp++

		case opcode.OpEqual, opcode.OpNotEqual, opcode.OpGreaterThan, opcode.OpLessThan, opcode.OpGreaterThanEqual, opcode.OpLessThanEqual:
			right := stack[sp-1]
			left := stack[sp-2]
			sp -= 2

			var result bool

			// Try integer comparison first (most common)
			if leftInt, leftOk := left.(*eval.Integer); leftOk {
				if rightInt, rightOk := right.(*eval.Integer); rightOk {
					switch op {
					case opcode.OpEqual:
						result = leftInt.Value == rightInt.Value
					case opcode.OpNotEqual:
						result = leftInt.Value != rightInt.Value
					case opcode.OpGreaterThan:
						result = leftInt.Value > rightInt.Value
					case opcode.OpLessThan:
						result = leftInt.Value < rightInt.Value
					case opcode.OpGreaterThanEqual:
						result = leftInt.Value >= rightInt.Value
					case opcode.OpLessThanEqual:
						result = leftInt.Value <= rightInt.Value
					}
					if result {
						stack[sp] = eval.TRUE
					} else {
						stack[sp] = eval.FALSE
					}
					sp++
					continue
				}
			}

			// String comparison
			if leftStr, leftOk := left.(*eval.String); leftOk {
				if rightStr, rightOk := right.(*eval.String); rightOk {
					switch op {
					case opcode.OpEqual:
						result = leftStr.Value == rightStr.Value
					case opcode.OpNotEqual:
						result = leftStr.Value != rightStr.Value
					}
					if result {
						stack[sp] = eval.TRUE
					} else {
						stack[sp] = eval.FALSE
					}
					sp++
					continue
				}
			}

			// Boolean/pointer comparison
			switch op {
			case opcode.OpEqual:
				result = right == left
			case opcode.OpNotEqual:
				result = right != left
			}

			if result {
				stack[sp] = eval.TRUE
			} else {
				stack[sp] = eval.FALSE
			}
			sp++

		case opcode.OpMinus:
			operand := stack[sp-1]
			intOperand := operand.(*eval.Integer)
			stack[sp-1] = eval.NewInteger(-intOperand.Value)

		case opcode.OpBang:
			operand := stack[sp-1]
			switch operand {
			case eval.TRUE:
				stack[sp-1] = eval.FALSE
			case eval.FALSE, eval.NULL:
				stack[sp-1] = eval.TRUE
			default:
				stack[sp-1] = eval.FALSE
			}

		case opcode.OpNull:
			stack[sp] = eval.NULL
			sp++

		case opcode.OpSetGlobal:
			globalIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2
			sp--
			vm.globals[globalIndex] = stack[sp]

		case opcode.OpGetGlobal:
			globalIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2
			stack[sp] = vm.globals[globalIndex]
			sp++

		case opcode.OpSetLocal:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1
			sp--
			stack[frame.basePointer+localIndex] = stack[sp]

		case opcode.OpGetLocal:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1
			stack[sp] = stack[frame.basePointer+localIndex]
			sp++

		case opcode.OpJump:
			pos := int(opcode.ReadUint16(ins[ip+1:]))
			ip = pos - 1

		case opcode.OpJumpIfLocalGreaterEqualConst:
			localIndex := int(opcode.ReadUint8(ins[ip+1:]))
			constIndex := int(opcode.ReadUint16(ins[ip+2:]))
			pos := int(opcode.ReadUint16(ins[ip+4:]))
			ip += 5

			localObj := stack[frame.basePointer+localIndex]
			constObj := constants[constIndex]

			localInt, ok1 := localObj.(*eval.Integer)
			constInt, ok2 := constObj.(*eval.Integer)

			if !ok1 || !ok2 {
				return fmt.Errorf("operands to OpJumpIfLocalGreaterEqualConst must be integers")
			}

			if localInt.Value >= constInt.Value {
				ip = pos - 1
			}

		case opcode.OpJumpNotTruth:
			pos := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2

			sp--
			condition := stack[sp]
			if !isTruthy(condition) {
				ip = pos - 1
			}

		case opcode.OpCall:
			numArgs := int(opcode.ReadUint8(ins[ip+1:]))
			ip += 1

			// CRITICAL FIX: Save current IP to the frame BEFORE calling
			// This ensures when we return, we continue from the correct position
			frame.ip = ip

			// Sync vm.sp before calling method
			vm.sp = sp

			// Check current frame before call
			currentFrame := frame

			err := vm.executeCall(numArgs)
			if err != nil {
				return err
			}

			// Sync local sp back
			sp = vm.sp

			// Only reload if frame changed (normal function call)
			// For builtins, we stay in the same frame and continue execution
			if vm.currentFrame() != currentFrame {
				frame = vm.currentFrame()
				ip = frame.ip
				ins = frame.fn.Instructions
				stack = vm.stack
				constants = vm.constants
			}

		case opcode.OpReturnValue:
			returnValue := stack[sp-1]
			sp--

			frame = vm.popFrame()
			vm.sp = frame.basePointer - 1

			stack[vm.sp] = returnValue
			vm.sp++

			// Reload cached values
			frame = vm.currentFrame()
			ip = frame.ip
			ins = frame.fn.Instructions
			sp = vm.sp

		case opcode.OpImport:
			// Import a module from file
			constIndex := int(opcode.ReadUint16(ins[ip+1:]))
			ip += 2

			// Get the module path
			pathObj := constants[constIndex]
			pathStr, ok := pathObj.(*eval.String)
			if !ok {
				return fmt.Errorf("import path must be string")
			}

			// Load and execute the module file
			moduleObj, err := vm.loadModule(pathStr.Value)
			if err != nil {
				return fmt.Errorf("failed to import %s: %v", pathStr.Value, err)
			}

			// Push module object onto stack
			stack[sp] = moduleObj
			sp++

		case opcode.OpReturn:
			frame = vm.popFrame()
			vm.sp = frame.basePointer - 1

			stack[vm.sp] = eval.NULL
			vm.sp++

			// Reload cached values
			frame = vm.currentFrame()
			ip = frame.ip
			ins = frame.fn.Instructions
			sp = vm.sp
		}
	}

	// Save final state
	frame.ip = ip
	vm.sp = sp

	return nil
}

func (vm *VM) executeCall(numArgs int) error {
	callee := vm.stack[vm.sp-1-numArgs]
	if callee == nil {
		return fmt.Errorf("callee is nil! sp=%d, numArgs=%d", vm.sp, numArgs)
	}
	switch fn := callee.(type) {
	case *eval.Function:
		frame := NewFrame(fn, vm.sp-numArgs)
		vm.pushFrame(frame)
		vm.sp = frame.basePointer + fn.SlotCount
		return nil

	case *eval.BuiltinFunction:
		// Special handling for route() and listen() builtins that need VM access
		args := vm.stack[vm.sp-numArgs : vm.sp]

		// Check if this is the route builtin (index 15)
		if fn == vm.builtins[15] {
			if len(args) != 3 {
				return fmt.Errorf("route expects 3 arguments (method, path, handler)")
			}
			method, ok1 := args[0].(*eval.String)
			path, ok2 := args[1].(*eval.String)
			handler, ok3 := args[2].(*eval.Function)
			if !ok1 || !ok2 || !ok3 {
				return fmt.Errorf("route: invalid arguments - expected (string, string, function)")
			}

			// Store route in VM
			vm.HandleHTTPRoute(method.Value, path.Value, handler)

			result := &eval.String{Value: fmt.Sprintf("Route registered: %s %s", method.Value, path.Value)}
			vm.sp -= numArgs + 1
			return vm.push(result)
		}

		// Check if this is the listen builtin (index 16)
		if fn == vm.builtins[16] {
			if len(args) != 1 {
				return fmt.Errorf("listen expects 1 argument (port)")
			}
			port, ok := args[0].(*eval.Integer)
			if !ok {
				return fmt.Errorf("listen: port must be an integer")
			}

			// Start HTTP server (this will block!)
			fmt.Printf("🚀 Starting Flowa HTTP server on port %d...\n", port.Value)
			fmt.Printf("   Press Ctrl+C to stop\n\n")
			err := vm.StartHTTPServer(port.Value)
			if err != nil {
				return fmt.Errorf("server error: %v", err)
			}

			// This won't be reached unless server stops
			result := &eval.String{Value: "Server stopped"}
			vm.sp -= numArgs + 1
			return vm.push(result)
		}

		// Normal builtin function
		result := fn.Fn(args...)
		vm.sp -= numArgs + 1
		return vm.push(result)

	default:
		return fmt.Errorf("calling non-function: %s", callee.Kind())
	}
}

func (vm *VM) push(o eval.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() eval.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) LastPoppedStackElem() eval.Object {
	return vm.stack[vm.sp]
}

// CallGlobalFunction calls a function stored in a global variable by its index
func (vm *VM) CallGlobalFunction(globalIndex int) error {
	if globalIndex < 0 || globalIndex >= len(vm.globals) {
		return fmt.Errorf("global index out of range: %d", globalIndex)
	}

	fn, ok := vm.globals[globalIndex].(*eval.Function)
	if !ok {
		return fmt.Errorf("global at index %d is not a function", globalIndex)
	}

	// Push the function onto the stack
	vm.stack[vm.sp] = fn
	vm.sp++

	// Execute the call (this sets up the frame)
	err := vm.executeCall(0)
	if err != nil {
		return err
	}

	// Now continue execution to actually run the function
	return vm.Run()
}
func (vm *VM) executeIndexExpression(left, index eval.Object) error {
	switch {
	case left.Kind() == eval.KindArray && index.Kind() == eval.KindInteger:
		arrayObject := left.(*eval.Array)
		idx := index.(*eval.Integer).Value
		max := int64(len(arrayObject.Elements) - 1)

		if idx < 0 || idx > max {
			return vm.push(eval.NULL)
		}

		return vm.push(arrayObject.Elements[idx])

	case left.Kind() == eval.KindMap:
		hashObject := left.(*eval.Map)

		// Try direct lookup first
		if val, ok := hashObject.Pairs[index]; ok {
			return vm.push(val)
		}

		// Fallback: Linear search for value equality
		for k, v := range hashObject.Pairs {
			if k.Kind() == index.Kind() {
				switch k := k.(type) {
				case *eval.String:
					if idx, ok := index.(*eval.String); ok && k.Value == idx.Value {
						return vm.push(v)
					}
				case *eval.Integer:
					if idx, ok := index.(*eval.Integer); ok && k.Value == idx.Value {
						return vm.push(v)
					}
				case *eval.Boolean:
					if idx, ok := index.(*eval.Boolean); ok && k.Value == idx.Value {
						return vm.push(v)
					}
				}
			}
		}

		return vm.push(eval.NULL)

	case left.Kind() == eval.KindModule && index.Kind() == eval.KindString:
		module := left.(*eval.Module)
		key := index.(*eval.String).Value
		obj, ok := module.Env.Get(key)
		if !ok {
			return fmt.Errorf("property %s not found in module %s", key, module.Name)
		}
		if obj == nil {
			return fmt.Errorf("property %s in module %s is nil", key, module.Name)
		}
		return vm.push(obj)
	default:
		return fmt.Errorf("index operator not supported: %s[%s]", left.Kind(), index.Kind())
	}
}
func isTruthy(obj eval.Object) bool {
	switch obj {
	case eval.NULL:
		return false
	case eval.TRUE:
		return true
	case eval.FALSE:
		return false
	default:
		return true
	}
}
