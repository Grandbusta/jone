package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadModulePath extracts the module path from go.mod in the given project root.
// Returns an empty string if go.mod doesn't exist or can't be read.
func ReadModulePath(projectRoot string) string {
	f, err := os.Open(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

// FindOrCreateGoMod checks for go.mod and optionally creates it interactively.
// Returns the path to go.mod on success, or an empty string on failure/abort.
func FindOrCreateGoMod(cwd string) string {
	goModPath := filepath.Join(cwd, "go.mod")

	if _, err := os.Stat(goModPath); err == nil {
		return goModPath
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Error checking go.mod: %v\n", err)
		return ""
	}

	// go.mod doesn't exist - prompt user
	fmt.Println("No go.mod found in the current directory.")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Initialize a Go module now? (y/N): ")
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(strings.ToLower(resp))

	if resp != "y" && resp != "yes" {
		fmt.Println("Please run: go mod init <module> to continue.")
		return ""
	}

	fmt.Print("Enter module path (e.g., github.com/yourname/project): ")
	modPath, _ := reader.ReadString('\n')
	modPath = strings.TrimSpace(modPath)

	if modPath == "" {
		fmt.Println("Module path is required; aborting.")
		return ""
	}

	initCmd := exec.Command("go", "mod", "init", modPath)
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	initCmd.Stdin = os.Stdin

	if err := initCmd.Run(); err != nil {
		fmt.Printf("Failed to initialize Go module: %v\n", err)
		return ""
	}

	fmt.Println("Go module initialized.")
	return goModPath
}

// JoneDependencyPresent checks whether go.mod lists the jone module as a dependency.
func JoneDependencyPresent(modPath string) bool {
	f, err := os.Open(modPath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Enter a require block
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		// Exit a require block
		if inRequireBlock && strings.HasPrefix(line, ")") {
			inRequireBlock = false
			continue
		}

		// Process lines inside the require block
		if inRequireBlock {
			if line == "" || strings.HasPrefix(line, "//") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 1 && fields[0] == RuntimePackage {
				return true
			}
			continue
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "require"))
			fields := strings.Fields(rest)
			if len(fields) >= 1 && fields[0] == RuntimePackage {
				return true
			}
		}
	}

	return false
}
