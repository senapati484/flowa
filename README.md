# Flowa Programming Language

**Flowa** is a compact, Pythonic, pipeline-first scripting language designed for fast prototyping and server-first applications. Built with Go for blazing-fast compilation and runtime performance.

## Key Features

- 🔀 **Pipeline-first**: Elegant `|>` operator for data transformation chains
- 🐍 **Pythonic Syntax**: Indentation-sensitive, readable, easy to learn
- ⚡ **High Performance**: Go-based compiler and runtime
- 🧵 **Concurrency** (planned): Built-in `spawn` and `await` for lightweight tasks
- 🎯 **Simple & Minimal**: Clean design with powerful primitives

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/senapati484/flowa.git
cd flowa

# Build the compiler
go build -o flowac ./cmd/flowac

# Optionally install to PATH
sudo mv flowac /usr/local/bin/
```

### Run Your First Flowa Program

Create a file `hello.flowa`:

```python
def greet(name):
    return name

result = "World" |> greet()
# result = "World"
```

Run it:

```bash
flowac run hello.flowa
```

### Interactive REPL

```bash
flowac repl
```

```
Flowa REPL v0.1 (MVP)
>>> def double(x):
...     return x * 2
function
>>> 5 |> double()
10
>>> 10 |> double() |> double()
40
```

## Language Syntax

### Functions

```python
def add(x, y):
    return x + y

result = add(5, 10)  # 15
```

### Pipeline Operator

The `|>` operator passes the left-hand value as the **first argument** to the right-hand function:

```python
def increment(x):
    return x + 1

def square(x):
    return x * x

# Traditional style
result = square(increment(5))  # 36

# Pipeline style (cleaner!)
result = 5 |> increment() |> square()  # 36
```

### Pipelines with Multiple Arguments

```python
def add(x, y):
    return x + y

result = 5 |> add(10)  # 5 + 10 = 15
```

### Recursion

> **Note**: Conditional statements are being implemented. For now, recursion works but requires manual termination conditions.

```python
def double(x):
    return x * 2

result = 5 |> double()  # 10
```

## Example Programs

### 1. Hello World ([examples/hello.flowa](examples/hello.flowa))

```python
def add(x, y):
    return x + y

def multiply(x, factor):
    return x * factor

def process(n):
    return n |> add(10) |> multiply(2)

result = process(5)  # 30
```

```bash
flowac run examples/hello.flowa
```

### 2. Data Pipeline ([examples/pipeline.flowa](examples/pipeline.flowa))

```python
def square(x):
    return x * x

def increment(x):
    return x + 1

result = 5 |> increment() |> square()  # 36
```

```bash
flowac run examples/pipeline.flowa
```

### 3. Simple Recursion ([examples/fibonacci.flowa](examples/fibonacci.flowa))

```python
def double(x):
    return x * 2

result = 5 |> double()  # 10
```

```bash
flowac run examples/fibonacci.flowa
```

> **Note**: Conditional statements (`if`/`else`) are defined in the AST but not yet fully working in the  interpreter. Coming in next update!

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/lexer
go test ./pkg/parser
```

### Code Formatting

```bash
go fmt ./...
```

### Project Structure

```
flowa/
├── cmd/flowac/         # CLI compiler tool
│   └── main.go
├── pkg/
│   ├── token/          # Token definitions
│   ├── lexer/          # Tokenizer (indentation-aware)
│   ├── ast/            # Abstract Syntax Tree nodes
│   ├── parser/         # Recursive descent parser
│   └── eval/           # AST interpreter
├── examples/           # Example .flowa scripts
│   ├── hello.flowa
│   ├── fibonacci.flowa
│   └── pipeline.flowa
├── legacy_prototype/   # Original Python prototype
├── go.mod
└── README.md
```

## Implementation Status

### ✅ Phase 0: MVP (Complete)

- [x] Lexer with indentation handling
- [x] Pratt parser with proper precedence
- [x] AST-based interpreter
- [x] Pipeline operator (`|>`)
- [x] Functions and closures
- [x] REPL and CLI tool
- [x] Basic arithmetic and recursion

### 🚧 Phase 1: Compiler & Runtime (Planned)

- [ ] SSA IR generation
- [ ] LLVM backend for native compilation
- [ ] M:N scheduler for concurrency
- [ ] `spawn` and `await` keywords
- [ ] Async I/O (epoll/io_uring on Linux)
- [ ] Standard library (HTTP, DB, JSON)
- [ ] Type checker and inference

### 🔮 Phase 2: Optimization (Future)

- [ ] Escape analysis
- [ ] GC tuning
- [ ] Per-request memory arenas
- [ ] Cross-compilation support
- [ ] Zero-cost abstractions

## Design Philosophy

Flowa is designed to be:

1. **Familiar**: Pythonic syntax for easy adoption
2. **Fast**: Go-based toolchain with future native compilation
3. **Expressive**: Pipeline operators make data flows clear
4. **Simple**: Minimal core language with powerful composition
5. **Server-ready**: Built for backend services and microservices

## File Extension

Flowa files use the `.flowa` extension.

## Contributing

This is currently a prototype/learning project. Contributions and feedback are welcome!

## License

MIT

---

**Note**: This is an MVP implementation. The language is under active development. Many features (concurrency, standard library, native compilation) are planned for future phases.
