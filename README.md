<p align="center">
  <img src="https://github.com/senapati484/flowa/blob/main/data/flowa-bg-removed.png" alt="Flowa Logo" width="200" />
</p>

<h1 align="center">Flowa</h1>

<p align="center"><strong>Server-first language for modern web applications</strong></p>

<p align="center">
  <em>Python-style syntax. Go-powered performance. Built-in auth, real-time, and more.</em>
</p>

---

## 🚀 What is Flowa?

**Flowa** is a programming language designed to make server development simple and powerful. Built on Go, it combines:

- **🐍 Python-style syntax** – Clean, readable, indentation-based
- **⚡ Go performance** – Fast runtime, single binary
- **🔋 Batteries included** – Auth, JWT, WebSocket, Email built-in
- **📊 Pipeline-first** – Data flows naturally with `|>` operator

```python
# Simple, powerful syntax
def process(data):
    return data |> validate() |> transform() |> save()

# Built-in server with auth
service API on ":8080":
    post "/register" -> register_user
    post "/login" -> login_user
    get "/ws" -> websocket_handler
```

---

## 📦 Installation

### Prerequisites

**Go 1.20+** must be installed on your system.

```bash
# Check Go installation
go version
```

Don't have Go? [Install Go](https://go.dev/doc/install) first.

### Install Flowa (macOS/Linux)

#### Homebrew (Recommended)

```bash
brew tap senapati484/flowa
brew install flowa
```

#### From Source

```bash
git clone https://github.com/senapati484/flowa
cd flowa
go build -o flowa ./cmd/flowa
sudo mv flowa /usr/local/bin/
```

### Verify Installation

```bash
flowa --version
# Flowa 0.1.0
```

---

## 🎯 Why Flowa?

### Built on Go, Better than Go

Flowa inherits Go's performance and reliability, but adds:

1. **🚄 Faster Development**
   - No boilerplate (no `package main`, `import`, `func main()`)
   - Python-style syntax you already know
   - No manual error handling everywhere

2. **🔋 Standard Library on Steroids**
   - Authentication (bcrypt) built-in
   - JWT signing & verification
   - WebSocket support
   - SMTP email with templates
   - No need for external frameworks

3. **📊 Pipeline Operator**
   ```python
   # Traditional
   save(optimize(resize(image)))
   
   # Flowa
   image |> resize() |> optimize() |> save()
   ```

4. **⚡ Single Binary**
   - No virtual environments
   - No dependency hell
   - Just one executable

### vs Other Languages

| Feature | Flowa | Python | Go | Node.js |
|---------|-------|--------|-------|---------|
| Easy Syntax | ✅ | ✅ | ❌ | ✅ |
| Fast Performance | ✅ | ❌ | ✅ | ⚠️ |
| Built-in Auth | ✅ | ❌ | ❌ | ❌ |
| Built-in WebSocket | ✅ | ❌ | ⚠️ | ⚠️ |
| Single Binary | ✅ | ❌ | ✅ | ❌ |
| Pipeline Operator | ✅ | ❌ | ❌ | ❌ |

---

## 🧪 Quick Start

### 1. Hello World

```python
# hello.flowa
def greet(name):
    return "Hello, " + name

result = greet("World")
print(result)
```

```bash
flowa hello.flowa
# Hello, World
```

### 2. Pipeline Example

```python
# pipeline.flowa
def increment(x):
    return x + 1

def square(x):
    return x * x

result = 5 |> increment() |> square()
print(result)  # 36
```

### 3. Create a Server (The Flowa Way)

```python
# server.flowa
def hello(req):
    name = req.query["name"]
    return response.json({"message": "Hello, " + name}, 200)

def get_user(req):
    id = req.params["id"]
    return response.json({"user_id": id}, 200)

service MyAPI on ":8080":
    get "/" -> hello
    get "/users/:id" -> get_user
```

```bash
flowa server.flowa
# Starting service MyAPI on :8080
```

**Test it:**
```bash
curl "http://localhost:8080?name=Flowa"
# {"message":"Hello, Flowa"}

curl "http://localhost:8080/users/123"
# {"user_id":"123"}
```

---

## 🔥 Key Features

### 🌐 HTTP Server Made Easy

No framework needed. Built-in routing, middleware, and response helpers.

```python
service API on ":8080":
    get "/health" -> health_check
    post "/users" -> create_user
    get "/users/:id" -> get_user
```

### 🔐 Authentication Built-in

Bcrypt password hashing out of the box.

```python
hash = auth.hash_password("secret123")
valid = auth.verify_password(hash, "secret123")  # True
```

### 🎫 JWT Tokens

Sign and verify JSON Web Tokens for stateless auth.

```python
token = jwt.sign({"user_id": 123}, "secret", "24h")
claims = jwt.verify(token, "secret")
```

### 🔌 WebSocket Support

Real-time bidirectional communication.

```python
def chat(req):
    conn = websocket.upgrade(req)
    while True:
        msg = websocket.read(conn)
        websocket.send(conn, "Echo: " + msg)
```

### 📧 Email with Templates

SMTP email sending with template support.

```python
mail.send_template("Hello {{name}}", {
    "to": "user@example.com",
    "name": "Alice"
})
```

---

## 📚 Learn More

- **[📖 Full Documentation](DOCUMENTATION.md)** – Complete language guide with examples
- **[🔌 API Reference](API.md)** – All built-in modules and functions
- **[💻 Examples](examples/)** – Sample projects and code

---

## 📊 Implementation Status

### ✅ Complete Features
- [x] Lexer with indentation
- [x] Parser with pipeline operator
- [x] AST interpreter
- [x] Functions and closures
- [x] REPL and CLI
- [x] **String escape sequences** (`\n`, `\t`, `\r`, `\\`, `\"`, `\0`)
- [x] **JSON encoding/decoding**
- [x] **HTTP server with routing**
- [x] **Response helpers**
- [x] **Config (environment variables, `config.env()`)**
- [x] **Middleware (logger, CORS)**
- [x] **Mail (SMTP with templates)**
- [x] **Auth (bcrypt password hashing)**
- [x] **JWT (token sign/verify)**
- [x] **WebSockets (real-time communication)**

### 🚧 In Progress
- [ ] Database module (SQL, migrations)
- [ ] File system module
- [ ] HTTP client module
- [ ] Type system

### 🔮 Future
- [ ] Native compilation (LLVM)
- [ ] M:N scheduler for async
- [ ] Static typing
- [ ] Package manager

---

## 🎯 Design Philosophy

Flowa is designed to be:

1. **Familiar** – Python-style syntax for easy adoption
2. **Fast** – Go-based runtime, single binary
3. **Complete** – Built-in auth, real-time, email (no external frameworks)
4. **Expressive** – Pipeline operators make code readable
5. **Server-ready** – Perfect for APIs and microservices

---

## 👨‍💻 Creator

<p align="center">
  <img src="https://avatars.githubusercontent.com/u/112938237?v=4" alt="Creator" width="150" style="border-radius: 50%;" />
</p>

<p align="center">
  <strong>Sayan Senapati</strong><br>
  Language Designer & Developer
</p>

---

## 📧 Contact

**Email**: flowalang@gmail.com  
**GitHub**: [github.com/senapati484/flowa](https://github.com/senapati484/flowa)

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>Built with ❤️ for developers who love clean code and powerful features.</strong>
</p>
