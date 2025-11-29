package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	customPath := flag.String("path", "", "Custom install directory")
	flag.Parse()

	repoRoot, err := os.Getwd()
	if err != nil {
		exitWithError("unable to determine working directory", err)
	}

	binaryName := "flowa"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	buildOutput := filepath.Join(repoRoot, binaryName)

	fmt.Println("🚧 Building Flowa CLI...")
	buildCmd := exec.Command("go", "build", "-o", buildOutput, "./cmd/flowa")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	buildCmd.Dir = repoRoot
	if err := buildCmd.Run(); err != nil {
		exitWithError("Go build failed", err)
	}

	targetDir := *customPath
	if targetDir == "" {
		targetDir = defaultInstallDir()
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		os.Remove(buildOutput)
		exitWithError("unable to create install directory", err)
	}

	destPath := filepath.Join(targetDir, binaryName)
	fmt.Printf("📦 Installing to %s\n", destPath)

	if err := copyFile(buildOutput, destPath); err != nil {
		os.Remove(buildOutput)
		exitWithError("failed to copy binary (try running with elevated permissions)", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0o755); err != nil {
			os.Remove(buildOutput)
			exitWithError("failed to set executable bit", err)
		}
	}

	os.Remove(buildOutput)

	fmt.Println("✅ Flowa installed successfully!")
	fmt.Println("Run 'flowa --help' to verify the CLI is available in your PATH.")
}

func defaultInstallDir() string {
	switch runtime.GOOS {
	case "windows":
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, "Programs", "Flowa")
		}
		return filepath.Join(os.TempDir(), "Flowa")
	default:
		return "/usr/local/bin"
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func exitWithError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}
