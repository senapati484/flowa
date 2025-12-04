package main

import (
	"bufio"
	"flag"
	"flowa/pkg/compiler"
	"flowa/pkg/lexer"
	"flowa/pkg/parser"
	"flowa/pkg/version"
	"flowa/pkg/vm"
	"fmt"
	"os"
	"strings"
)

// loadEnvFile loads environment variables from .env file
func loadEnvFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		// .env file is optional, don't error if it doesn't exist
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Set environment variable
		os.Setenv(key, value)
	}

	return scanner.Err()
}

func printUsage() {
	fmt.Println("Flowa - A fast, modern programming language")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  flowa <script.flowa>       Run a Flowa script")
	fmt.Println("  flowa --help, -h           Show this help message")
	fmt.Println("  flowa --version, -v        Show version information")
	fmt.Println("  flowa --examples, -ex      Show code examples for all features")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  flowa examples/01_basics.flowa")
	fmt.Println("  flowa server.flowa")
	fmt.Println()
	fmt.Println("Documentation: https://github.com/senapati484/flowa")
	fmt.Println("Email: flowalang@gmail.com")
}

func printVersion() {
	fmt.Printf("Flowa version %s\n", version.Version)
	fmt.Printf("Build date: %s\n", version.BuildDate)
	fmt.Printf("Repository: %s\n", version.GitCommit)
}

func printExamples() {
	fmt.Println("=== Flowa Code Examples ===")
	fmt.Println()

	fmt.Println("📚 BASICS:")
	fmt.Println(`  # Variables`)
	fmt.Println(`  name = "Alice"`)
	fmt.Println(`  age = 30`)
	fmt.Println(`  numbers = [1, 2, 3, 4, 5]`)
	fmt.Println(`  user = {"name": "Bob", "age": 25}`)
	fmt.Println()
	fmt.Println(`  # Output`)
	fmt.Println(`  print("Hello, World!")`)
	fmt.Println(`  print("Name:", name, "Age:", age)`)
	fmt.Println()

	fmt.Println("📧 EMAIL (mail module):")
	fmt.Println(`  mail.send({"to": "user@example.com", "subject": "Hello", "body": "Message"})`)
	fmt.Println()

	fmt.Println("🔐 AUTHENTICATION (auth module):")
	fmt.Println(`  hash = auth.hash_password("mypassword")`)
	fmt.Println(`  if auth.verify_password("mypassword", hash) { print("Valid!") }`)
	fmt.Println()

	fmt.Println("🎫 JWT TOKENS (jwt module):")
	fmt.Println(`  token = jwt.sign({"user": "alice"}, "secret", "24h")`)
	fmt.Println(`  claims = jwt.verify(token, "secret")`)
	fmt.Println()

	fmt.Println("🌐 HTTP SERVER (http module):")
	fmt.Println(`  func handler(req){ return response.json({"status": "ok"}, 200) }`)
	fmt.Println(`  route("GET", "/api/users", handler)`)
	fmt.Println(`  listen(8080)`)
	fmt.Println()

	fmt.Println("🌍 HTTP CLIENT (http module):")
	fmt.Println(`  resp = http.get("https://api.example.com/data")`)
	fmt.Println(`  data = json.decode(resp.body)`)
	fmt.Println()

	fmt.Println("🔌 WEBSOCKETS (websocket module):")
	fmt.Println(`  conn = websocket.upgrade(req)`)
	fmt.Println(`  websocket.send(conn, "Hello!")`)
	fmt.Println(`  msg = websocket.read(conn)`)
	fmt.Println()

	fmt.Println("📊 JSON (json module):")
	fmt.Println(`  json_str = json.encode({"name": "Alice", "age": 30})`)
	fmt.Println(`  data = json.decode('{"key": "value"}')`)
	fmt.Println()

	fmt.Println("📂 FILE SYSTEM (fs module):")
	fmt.Println(`  content = fs.read("file.txt")`)
	fmt.Println(`  fs.write("output.txt", "Hello World")`)
	fmt.Println(`  if fs.exists("config.json") { print("Found!") }`)
	fmt.Println()

	fmt.Println("⚙️  CONFIG (config module):")
	fmt.Println(`  port = config.env("PORT", "8080")`)
	fmt.Println(`  secret = config.env("JWT_SECRET", "default")`)
	fmt.Println()

	fmt.Println("🚀 CONTROL FLOW:")
	fmt.Println(`  if x > 5 { print("Big") } else { print("Small") }`)
	fmt.Println(`  while count < 10 { count = count + 1 }`)
	fmt.Println(`  for item in [1,2,3] { print(item) }`)
	fmt.Println()

	fmt.Println("📝 FUNCTIONS:")
	fmt.Println(`  func add(a, b) { return a + b }`)
	fmt.Println(`  result = add(5, 10)`)
	fmt.Println()

	fmt.Println("📚 For more examples, visit: https://github.com/senapati484/flowa/tree/main/examples")
	fmt.Println()
	fmt.Println("💡 Tip: Run 'flowa --help' for usage information")
}

func main() {
	// Load .env file first (before anything else)
	loadEnvFile(".env")

	// Define flags
	helpFlag := flag.Bool("help", false, "Show help message")
	helpShort := flag.Bool("h", false, "Show help message")
	versionFlag := flag.Bool("version", false, "Show version information")
	versionShort := flag.Bool("v", false, "Show version information")
	examplesFlag := flag.Bool("examples", false, "Show code examples")
	examplesShort := flag.Bool("ex", false, "Show code examples")

	// Custom usage message
	flag.Usage = printUsage

	// Parse flags
	flag.Parse()

	// Handle flags
	if *helpFlag || *helpShort {
		printUsage()
		os.Exit(0)
	}

	if *versionFlag || *versionShort {
		printVersion()
		os.Exit(0)
	}

	if *examplesFlag || *examplesShort {
		printExamples()
		os.Exit(0)
	}

	// Check for script file
	args := flag.Args()
	if len(args) < 1 {
		// Show usage when no file is specified
		printUsage()
		os.Exit(0)
	}

	filename := args[0]
	runFile(filename)
}

func runFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Lexer
	l := lexer.New(string(content))

	// Parser
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, msg := range p.Errors() {
			fmt.Println("\t", msg)
		}
		os.Exit(1)
	}

	// Compiler
	comp := compiler.New()
	err = comp.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	// VM
	machine := vm.New(comp.Bytecode())
	err = machine.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
