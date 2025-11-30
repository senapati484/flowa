# Flowa API Reference

Complete reference for all built-in modules and functions.

---

## Core Language

### `print(value...)`
Print values to console.

```python
print("Hello")
print("Name:", name, "Age:", age)
```

---

## JSON Module

### `json.encode(data)`
Convert Flowa objects to JSON string.

**Parameters:**
- `data` - Map, Array, String, Integer, or Boolean

**Returns:** String (JSON)

```python
data = {"name": "Alice", "items": [1, 2, 3]}
json_str = json.encode(data)
# → '{"name":"Alice","items":[1,2,3]}'
```

### `json.decode(json_string)`
Parse JSON string to Flowa object.

**Parameters:**
- `json_string` - Valid JSON string

**Returns:** Map, Array, or primitive type

```python
json_str = '{"name":"Alice","age":30}'
data = json.decode(json_str)
name = data["name"]  # "Alice"
```

---

## Response Module

### `response.json(data, status)`
Create JSON response.

```python
response.json({"status": "ok"}, 200)
response.json({"error": "Not found"}, 404)
```

### `response.text(text, status)`
Create plain text response.

```python
response.text("Hello World", 200)
```

### `response.html(html, status)`
Create HTML response.

```python
response.html("<h1>Welcome</h1>", 200)
```

### `response.redirect(url, status)`
Create redirect response.

```python
response.redirect("/dashboard", 302)
response.redirect("/login", 301)
```

---

## Config Module

### `config.env(key, default)`
Read environment variable.

**Parameters:**
- `key` - Environment variable name
- `default` - Default value if not set

**Returns:** String

```python
port = config.env("PORT", "8080")
db_url = config.env("DATABASE_URL", "postgres://localhost/db")
```

---

## Middleware Module

### `middleware.logger()`
Request logging middleware.

Logs: `[LOG] METHOD PATH`

```python
logger = middleware.logger()
use logger
```

### `middleware.cors()`
Add CORS headers to response.

Headers:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`

```python
cors = middleware.cors()
get "/api/data" -> handler, [cors]
```

---

## Mail Module

### `mail.send(config)`
Send email via SMTP.

**Parameters:**
- `config` - Map with keys:
  - `to` (String) - Recipient email
  - `from` (String, optional) - Sender email
  - `subject` (String) - Email subject
  - `body` (String) - Email body

**Returns:** Boolean (True on success)

```python
mail.send({
    "to": "user@example.com",
    "from": "noreply@app.com",
    "subject": "Welcome!",
    "body": "Thanks for signing up!"
})
```

**Environment Variables:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASS=your_app_password
```

### `mail.send_template(template, data)`
Send email with template.

**Parameters:**
- `template` - String with `{{variable}}` placeholders
- `data` - Map with template variables + email config

```python
template = "Hello {{name}}, your code is {{code}}"
mail.send_template(template, {
    "to": "user@example.com",
    "subject": "Verification",
    "name": "Alice",
    "code": "1234"
})
```

### `mail.queue(config)`
Send email in background (async).

```python
mail.queue({
    "to": "admin@app.com",
    "subject": "Report",
    "body": "Daily metrics..."
})
```

---

## Auth Module

### `auth.hash_password(password)`
Hash password with bcrypt.

**Parameters:**
- `password` - Plain text password (String)

**Returns:** String (bcrypt hash)

**Cost Factor:** 10 (default bcrypt cost)

```python
password = "mysecret123"
hash = auth.hash_password(password)
# → "$2a$10$NTu//lQrlUdZwr2Ns862duXq..."
```

### `auth.verify_password(hash, password)`
Verify password against hash.

**Parameters:**
- `hash` - Bcrypt hash (String)
- `password` - Plain text password to verify (String)

**Returns:** Boolean (True if valid)

```python
hash = "$2a$10$NTu//lQrlUdZwr2Ns862duXq..."
valid = auth.verify_password(hash, "mysecret123")  # True
invalid = auth.verify_password(hash, "wrong")       # False
```

---

## JWT Module

### `jwt.sign(payload, secret, expiresIn)`
Create signed JWT token.

**Parameters:**
- `payload` - Map with data to encode
- `secret` - Signing key (String)
- `expiresIn` - Duration string (e.g., "1h", "24h", "7d")

**Returns:** String (JWT token)

**Algorithm:** HS256

