# Flowa Quickstart

This guide gets you from clone → first pipeline in a few commands.

---

## 1. Install

### Option 1 · Recommended script

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
chmod +x install.sh
./install.sh
```

This builds and installs `flowa` to `/usr/local/bin`.

### Option 2 · Makefile

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
make install
```

### Option 3 · Manual

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
make build
sudo cp flowa /usr/local/bin/
```

---

## 2. Verify

```bash
flowa examples/hello.flowa
```

You should see a simple pipeline‑driven program execute.

---

## 3. Your first script

Create `demo.flowa`:

```python
func double(x){
    return x * 2
}

result = 5 |> double()
print(result)
```

Run it:

```bash
flowa demo.flowa
```

---

## 4. REPL

```bash
flowa repl
```

```text
Flowa REPL v0.1 (MVP)
>>> func double(x){
...     return x * 2
... }
>>> 5 |> double()
10
```

---

## 5. Tiny HTTP example

After installing, try the HTTP helper demo:

```bash
flowa examples/server.flowa
```

Then in another terminal:

```bash
curl http://localhost:8080/hello
```

---

## Uninstall

With make:

```bash
cd flowa
make uninstall
```

Or manually:

```bash
sudo rm /usr/local/bin/flowa
```
