package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// If go.mod file exists, confirm if jone is already installed, if not, ask the user to install jone with go install github.com/Grandbusta/jone@latest

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a jone project",
	Long:  `Initializes a jone project`,
	Run:   initJone,
}

func initJone(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	modFilePath := findOrCreateGoMod(cwd)

	if modFilePath == "" {
		return
	}

	if !joneDependencyPresent(modFilePath) {
		fmt.Println("jone is not installed in this project.")
		fmt.Println("To add it, run: go get github.com/Grandbusta/jone")
	}
	createJoneFolderAndFiles(cwd)
	fmt.Println("jone init complete.")
}

// Checks if there is a go.mod file, if not, ask the user to setup a go module
func findOrCreateGoMod(cwd string) (modFilePath string) {
	if _, err := os.Stat("go.mod"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No go.mod found in the current directory.")
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Initialize a Go module now? (y/N): ")
			resp, _ := reader.ReadString('\n')
			resp = strings.TrimSpace(strings.ToLower(resp))
			if resp == "y" || resp == "yes" {
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
				if err = initCmd.Run(); err != nil {
					fmt.Printf("Failed to initialize Go module: %v\n", err)
					return ""
				}
				fmt.Println("Go module initialized.")
				return cwd + "/go.mod"
			} else {
				fmt.Println("Please run: go mod init <module> to continue.")
				return ""
			}
		} else {
			fmt.Printf("Error checking go.mod: %v\n", err)
			return ""
		}
	} else {
		return cwd + "/go.mod"
	}
}

// Checks whether the current project's go.mod lists the jone module.
func joneDependencyPresent(modPath string) bool {
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
			if len(fields) >= 1 && fields[0] == "github.com/Grandbusta/jone" {
				return true
			}
			continue
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "require"))
			fields := strings.Fields(rest)
			if len(fields) >= 1 && fields[0] == "github.com/Grandbusta/jone" {
				return true
			}
		}
	}

	return false
}

func createJoneFolderAndFiles(cwd string) {
	// Create jone folder
	joneFolderPath := cwd + "/jone"
	if _, err := os.Stat(joneFolderPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.Mkdir(joneFolderPath, 0755)
		} else {
			fmt.Printf("Error checking jone folder: %v\n", err)
			return
		}
	}

	// Create a jonefile.go file
	joneFilePath := joneFolderPath + "/jonefile.go"
	if _, err := os.Stat(joneFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.Create(joneFilePath)
		} else {
			fmt.Printf("Error checking jonefile.go: %v\n", err)
			return
		}
	}

	// Write jonefile.go contents
	file, err := os.OpenFile(joneFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("Error opening jonefile.go: %v\n", err)
		return
	}
	defer file.Close()

	joneFileContents := `package jone

import "github.com/Grandbusta/jone"

var Config = jone.Config{
	User: "root",
	Pass: "root",
	Host: "localhost",
	Port: 3306,
	DB:   "jone",
}
	`
	_, err = file.WriteString(joneFileContents)
	if err != nil {
		fmt.Printf("Error writing to jonefile.go: %v\n", err)
		return
	}

}
