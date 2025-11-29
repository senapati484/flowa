# Flowa Language Documentation

Welcome to the official documentation for **Flowa**, the pipeline-first programming language.

## 🚀 Getting Started

### Running Code
There are three ways to run Flowa code:

1.  **Run a Script**:
    ```bash
    flowa my_script.flowa
    ```

2.  **Interactive REPL**:
    ```bash
    flowa repl
    ```

3.  **Evaluate Expression**:
    ```bash
    flowa eval 'print(10 |> double())'
    ```

---

## 🔑 Keywords & Syntax

### Functions (`def`)
Define functions using `def` and indentation.
```python
def add(x, y):
    return x + y
```

### Pipeline Operator (`|>`)
Pass the result of the left expression as the *first argument* to the function on the right.
```python
# Equivalent to: square(increment(5))
result = 5 |> increment() |> square()
```

### Return (`return`)
Return a value from a function.
```python
def greet(name):
    return "Hello " + name
```

### Assignments
Assign values to variables.
```python
x = 10
y = x * 2
```

---

## 🛠 Built-in Functions

Flowa comes with a set of useful built-in functions available globally.

### `print(args...)`
Prints values to the standard output.
```python
print("Hello World")
print(10, 20, 30)
```

### `input(prompt)`
Reads a line of text from the user.
```python
print("Enter your name:")
name = input()
```

### `type(object)`
Returns the type of an object (e.g., `INTEGER`, `FUNCTION`).
```python
t = type(123)
print(t)  # Output: INTEGER
```

### `exit(code)`
Terminates the program with the specified exit code.
```python
exit(0)  # Success
exit(1)  # Error
```

---

## 🔀 Control Flow

### `if` / `elif` / `else`
Conditional branching with Pythonic syntax.

```python
x = 10
if x > 10:
    print("Greater than 10")
elif x == 10:
    print("Equal to 10")
else:
    print("Less than 10")
```

---

## 💎 Unique Flowa Keywords

These built-in functions are designed specifically for the **Pipeline-First** philosophy.

### `tap(function)`
Executes a function with the current value as an argument, but returns the *original value*. This allows you to "tap" into a pipeline for side effects (like logging) without breaking the flow.

**Demo:**
```python
def double(x):
    return x * 2

# Print the value '5' but pass '5' to double()
result = 5 |> tap(print) |> double()
# Output: 5
# result = 10
```

### `inspect()`
Prints debug information (Type and Value) about the current item in the pipeline and passes it through unchanged.

**Demo:**
```python
# Debug the intermediate value
result = 10 |> double() |> inspect() |> double()
# Output: [DEBUG] Type: INTEGER, Value: 20
# result = 40
```

---

## 🧮 Utility Functions

### `min(a, b)` / `max(a, b)`
Returns the minimum or maximum of two integers.

```python
low = min(10, 20)  # 10
high = max(10, 20) # 20
```

---

## 🧮 Operators

| Operator | Description | Example |
| :--- | :--- | :--- |
| `+` | Addition | `10 + 5` |
| `-` | Subtraction | `10 - 5` |
| `*` | Multiplication | `10 * 5` |
| `/` | Division | `10 / 2` |
| `( )` | Grouping | `(10 + 5) * 2` |

---

## 📝 Example

Here is a complete example combining these features:

```python
def square(x):
    return x * x

def main():
    print("Calculating square...")
    result = 10 |> square()
    print("Result:", result)

main()
```
