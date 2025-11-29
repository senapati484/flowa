# Flowa Language Documentation

Welcome to the reference documentation for **Flowa**, the pipeline‑first
language. This document focuses on the _language itself_; for installation and
CLI usage, see `QUICKSTART.md` and `README.md`.

---

## 1. Core Concepts

- **Indentation‑based syntax** – Blocks are defined by indentation, not braces.
- **Expressions everywhere** – Most constructs are expression‑oriented and can
  be combined freely.
- **Pipeline operator (`|>`)** – Passes the value on the left as the **first
  argument** to the function on the right.

### 1.1 Functions (`def`)

```python
def add(x, y):
    return x + y

result = add(5, 7)
print(result)
```

### 1.2 Pipeline operator (`|>`)

```python
def increment(x):
    return x + 1

def square(x):
    return x * x

# Equivalent to: square(increment(5))
result = 5 |> increment() |> square()
print(result)  # 36
```

### 1.3 Assignments & values

```python
x = 10
y = 20
name = "Flowa"
is_active = True
nothing = None
```

---

## 2. Control Flow

### 2.1 `if` / `elif` / `else`

```python
x = 10

if x > 10:
    print("Greater than 10")
elif x == 10:
    print("Equal to 10")
else:
    print("Less than 10")
```

### 2.2 `while`

```python
i = 0
while i < 3:
    print(i)
    i = i + 1
```

### 2.3 `for` with `range`

```python
for i in range(3):
    print(i)
```

The built‑in `range(n)` returns an array `[0, 1, ..., n-1]`.

---

## 3. Data Types

- **INTEGER** – 64‑bit integers.
- **STRING** – UTF‑8 strings.
- **BOOLEAN** – `True` / `False`.
- **NULL** – `None` in source maps to a `NULL` value.
- **ARRAY** – Ordered list of values.
- **MAP** – Key/value map.
- **FUNCTION** / **BUILTIN** – User and built‑in functions.
- **TASK** – Result of `spawn` (lightweight async surface).
- **STRUCT_INSTANCE** – Simple records created by `type`.
- **MODULE** – Values created by `module` declarations.

### 3.1 Arrays

```python
nums = [1, 2, 3]
print(len(nums))  # 3
first = first(nums)
last_item = last(nums)
rest_items = rest(nums)
```

### 3.2 Maps

```python
user = { "name": "Ada", "age": 32 }
print(len(user))  # 2
```

### 3.3 Structs (`type`)

```python
type Point:
    x
    y

p = Point(10, 20)
print(p)  # Point(x=10, y=20)
```

### 3.4 Modules (`module`)

```python
module Math:
    def square(n):
        return n * n

    pi = 3

print(Math.pi)
```

---

## 4. Built‑ins

### 4.1 Core

- **`print(args...)`** – Print any number of values.
- **`len(x)`** – Length of string, array, or map.
- **`first(array)` / `last(array)` / `rest(array)`** – Basic array helpers.
- **`push(array, value)`** – Returns a new array with `value` appended.
- **`range(n)`** – Returns `[0, 1, ..., n‑1]`.

### 4.2 Utility & debugging

- **`min(a, b)` / `max(a, b)`** – Integer min/max.
- **`tap(fn)`** – In pipelines, call `fn` with the current value but return
  the original value.
- **`inspect(x)`** – Prints `[DEBUG] Type: T, Value: V` and returns `x`.

Example:

```python
def double(x):
    return x * 2

value = 5 |> tap(print) |> double() |> inspect()
```

### 4.3 Async surface (`spawn` / `await`)

Flowa exposes a minimal async surface. In the current interpreter tasks are
evaluated eagerly, but the syntax is future‑proofed for a richer runtime.

```python
async def async_task(n):
    return n * 2

task = spawn async_task(10)
result = await task
print(result)
```

### 4.4 HTTP helpers

The `examples/server.flowa` script showcases tiny HTTP helpers that make it
easy to spin up demo endpoints:

- **`response(status, body)`** – Construct a simple HTTP response object.
- **`route(method, path, handler)`** – Register a request handler.
- **`listen(port)`** – Start an HTTP server on the given port.

```python
def hello(req):
    return response(200, "Hello World")

route("GET", "/hello", hello)
listen(8080)
```

---

## 5. Operators

| Operator          | Description      | Example           |
| ----------------- | ---------------- | ----------------- |
| `+`               | Addition         | `10 + 5`          |
| `-`               | Subtraction      | `10 - 5`          |
| `*`               | Multiplication   | `10 * 5`          |
| `/`               | Division         | `10 / 2`          |
| `==`              | Equality         | `x == 10`         |
| `!=`              | Inequality       | `x != y`          |
| `<` `>` `<=` `>=` | Comparisons      | `x < y`, `x >= y` |
| `!`               | Boolean negation | `!False`          |
| `( )`             | Grouping         | `(10 + 5) * 2`    |

---

## 6. Putting It Together

```python
print("Assignments:")
x = 10
y = 20
print(x, y)

print("Pipelines:")
def double(n):
    return n * 2

def increment(n):
    return n + 1

res = 5 |> double() |> increment()
print(res)

print("Loops:")
for i in range(3):
    print(i)
```

For more end‑to‑end examples, explore the `examples/` directory.
