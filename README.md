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

### macOS/Linux

**Quick Install (Recommended):**
```bash
# Using Homebrew
brew tap senapati484/flowa
brew install flowa
```

**From Source:**
```bash
git clone https://github.com/senapati484/flowa
cd flowa
./install.sh
```

**Verify:**
```bash
flowa --version  # Flowa 0.1.1
```

> **Note:** Requires Go 1.20+. [Install Go](https://go.dev/doc/install) if needed.
> 
> For detailed installation instructions, see:
> - [Mac/Linux Installation Guide](INSTALL_MAC_LINUX.md)
> - [Windows Installation Guide](INSTALL_WINDOWS.md)

---

## 🎯 Why Flowa?

Flowa combines **Python's simplicity** with **Go's performance**, plus built-in features that usually require frameworks:

- **🚄 Faster Development** – No boilerplate, Python-style syntax
- **🔋 Batteries Included** – Auth, JWT, WebSocket, Email built-in
- **📊 Pipeline Operator** – Clean data transformations with `|>`
- **⚡ Single Binary** – No dependencies, just one executable

### vs Other Languages

| Feature | Flowa | Python | Go | Node.js |
|---------|-------|--------|-------|------------|
| Easy Syntax | ✅ | ✅ | ❌ | ✅ |
| Fast Performance | ✅ | ❌ | ✅ | ⚠️ |
| Built-in Auth | ✅ | ❌ | ❌ | ❌ |
| Built-in WebSocket | ✅ | ❌ | ⚠️ | ⚠️ |
| Single Binary | ✅ | ❌ | ✅ | ❌ |
| Pipeline Operator | ✅ | ❌ | ❌ | ❌ |

---

## 🧪 Quick Start

**1. Hello World**
```python
# hello.flowa
def greet(name):
    return "Hello, " + name

print(greet("World"))  # Hello, World
```

**2. Pipeline Example**
```python
def increment(x): return x + 1
def square(x): return x * x

result = 5 |> increment() |> square()
print(result)  # 36
```

**3. Simple Web Server**
```python
def hello(req):
    return response.json({"message": "Hello, Flowa!"}, 200)

service API on ":8080":
    get "/" -> hello
```

Run it: `flowa server.flowa`

---

## 🔥 Key Features

**🌐 HTTP Server** – Built-in routing and middleware
```python
service API on ":8080":
    get "/users/:id" -> get_user
    post "/users" -> create_user
```

**🔐 Authentication** – Bcrypt password hashing
```python
hash = auth.hash_password("secret123")
valid = auth.verify_password(hash, "secret123")
```

**🎫 JWT Tokens** – Stateless authentication
```python
token = jwt.sign({"user_id": 123}, "secret", "24h")
claims = jwt.verify(token, "secret")
```

**🔌 WebSocket** – Real-time communication
```python
def chat(req):
    conn = websocket.upgrade(req)
    msg = websocket.read(conn)
    websocket.send(conn, "Echo: " + msg)
```

**📧 Email** – SMTP with HTML templates
```python
mail.send({
    "to": "user@example.com",
    "subject": "Welcome!",
    "html": "<h1>Hello!</h1>"
})
```

---

## 📚 Documentation

- **[📖 Complete Guide](DOCUMENTATION.md)** – Full language reference
- **[🔌 API Reference](API.md)** – All built-in modules
- **[💻 Examples](examples/)** – Sample projects
- **[🚀 Quick Start](QUICKSTART.md)** – Get started in 5 minutes
- **[📊 Benchmarks](benchmarks/RESULTS.md)** – Performance comparisons

---

## 📊 Implementation Status

### ✅ Complete
- [x] Lexer, Parser, AST Interpreter
- [x] Functions, closures, pipelines
- [x] String operations with escape sequences
- [x] JSON encoding/decoding
- [x] HTTP server with routing
- [x] Response helpers (JSON, HTML, text)
- [x] Environment configuration
- [x] Middleware (logger, CORS)
- [x] Mail (SMTP with templates)
- [x] Auth (bcrypt hashing)
- [x] JWT (sign/verify)
- [x] WebSocket support

### 🚧 In Progress
- [ ] File system module
- [ ] HTTP client module
- [ ] Database support
- [ ] Type system

### 🔮 Planned
- [ ] Package manager
- [ ] Static typing
- [ ] Native compilation

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
