<p align="center">
  <img src="https://github.com/senapati484/flowa/blob/main/data/flowa-bg-removed.png" alt="Flowa Logo" width="200" />
</p>

<h1 align="center">Flowa</h1>

<p align="center"><strong>Pipeline‑first language for modern data and backend flows.</strong></p>

<p align="center">
  <em>Python‑style readability. Go‑powered speed. Built for the pipeline era.</em>
</p>

---

**Flowa** is a small but expressive programming language focused on one idea:
make **data flow** the center of your code. The pipeline operator `|>` turns
deeply nested calls into readable, linear flows that are perfect for scripting,
services, and experimentation.

## 🚀 Why Flowa?

Most languages force you to choose between **simplicity** (Python) and **performance** (Go/Rust). Flowa eliminates this compromise.

### 🌟 Unique Selling Points:

1.  **Pipeline‑first design** – The `|>` operator isn't just sugar; it's the
    core primitive. Data flows linearly, eliminating nested function hell.

    ```python
    # Traditional
    save(optimize(resize(image)))

    # Flowa
    image |> resize() |> optimize() |> save()
    ```

2.  **Zero‑boilerplate async (MVP semantics today)** – `spawn` and `await`
    give you a task abstraction for simple concurrency experiments. The
    current interpreter runs tasks synchronously but preserves the language
    surface so the runtime can evolve.

3.  **The "Goldilocks" syntax**:

    - Indentation-based (like Python) for readability.
    - Static typing potential (like Go) for reliability.
    - Minimalist keywords for zero learning curve.

4.  **Single‑binary toolchain** – No virtual environments, no complex build
    tools. One binary (`flowa`) does it all: run scripts, REPL, inspect
    pipelines, print ASTs.

## ✨ Highlights

- **Pipeline operator (`|>`)** – Compose transformations in a straight line.
- **Clean, indentation‑based syntax** – Instantly familiar to Python users.
- **Go‑powered implementation** – A tiny, fast single binary.
- **REPL & tooling** – `repl`, `ast`, `inspect`, and `pipelines` help you
  explore programs as data flows.
- **HTTP helpers (MVP)** – Tiny `route`, `response`, `listen` helpers make it
  easy to spin up demo servers in pure Flowa.

## 📦 Installation

### Quick Start

Choose your platform for detailed installation instructions:

- **macOS**: [macOS Installation Guide](INSTALL_MAC_LINUX.md#macos-installation-methods)
- **Linux**: [Linux Installation Guide](INSTALL_MAC_LINUX.md#linux-installation-methods)
- **Windows**: [Windows Installation Guide](INSTALL_WINDOWS.md)

### Quick Install (macOS/Linux)

```bash
# Clone and install
curl -L https://github.com/senapati484/flowa/raw/main/install.sh | sh

# Verify installation
flowa --version
```

### Homebrew (macOS)

```bash
brew tap senapati484/flowa
brew install flowa
```

### Verify Installation

```bash
flowa --version
flowa run examples/hello.flowa
```

For more installation options and troubleshooting, see the full [Installation Guide](INSTALL_MAC_LINUX.md).

---

## 🧪 First Steps

### Run a Flowa script

```bash
# Simple: just provide the filename
flowa hello.flowa

# Or use the explicit 'run' command
flowa run hello.flowa
```

### Interactive REPL

```bash
flowa repl
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

### Language insights & tooling

Flowa ships with exploratory commands so you can treat your scripts as living
pipelines:

```bash
# Print version + build metadata
flowa --version

# Summarize functions and pipeline chains inside a script
flowa inspect examples/pipeline.flowa

# Show only pipeline chains (great for docs / reviews)
flowa pipelines examples/pipeline.flowa

# Dump the parsed AST to understand how the compiler sees your code
flowa ast examples/hello.flowa

# Need a quick refresher on commands?
flowa help
```

Example inspector output:

```
Functions (2)
  · def square(x)
  · def increment(x)
Pipelines (1)
  · Pipeline 1: 5 |> increment() |> square()
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
flowa examples/hello.flowa
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
flowa examples/pipeline.flowa
```

### 3. Simple Recursion ([examples/fibonacci.flowa](examples/fibonacci.flowa))

```python
def double(x):
    return x * 2

result = 5 |> double()  # 10
```

```bash
flowa examples/fibonacci.flowa
```

### HTTP Helper

Coming soon!

## Development

### Building from Source

```bash
# Using Make
make build

# Or using Go directly
go build -o flowa ./cmd/flowa
```

### Running Tests

```bash
# Run all tests
make test

# Or using Go
go test ./...

# Run specific package tests
go test ./pkg/lexer
go test ./pkg/parser
```

### Code Formatting

```bash
go fmt ./...
```

### Uninstalling

```bash
make uninstall
```

### Project Structure

```
flowa/
├── cmd/flowa/          # CLI compiler tool
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