```python
payload = {"user_id": 123, "role": "admin"}
token = jwt.sign(payload, "my-secret-key", "24h")
# → "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**Duration formats:**
- `"1h"` - 1 hour
- `"24h"` - 24 hours
- `"30m"` - 30 minutes
- `"7d"` - 7 days

### `jwt.verify(token, secret)`
Verify and decode JWT token.

**Parameters:**
- `token` - JWT token string
- `secret` - Signing key (must match sign key)

**Returns:** Map (payload) or None (invalid/expired)

```python
claims = jwt.verify(token, "my-secret-key")
if claims != None:
    user_id = claims["user_id"]
    role = claims["role"]
    exp = claims["exp"]  # Expiration timestamp
else:
    # Invalid or expired token
    print("Auth failed")
```

---

## WebSocket Module

### `websocket.upgrade(req)`
Upgrade HTTP connection to WebSocket.

**Parameters:**
- `req` - Request object from route handler

**Returns:** Connection object or None (upgrade failed)

```python
def wsHandler(req):
    conn = websocket.upgrade(req)
    if conn == None:
        return response.text("Upgrade failed", 500)
    # ... use connection
```

### `websocket.send(conn, message)`
Send text message to client.

**Parameters:**
- `conn` - WebSocket connection
- `message` - String to send

**Returns:** Boolean (True on success)

```python
websocket.send(conn, "Hello Client!")
websocket.send(conn, json.encode({"type": "update", "data": 123}))
```

### `websocket.read(conn)`
Read next message from client (blocking).

**Parameters:**
- `conn` - WebSocket connection

**Returns:** String (message) or None (disconnected)

```python
while True:
    msg = websocket.read(conn)
    if msg == None:
        break  # Client disconnected
    print("Received:", msg)
```

### `websocket.close(conn)`
Close WebSocket connection.

**Parameters:**
- `conn` - WebSocket connection

```python
websocket.close(conn)
```

---

## Request Object

Available in route handlers as `req` parameter.

### Fields

- `req.method` - HTTP method (String): "GET", "POST", etc.
- `req.path` - Request path (String): "/users/123"
- `req.body` - Raw request body (String)
- `req.ip` - Client IP address (String)

### Maps

- `req.params` - Path parameters (Map)
  ```python
  # Route: /users/:id
  id = req.params["id"]
  ```

- `req.query` - Query parameters (Map)
  ```python
  # GET /search?q=flowa&page=1
  q = req.query["q"]        # "flowa"
  page = req.query["page"]  # "1"
  ```

- `req.headers` - HTTP headers (Map, case-insensitive)
  ```python
  content_type = req.headers["content-type"]
  auth = req.headers["authorization"]
  ```

- `req.cookies` - Cookies (Map)
  ```python
  session = req.cookies["session_id"]
  ```

### Methods

- `req.text()` - Get body as string
- `req.json()` - Parse body as JSON (returns Map/Array)
- `req.form()` - Parse form data (Map)

---

## Service Definition

### `service NAME on "ADDRESS":`
Define HTTP server.

```python
service MyAPI on ":8080":
    get "/" -> index
    post "/users" -> create_user
```

### Route Methods
- `get PATH -> HANDLER`
- `post PATH -> HANDLER`
- `put PATH -> HANDLER`
- `delete PATH -> HANDLER`

### Path Parameters
```python
get "/users/:id" -> get_user
get "/posts/:post_id/comments/:comment_id" -> get_comment
```

### Middleware
```python
# Global middleware
use logger

# Route-specific middleware
get "/api/data" -> handler, [cors, auth]
```

---

## Examples

### Full Auth Flow
```python
users = {}

def register(req):
    data = json.decode(req.body)
    hash = auth.hash_password(data["password"])
    users[data["username"]] = hash
    return response.json({"message": "Registered"}, 201)

def login(req):
    data = json.decode(req.body)
    hash = users[data["username"]]
    if auth.verify_password(hash, data["password"]):
        token = jwt.sign({"user": data["username"]}, "secret", "24h")
        return response.json({"token": token}, 200)
    return response.json({"error": "Invalid"}, 401)

service AuthAPI on ":8080":
    post "/register" -> register
    post "/login" -> login
```

### WebSocket Chat
```python
def chat(req):
    conn = websocket.upgrade(req)
    websocket.send(conn, "Welcome!")
    
    while True:
        msg = websocket.read(conn)
        if msg == None:
            break
        # Broadcast to all clients
        websocket.send(conn, "You: " + msg)
    
    websocket.close(conn)
    return None

service ChatServer on ":8080":
    get "/ws" -> chat
```

---

For more examples, see the `/examples` directory.
