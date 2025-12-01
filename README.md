# Flowa Programming Language 🚀

**Flowa** is a modern, developer-friendly programming language designed for building scalable web servers and real-time applications with ease. It combines the simplicity of Python with the performance of Go.

<p align="center">
  <img src="https://avatars.githubusercontent.com/u/112938237?v=4" alt="Creator" width="150" style="border-radius: 50%;" />
</p>

---

## ✨ Key Features

- **Python-like Syntax**: Clean, readable code with significant indentation.
- **Built-in Web Server**: HTTP and WebSocket support out of the box.
- **Robust Email**: Send HTML emails easily with built-in SMTP support.
- **Secure Auth**: Built-in bcrypt hashing and JWT token generation.
- **Pipeline Operator**: `|>` for readable data transformations.
- **Zero Config**: `.env` support and sensible defaults.

## 🚀 Quick Start

### 1. Install Flowa

**Mac/Linux:**
```bash
./install.sh
```
See [INSTALL_MAC_LINUX.md](INSTALL_MAC_LINUX.md) for details.

**Windows:**
See [INSTALL_WINDOWS.md](INSTALL_WINDOWS.md).

### 2. Create a Server

Create `server.flowa`:

```python
# Import logic
from "utils.flowa" import helper

def home(req):
    return response.html("<h1>Welcome to Flowa!</h1>", 200)

# Register routes
route("GET", "/", home)

# Start server
listen(8080)
```

### 3. Run it

```bash
flowa server.flowa
# Starting HTTP server on :8080
```

## 📚 Documentation

- **[DOCUMENTATION.md](DOCUMENTATION.md)**: Comprehensive guide to all features.
- **[API.md](API.md)**: Detailed API reference.
- **[examples/](examples/)**: Working example code.

## 🛠️ Building from Source

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
go build -o flowa ./cmd/flowa
```

## 👨‍💻 Creator

Created by **Sayan Senapati**.

---
*Built with ❤️ in Go.*
