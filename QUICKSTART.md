# Flowa Quick Start Guide

## Installation

### Option 1: Automatic Installation (Recommended)

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
chmod +x install.sh
./install.sh
```

This will automatically build and install `flowa` to `/usr/local/bin`.

### Option 2: Using Make

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
make install
```

### Option 3: Manual Installation

```bash
git clone https://github.com/senapati484/flowa.git
cd flowa
make build
sudo cp flowa /usr/local/bin/
```

## Verify Installation

```bash
flowa examples/hello.flowa
```

## Basic Usage

### Run a script

```bash
flowa myscript.flowa
```

### Start REPL

```bash
flowa repl
```

## Uninstall

```bash
cd flowa
make uninstall
```

Or manually:

```bash
sudo rm /usr/local/bin/flowa
```
