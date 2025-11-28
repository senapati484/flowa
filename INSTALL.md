# Flowa Installation Options

## Quick Summary

| Platform | Method | Command |
|----------|--------|---------|
| **macOS** | Homebrew (recommended) | `brew install --formula ./flowa.rb` |
| **macOS/Linux** | Auto installer | `./install.sh` |
| **Any** | Make | `make install` |
| **Any** | Manual | `go build -o flowa ./cmd/flowa && sudo cp flowa /usr/local/bin/` |

## Installation Files Created

1. **install.sh** - Universal bash installer for Linux/macOS
2. **Makefile** - Build and install targets  
3. **flowa.rb** - Homebrew formula for macOS
4. **INSTALL_MAC.md** - Detailed Mac installation guide
5. **QUICKSTART.md** - Quick start guide

## Testing Your Installation

After installing, run:

```bash
# Check it's installed
which flowa

# Run an example
flowa examples/hello.flowa

# Start REPL
flowa repl
```

## Usage

```bash
# Run a script (simple syntax)
flowa myscript.flowa

# Or explicit run command
flowa run myscript.flowa

# Interactive REPL
flowa repl
```

## Uninstall

```bash
# If installed via make/script
make uninstall

# If installed via Homebrew
brew uninstall flowa

# Manual
sudo rm /usr/local/bin/flowa
```
