package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Grandbusta/jone/cmd/jone/templates"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a jone project",
	Long:  `Initializes a jone project with the required folder structure and configuration`,
	Run:   initJone,
}

func initJone(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	modFilePath := FindOrCreateGoMod(cwd)
	if modFilePath == "" {
		return
	}

	if !JoneDependencyPresent(modFilePath) {
		fmt.Println("jone is not installed in this project.")
		fmt.Printf("To add it, run: go get %s\n", RuntimePackage)
	}

	if err := createJoneFolderAndFiles(cwd); err != nil {
		fmt.Printf("Error creating jone files: %v\n", err)
		return
	}

	fmt.Println("jone init complete.")
}

func createJoneFolderAndFiles(cwd string) error {
	joneFolderPath := filepath.Join(cwd, JoneFolderPath)

	// Create jone folder
	if _, err := os.Stat(joneFolderPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(joneFolderPath, 0755); err != nil {
				return fmt.Errorf("creating jone folder: %w", err)
			}
		} else {
			return fmt.Errorf("checking jone folder: %w", err)
		}
	}

	// Create jonefile.go
	joneFilePath := filepath.Join(cwd, JoneFilePath)
	if _, err := os.Stat(joneFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := writeJoneFile(joneFilePath); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("checking jonefile.go: %w", err)
		}
	}

	return nil
}

func writeJoneFile(path string) error {
	content, err := templates.RenderJoneFile(templates.JoneFileData{
		RuntimePackage: RuntimePackage,
	})
	if err != nil {
		return fmt.Errorf("rendering jonefile template: %w", err)
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("writing jonefile.go: %w", err)
	}

	return nil
}
