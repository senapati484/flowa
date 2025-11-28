# Flowa Installation Guide for macOS

## Method 1: Using Homebrew (Recommended for Mac)

### Step 1: Tap the Repository (Future)
Once published to Homebrew:
```bash
brew tap senapati484/flowa
brew install flowa
```

### Step 2: Local Formula Installation (Current)
For now, install from local formula:
```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
brew install --formula ./flowa.rb
```

After installation:
```bash
flowa examples/hello.flowa
flowa repl
```

## Method 2: Automatic Installer Script

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
chmod +x install.sh
./install.sh
```

This will:
- Check for Go installation
- Build the `flowa` binary
- Install to `/usr/local/bin`
- Make it available globally

## Method 3: Using Make

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
make install
```

## Method 4: Manual Installation

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
go build -o flowa ./cmd/flowa
sudo cp flowa /usr/local/bin/
sudo chmod +x /usr/local/bin/flowa
```

## Verify Installation

```bash
which flowa
flowa examples/hello.flowa
```

## Uninstall

### If installed via Homebrew:
```bash
brew uninstall flowa
```

### If installed via script/Make:
```bash
cd flowa
make uninstall
```

Or manually:
```bash
sudo rm /usr/local/bin/flowa
```

## Troubleshooting

### Go not installed?
```bash
brew install go
```

### Permission denied?
The installer will automatically request sudo access when needed.

### Command not found after installation?
Make sure `/usr/local/bin` is in your PATH:
```bash
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```
