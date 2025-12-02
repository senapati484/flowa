# Flowa Documentation

Complete guide to the Flowa programming language.

---

## Table of Contents

1. [Language Basics](#language-basics)
2. [⚡ Performance Builtins](#️-performance-builtins)
3. [🌐 HTTP Server](#-http-server)
4. [🔐 Authentication](#-authentication)
5. [🎫 JWT Tokens](#-jwt-tokens)
6. [🔌 WebSockets](#-websockets)
7. [📧 Email](#-email)
8. [📊 Data Handling](#-data-handling)
9. [⚙️ Configuration](#️-configuration)
10. [🛡️ Middleware](#️-middleware)
11. [Complete Examples](#complete-examples)

---

## Language Basics

### Data Types

```python
# Numbers
age = 25
price = 99.99

# Strings
name = "Flowa"
message = "Hello, World!"

# String concatenation
full_name = "First" + " " + "Last"

# Escape sequences (Python-like)
newline = "Line 1\nLine 2"  # Newline
tab = "Col1\tCol2"            # Tab
quote = "She said \"Hi\""     # Quote
backslash = "Path: C:\\folder"  # Backslash

# All escape sequences:
# \n  - Newline
# \t  - Tab
# \r  - Carriage return
# \\  - Backslash
# \"  - Double quote
# \0  - Null character

# Booleans
is_active = True
is_deleted = False

# None (null)
result = None

# Arrays
numbers = [1, 2, 3, 4, 5]
names = ["Alice", "Bob", "Charlie"]

# Maps (dictionaries)
user = {"name": "Alice", "age": 30, "role": "admin"}
```

### Functions

```python
# Basic function
def add(x, y):
    return x + y

# Function with default return
def greet(name):
    return "Hello, " + name

# Call functions
result = add(5, 10)  # 15
message = greet("World")  # "Hello, World"
```

### Pipeline Operator (`|>`)

The pipeline operator passes the left value as the **first argument** to the right function.

```python
def increment(x):
    return x + 1

def square(x):
    return x * x

def double(x):
    return x * 2

# Traditional nested calls
result = double(square(increment(5)))  # 72

# Pipeline style (reads left-to-right)
result = 5 |> increment() |> square() |> double()  # 72
```

**With multiple arguments:**

```python
def add(x, y):
    return x + y

def multiply(x, factor):
    return x * factor

# Pipeline passes value as first argument
result = 5 |> add(10) |> multiply(2)
# Equivalent to: multiply(add(5, 10), 2)
# = multiply(15, 2) = 30
```

### Control Flow

```python
# If-elif-else
if score >= 90:
    grade = "A"
elif score >= 80:
    grade = "B"
else:
    grade = "C"

# While loops
count = 0
while count < 5:
    print(count)
    count = count + 1

# For loops
for item in [1, 2, 3, 4, 5]:
    print(item)

for name in ["Alice", "Bob", "Charlie"]:
    print("Hello, " + name)
```

### Array & Map Access

```python
# Array indexing
fruits = ["apple", "banana", "cherry"]
first = fruits[0]    # "apple"
second = fruits[1]   # "banana"

# Map key access
person = {"name": "Alice", "age": 30}
name = person["name"]  # "Alice"
age = person["age"]    # 30

# Nested access
data = {"users": [{"name": "Alice"}, {"name": "Bob"}]}
first_user = data["users"][0]["name"]  # "Alice"
```

### Imports

Split your code into multiple files.

```python
# math_utils.flowa
def add(a, b):
    return a + b

PI = 3.14159
```

```python
# main.flowa
from "math_utils.flowa" import add, PI

result = add(10, 20)
print(PI)
```

**Wildcard Import:**

```python
from "math_utils.flowa" import *
```

**JS-Style Import:**

```python
import { add, PI } from "math_utils.flowa"
```

---

## 🌐 HTTP Server

Build HTTP servers with clean syntax and powerful built-in features.

### Basic Server

```python
def homepage(req):
    return response.text("Welcome to Flowa!", 200)

def about(req):
    html = "<h1>About Us</h1><p>Built with Flowa</p>"
    return response.html(html, 200)

# Register routes
route("GET", "/", homepage)
route("GET", "/about", about)

# Start server
listen(8080)
```

**Run it:**

```bash
flowa server.flowa
# Starting HTTP server on :8080
```

### Request Object

The `req` parameter contains all request information:

```python
def handle_request(req):
    # HTTP method
    method = req.method  # "GET", "POST", etc.

    # Request path
    path = req.path  # "/users/123"

    # Query parameters (?key=value)
    name = req.query["name"]
    page = req.query["page"]

    # Path parameters (:id in route)
    id = req.params["id"]

    # Headers
    content_type = req.headers["content-type"]
    auth = req.headers["authorization"]

    # Cookies
    session = req.cookies["session_id"]

    # Raw body
    body_text = req.body

    # Client IP
    ip = req.ip

    return response.json({"status": "ok"}, 200)
```

### Path Parameters

```python
def get_user(req):
    user_id = req.params["id"]
    return response.json({"user_id": user_id}, 200)

def get_comment(req):
    post_id = req.params["post_id"]
    comment_id = req.params["comment_id"]
    return response.json({
        "post": post_id,
        "comment": comment_id
    }, 200)

service API on ":8080":
    get "/users/:id" -> get_user
    get "/posts/:post_id/comments/:comment_id" -> get_comment
```

**New Syntax:**

```python
route("GET", "/users/:id", get_user)
route("GET", "/posts/:post_id/comments/:comment_id", get_comment)
listen(8080)
```

**Test:**

```bash
curl http://localhost:8080/users/123
# {"user_id":"123"}

curl http://localhost:8080/posts/456/comments/789
# {"post":"456","comment":"789"}
```

### Response Helpers

```python
# JSON response (auto Content-Type)
response.json({"name": "Alice", "age": 30}, 200)

# Plain text
response.text("Hello World", 200)

# HTML
response.html("<h1>Welcome</h1>", 200)

# Redirect
response.redirect("/dashboard", 302)

# Custom status codes
response.json({"error": "Not found"}, 404)
response.json({"error": "Unauthorized"}, 401)
response.json({"error": "Server error"}, 500)
```

### Handling POST Data

```python
def create_user(req):
    # Parse JSON body
    data = json.decode(req.body)
    username = data["username"]
    email = data["email"]

    # Process...

    return response.json({
        "message": "User created",
        "username": username
    }, 201)

# Register route
route("POST", "/users", create_user)
listen(8080)
```

**Test:**

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com"}'
```

---

## 🔐 Authentication

Built-in bcrypt password hashing for secure authentication.

### Password Hashing

```python
# Hash a password
password = "mysecretpassword123"
hash = auth.hash_password(password)
# → "$2a$10$abcdef1234567890..."

# Store hash in database (never store plain passwords!)
users["alice"] = hash
```

**Security Notes:**

- Uses bcrypt with cost factor 10
- Automatically salted
- Industry-standard secure hashing

### Password Verification

```python
# Verify password against hash
stored_hash = users["alice"]
password_attempt = "mysecretpassword123"

if auth.verify_password(stored_hash, password_attempt):
    print("Login successful!")
else:
    print("Invalid password")
```

### Complete Registration & Login Example

```python
# In-memory user database
users = {}

def register(req):
    data = json.decode(req.body)
    username = data["username"]
    password = data["password"]

    # Check if user exists
    if username in users:
        return response.json({"error": "User already exists"}, 400)

    # Hash password and store
    hash = auth.hash_password(password)
    users[username] = hash

    return response.json({"message": "Registration successful"}, 201)

def login(req):
    data = json.decode(req.body)
    username = data["username"]
    password = data["password"]

    # Check if user exists
    if username not in users:
        return response.json({"error": "Invalid credentials"}, 401)

    # Verify password
    hash = users[username]
    if auth.verify_password(hash, password):
        return response.json({"message": "Login successful"}, 200)
    else:
        return response.json({"error": "Invalid credentials"}, 401)

service AuthAPI on ":8080":
    post "/register" -> register
    post "/login" -> login
```

**Test Registration:**

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}'
```

**Test Login:**

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}'
```

---

## 🎫 JWT Tokens

JSON Web Tokens for stateless authentication.

### Creating Tokens

```python
# Sign a token
payload = {
    "user_id": 123,
    "username": "alice",
    "role": "admin"
}

secret = "your-secret-key-keep-it-safe"
expiry = "24h"  # Expires in 24 hours

token = jwt.sign(payload, secret, expiry)
# → "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**Duration formats:**

- `"1h"` - 1 hour
- `"24h"` - 24 hours
- `"30m"` - 30 minutes
- `"7d"` - 7 days

### Verifying Tokens

```python
# Verify and decode token
secret = "your-secret-key-keep-it-safe"
token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

claims = jwt.verify(token, secret)

if claims != None:
    # Token is valid
    user_id = claims["user_id"]
    username = claims["username"]
    role = claims["role"]
    print("Authenticated as:", username)
else:
    # Token is invalid or expired
    print("Authentication failed")
```

### Login with JWT

```python
users = {}

def register(req):
    data = json.decode(req.body)
    hash = auth.hash_password(data["password"])
    users[data["username"]] = hash
    return response.json({"message": "Registered"}, 201)

def login(req):
    data = json.decode(req.body)
    username = data["username"]

    # Verify password
    if username not in users:
        return response.json({"error": "Invalid credentials"}, 401)

    if not auth.verify_password(users[username], data["password"]):
        return response.json({"error": "Invalid credentials"}, 401)

    # Generate JWT token
    payload = {"username": username, "role": "user"}
    token = jwt.sign(payload, "my-secret-key", "24h")

    return response.json({
        "message": "Login successful",
        "token": token
    }, 200)

service AuthAPI on ":8080":
    post "/register" -> register
    post "/login" -> login
```

### Protected Routes

```python
def get_profile(req):
    # Extract token from Authorization header
    auth_header = req.headers["authorization"]

    if auth_header == None:
        return response.json({"error": "No token provided"}, 401)

    # In production: extract "Bearer <token>"
    token = auth_header

    # Verify token
    claims = jwt.verify(token, "my-secret-key")

    if claims == None:
        return response.json({"error": "Invalid token"}, 401)

    # Token is valid - return user data
    return response.json({
        "username": claims["username"],
        "role": claims["role"]
    }, 200)

service API on ":8080":
    post "/login" -> login
    get "/profile" -> get_profile  # Protected route
```

**Test Protected Route:**

```bash
# Login to get token
TOKEN=$(curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}' \
  | jq -r '.token')

# Access protected route
curl http://localhost:8080/profile \
  -H "Authorization: $TOKEN"
```

---

## 🔌 WebSockets

Real-time bidirectional communication.

### Basic WebSocket Server

```python
def ws_handler(req):
    # Upgrade HTTP connection to WebSocket
    conn = websocket.upgrade(req)

    if conn == None:
        return response.text("WebSocket upgrade failed", 500)

    # Send welcome message
    websocket.send(conn, "Connected to Flowa WebSocket!")

    # Read-echo loop
    while True:
        msg = websocket.read(conn)

        if msg == None:
            # Client disconnected
            print("Client disconnected")
            break

        print("Received:", msg)
        websocket.send(conn, "Echo: " + msg)

    # Clean up
    websocket.close(conn)
    return None  # Response handled by WebSocket

service ChatServer on ":8080":
    get "/ws" -> ws_handler
```

### WebSocket API

**`websocket.upgrade(req)`** - Upgrade HTTP to WebSocket

- Returns: Connection object or `None`

**`websocket.send(conn, message)`** - Send text message

- `conn`: WebSocket connection
- `message`: String to send

**`websocket.read(conn)`** - Read next message (blocking)

- Returns: String message or `None` (disconnected)

**`websocket.close(conn)`** - Close connection

### Chat Room Example

```python
# Simple chat with broadcast
connections = []

def chat_handler(req):
    conn = websocket.upgrade(req)
    if conn == None:
        return response.text("Failed", 500)

    # Add to connections list
    connections = connections + [conn]

    websocket.send(conn, "Welcome to chat!")

    while True:
        msg = websocket.read(conn)
        if msg == None:
            break

        # Broadcast to all connections
        broadcast_message = "User: " + msg
        for other_conn in connections:
            websocket.send(other_conn, broadcast_message)

    websocket.close(conn)
    return None

service ChatServer on ":8080":
    get "/chat" -> chat_handler
```

### Client-Side (JavaScript)

```html
<script>
  const ws = new WebSocket("ws://localhost:8080/ws");

  ws.onopen = () => {
    console.log("Connected");
    ws.send("Hello Server!");
  };

  ws.onmessage = (event) => {
    console.log("Received:", event.data);
  };

  ws.onclose = () => {
    console.log("Disconnected");
  };
</script>
```

---

## 📧 Email

SMTP email sending with template support.

### Environment Setup

Set these environment variables before sending emails:

```bash
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=your_email@gmail.com
export SMTP_PASS=your_app_password
```

**Gmail Users:** Use an [App Password](https://support.google.com/accounts/answer/185833), not your regular password.

### Simple Email

```python
mail.send({
    "to": "user@example.com",
    "from": "noreply@myapp.com",
    "subject": "Welcome to Our App",
    "body": "Thanks for signing up! We're excited to have you."
})
```

**Fields:**

- `to` (required): Recipient email
- `from` (optional): Sender email (defaults to SMTP_USER)
- `subject` (required): Email subject
- `subject` (required): Email subject
- `body` (required): Email body text (plain text)
- `html` (optional): HTML body content (overrides plain text)

### HTML Email

```python
html_content = "<h1>Welcome</h1><p>Thanks for joining!</p>"

mail.send({
    "to": "user@example.com",
    "subject": "Welcome",
    "html": html_content
})
```

### Template Emails

```python
# Define template with {{variable}} placeholders
template = "Hello {{name}},\n\nYour verification code is: {{code}}\n\nExpires in 10 minutes."

# Send with data
mail.send_template(template, {
    "to": "alice@example.com",
    "subject": "Email Verification",
    "name": "Alice",
    "code": "1234567"
})
```

### Registration Email Example

```python
def send_welcome_email(username, email):
    # Get SMTP credentials from environment
    smtp_host = config.env("SMTP_HOST", "")
    smtp_user = config.env("SMTP_USER", "")

    if smtp_host == "":
        print("Warning: SMTP not configured")
        return

    template = """
Hello {{username}},

Welcome to our platform! We're excited to have you on board.

Your account has been successfully created.

Best regards,
The Team
    """

    mail_data = {"to": email, "from": smtp_user, "subject": "Welcome to Our Platform", "username": username}
    mail.send_template(template, mail_data)

def register(req):
    data = json.decode(req.body)
    username = data["username"]
    email = data["email"]
    password = data["password"]

    # Create user
    hash = auth.hash_password(password)
    users[username] = {"hash": hash, "email": email}

    # Send welcome email
    send_welcome_email(username, email)

    return response.json({"message": "Registration successful"}, 201)
```

**Note:** Mail module automatically reads `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` from environment. You can also read them with `config.env()` for validation.

### Background Email (Async)

```python
# Send email in background (non-blocking)
mail.queue({
    "to": "admin@myapp.com",
    "subject": "Daily Report",
    "body": "Here's today's metrics..."
})

# Code continues immediately
return response.json({"status": "sent"}, 200)
```

**Use `mail.queue()` for:**

- Welcome emails
- Notifications
- Reports
- Any non-critical emails

---

## 📊 Data Handling

### JSON Module

**Encode (Object → JSON string):**

```python
data = {
    "name": "Alice",
    "age": 30,
    "skills": ["Python", "Go", "Flowa"],
    "active": True
}

json_string = json.encode(data)
# → '{"name":"Alice","age":30,"skills":["Python","Go","Flowa"],"active":true}'
```

**Decode (JSON string → Object):**

```python
json_string = '{"name":"Alice","age":30}'
data = json.decode(json_string)

name = data["name"]  # "Alice"
age = data["age"]    # 30
```

**With Arrays:**

```python
json_array = '[1, 2, 3, 4, 5]'
numbers = json.decode(json_array)
first = numbers[0]  # 1
```

---

## 📂 File System (`fs`)

Built-in module for file operations.

```python
# Read a file
content = fs.read("data.txt")

# Write to a file (overwrites)
fs.write("output.txt", "Hello World")

# Append to a file
fs.append("log.txt", "New log entry\n")

# Check if file exists
if fs.exists("config.json"):
    print("Config found")

# Remove a file
fs.remove("temp.txt")
```

---

## 🌐 HTTP Client

Make HTTP requests to external APIs.

### GET Request

```python
# Simple GET
resp = http.get("https://api.example.com/users")

print("Status:", resp.status)
print("Body:", resp.body)

# Parse JSON response
data = json.decode(resp.body)
print("First user:", data[0]["name"])
```

### POST Request

```python
# Prepare payload
payload = {"title": "New Post", "body": "Content", "userId": 1}
headers = {"Content-Type": "application/json"}

# Make POST request
resp = http.post(
    "https://api.example.com/posts",
    json.encode(payload),
    headers
)

if resp.status == 201:
    print("Created!")
    result = json.decode(resp.body)
    print("ID:", result["id"])
```

### Response Object

```python
resp = http.get("https://api.example.com/data")

# Available fields
resp.status    # HTTP status code (200, 404, etc.)
resp.body      # Response body as string
resp.headers   # Map of headers
```

---

## ⚙️ Configuration

### Environment Variables

```python
# Read environment variable with default
port = config.env("PORT", "8080")
db_url = config.env("DATABASE_URL", "postgres://localhost/mydb")
jwt_secret = config.env("JWT_SECRET", "default-secret")

# Use in service
service API on (":" + port):
    get "/" -> handler
```

**Set environment variables:**

```bash
# In terminal
export PORT=3000
export JWT_SECRET=super-secret-key

# Or in .env file (requires manual loading)
PORT=3000
JWT_SECRET=super-secret-key
```

---

## 🛡️ Middleware

### Logger Middleware

Logs all requests with method and path:

```python
logger = middleware.logger()

service API on ":8080":
    use logger  # Apply to all routes

    get "/" -> home
    get "/users" -> users
```

**Output:**

```
[LOG] GET /
[LOG] GET /users
```

### CORS Middleware

Adds CORS headers to responses:

```python
cors = middleware.cors()

service API on ":8080":
    get "/api/data" -> get_data, [cors]  # Route-specific
```

**Headers added:**

- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`

### Multiple Middleware

```python
logger = middleware.logger()
cors = middleware.cors()

service API on ":8080":
    use logger  # Global

    get "/public" -> public_handler
    get "/api/data" -> api_handler, [cors]  # Additional middleware
```

---

## Complete Examples

### Full Authentication System

```python
users = {}

def register(req):
    data = json.decode(req.body)
    username = data["username"]
    email = data["email"]
    password = data["password"]

    if username in users:
        return response.json({"error": "User exists"}, 400)

    hash = auth.hash_password(password)
    users[username] = {"hash": hash, "email": email}

    # Send welcome email
    mail.send_template("Welcome {{name}}!", {
        "to": email,
        "subject": "Welcome",
        "name": username
    })

    return response.json({"message": "Registered"}, 201)

def login(req):
    data = json.decode(req.body)
    username = data["username"]
    password = data["password"]

    if username not in users:
        return response.json({"error": "Invalid"}, 401)

    if not auth.verify_password(users[username]["hash"], password):
        return response.json({"error": "Invalid"}, 401)

    token = jwt.sign({"username": username}, "secret", "24h")

    return response.json({"token": token}, 200)

def get_profile(req):
    token = req.headers["authorization"]
    claims = jwt.verify(token, "secret")

    if claims == None:
        return response.json({"error": "Unauthorized"}, 401)

    username = claims["username"]
    user_data = users[username]

    return response.json({
        "username": username,
        "email": user_data["email"]
    }, 200)

service AuthAPI on ":8080":
    post "/register" -> register
    post "/login" -> login
    get "/profile" -> get_profile
```

### Real-time Chat with Auth

```python
def chat_handler(req):
    # Verify JWT before upgrading
    token = req.query["token"]
    claims = jwt.verify(token, "secret")

    if claims == None:
        return response.json({"error": "Unauthorized"}, 401)

    username = claims["username"]

    conn = websocket.upgrade(req)
    if conn == None:
        return response.text("Failed", 500)

    websocket.send(conn, "Welcome, " + username + "!")

    while True:
        msg = websocket.read(conn)
        if msg == None:
            break

        # Broadcast with username
        websocket.send(conn, username + ": " + msg)

    websocket.close(conn)
    return None

service ChatApp on ":8080":
    get "/chat" -> chat_handler
```

---

For more examples, see the [examples/](examples/) directory.

**→ [API Reference](API.md)** for detailed function documentation.

---

## ⚡ Performance Builtins

Flowa is an interpreted language, but for **numeric heavy workloads** you can call
special **performance-builtins** that are implemented directly in Go. These avoid
most per-iteration overhead and are much closer to raw Go performance.

### `fast_sum_to(n)`

Efficiently compute the sum \(0 + 1 + \dots + (n-1)\) in native Go code.

```python
n = 10000000
total = fast_sum_to(n)
print(total)  # 49999995000000
```

- **Arguments**:
  - `n` (INTEGER, non‑negative) – number of iterations
- **Returns**:
  - INTEGER sum
- **Use when**: you would otherwise write:

```python
sum = 0
i = 0
while i < n:
    sum = sum + i
    i = i + 1
```

For benchmarks like the intensive loop in `examples/benchmark_test`, prefer:

```python
print("Starting intensive benchmark...")

run = 0
while run < 5:
    sum = fast_sum_to(10000000)

    print("Run:")
    print(run + 1)
    print("Sum:")
    print(sum)

    run = run + 1

print("Benchmark complete!")
```

### `fast_sum_range(start, end)`

Efficiently compute the sum \(start + (start+1) + \dots + (end-1)\).

```python
total = fast_sum_range(10, 20)  # 10+...+19
print(total)
```

- **Arguments**:
  - `start` (INTEGER)
  - `end` (INTEGER, must be `>= start`)
- **Returns**: INTEGER sum

### `fast_repeat(n, fn)`

Call a function or builtin `fn(i)` **n times**, with a native loop managing
the counter.

```python
def work(i):
    # lightweight, side-effecting work
    print(i)

fast_repeat(1000000, work)
```

- **Arguments**:
  - `n` (INTEGER, non‑negative)
  - `fn` (FUNCTION or BUILTIN) taking one INTEGER argument
- **Returns**: `None` (NULL in Flowa); results of `fn` are ignored.

This does **not** remove the cost of calling back into the interpreter each
iteration, but it avoids extra allocations for loop structure and is a better
fit for simple repeated calls than hand‑written Flowa `while`/`for` loops.

> **Tip:** For maximum performance, prefer `fast_sum_to` / `fast_sum_range`
> when you can express your workload as a numeric accumulation. Use
> `fast_repeat` when you must run Flowa code each iteration but still want a
> cheaper loop structure implemented in Go.

### Timing & Benchmarking (`time` module)

Use the built-in `time` module to measure elapsed time in milliseconds, similar
to Go’s `time.Since(start).Seconds()` (but with integer millisecond precision).

```python
start = time.now_ms()

# ... your heavy work here ...

elapsed_ms = time.since_ms(start)
print("Elapsed (ms):")
print(elapsed_ms)
```

Combined with `fast_sum_to`, you can write Go-style numeric benchmarks:

```python
print("Starting intensive benchmark...")

run = 0
while run < 5:
    start = time.now_ms()

    sum = fast_sum_to(100000000)

    elapsed_ms = time.since_ms(start)

    print("Run:")
    print(run + 1)
    print("Sum:")
    print(sum)
    print("Time (ms):")
    print(elapsed_ms)

    run = run + 1

print("Benchmark complete!")
```
