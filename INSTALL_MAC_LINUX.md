# Flowa for macOS & Linux

## macOS Installation Methods

### Method 1: Homebrew (Recommended for macOS)

```bash
# Add the Flowa tap (when published)
brew tap senapati484/flowa
brew install flowa

# Or install from local formula
brew install --formula ./flowa.rb

# Or uninstall from Homebrew
brew uninstall flowa
```

### Method 2: Universal Installer Script

```bash
# Clone the repository
git clone https://github.com/senapati484/flowa.git
cd flowa

# Make the installer executable and run it
chmod +x install.sh
./install.sh

# Or uninstall using the CLI
flowa uninstall
```

### Method 3: Using Make

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
make install  # Installs to /usr/local/bin by default
```

## Linux Installation Methods

### Method 1: Standalone Installer

```bash
# Download the latest release
curl -L https://github.com/senapati484/flowa/releases/latest/download/flowa-linux-amd64 -o flowa
chmod +x flowa
sudo mv flowa /usr/local/bin/
```

### Method 2: Build from Source

```bash
# Install Go if needed
sudo apt-get update && sudo apt-get install -y golang

# Build and install
make build
sudo make install
```

## Verifying Installation

```bash
flowa --version
flowa run examples/hello.flowa
```

## Uninstalling

### Using the CLI:

```bash
flowa uninstall
# Or with sudo if installed to /usr/local/bin
sudo flowa uninstall
```

### Manual Removal:

```bash
# For Homebrew
brew uninstall flowa

# For manual installs
sudo rm /usr/local/bin/flowa
```

## Troubleshooting

### Permission Denied

- Use `sudo` when installing to system directories
- Or install to a user-writable location (e.g., `~/.local/bin`)

### Command Not Found

- Ensure `/usr/local/bin` (or your install directory) is in your PATH
- Try opening a new terminal window

### Go Version

- Requires Go 1.16 or later
- Check with `go version`

---

For Windows installation, see [INSTALL_WINDOWS.md](INSTALL_WINDOWS.md)
