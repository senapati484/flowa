# Flowa for Windows

## Installation Methods

### Method 1: Standalone Installer (Recommended)

1. Download the latest `flowa-installer.exe` from the [Releases](https://github.com/senapati484/flowa/releases) page
2. Run the installer as Administrator:
   ```powershell
   .\flowa-installer.exe --path "$env:ProgramFiles\Flowa"
   ```
3. Add to PATH (if not done automatically):
   ```powershell
   [Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User) + ";$env:ProgramFiles\Flowa", [EnvironmentVariableTarget]::User)
   ```

### Method 2: Using PowerShell Script

```powershell
# Clone the repository
git clone https://github.com/senapati484/flowa.git
cd flowa

# Run the Windows installer script
pwsh -File scripts/install-windows.ps1
```

### Method 3: Manual Installation

```powershell
# Build the binary
go build -o flowa.exe ./cmd/flowa

# Create installation directory
$installDir = "$env:LOCALAPPDATA\Programs\Flowa"
New-Item -ItemType Directory -Force -Path $installDir

# Copy binary
Copy-Item flowa.exe "$installDir\flowa.exe"

# Add to PATH (current session)
$env:PATH = "$installDir;" + $env:PATH

# Make persistent (requires admin)
[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User) + ";$installDir", [EnvironmentVariableTarget]::User)
```

## Verifying Installation

Open a new PowerShell window and run:

```powershell
flowa --version
flowa examples/hello.flowa
```

## Uninstalling

### Using the CLI (if in PATH):

```powershell
flowa uninstall
```

### Manual Removal:

1. Delete the installation directory:
   ```powershell
   Remove-Item -Recurse -Force "$env:ProgramFiles\Flowa"
   # OR
   Remove-Item -Recurse -Force "$env:LOCALAPPDATA\Programs\Flowa"
   ```
2. Remove from PATH (if needed)

## Troubleshooting

### Command not found

- Ensure the installation directory is in your PATH
- Open a new terminal window after installation

### Permission denied

- Run PowerShell as Administrator when installing to protected locations
- Or choose a user-writable directory (like `$env:LOCALAPPDATA\Programs\Flowa`)

### Go not found

- Install Go from [golang.org/dl](https://golang.org/dl/)
- Ensure `go` is in your PATH

---

For other platforms, see [INSTALL_MAC_LINUX.md](INSTALL_MAC_LINUX.md)
