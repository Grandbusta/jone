package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Grandbusta/jone/cmd/jone/templates"
)

// runMigrations generates a runner, builds it, and executes it with the given command.
func runMigrations(command string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check jone folder exists
	if _, err := os.Stat(JoneFolderPath); os.IsNotExist(err) {
		return fmt.Errorf("jone folder not found, please run jone init first")
	}

	// Get module path for imports
	modulePath := ReadModulePath(cwd)
	if modulePath == "" {
		return fmt.Errorf("could not read module path from go.mod")
	}
	// Generate runner in .runner folder
	runnerDir := filepath.Join(cwd, JoneFolderPath, ".runner")
	if err := os.MkdirAll(runnerDir, 0o755); err != nil {
		return fmt.Errorf("creating .runner directory: %w", err)
	}

	runnerPath := filepath.Join(runnerDir, "main.go")
	binaryPath := filepath.Join(runnerDir, "runner")

	if err := generateRunner(runnerPath, modulePath); err != nil {
		return fmt.Errorf("generating runner: %w", err)
	}

	// Build runner
	if err := buildRunner(cwd, runnerPath, binaryPath); err != nil {
		return fmt.Errorf("building runner: %w", err)
	}

	// Execute runner
	if err := executeRunner(binaryPath, command); err != nil {
		return fmt.Errorf("executing runner: %w", err)
	}

	return nil
}

func generateRunner(runnerPath, modulePath string) error {
	registryPackage := modulePath + "/" + MigrationsPath + "/registry"

	content, err := templates.RenderRunner(templates.RunnerData{
		RuntimePackage:  RuntimePackage,
		RegistryPackage: registryPackage,
	})
	if err != nil {
		return fmt.Errorf("rendering runner template: %w", err)
	}

	if err := os.WriteFile(runnerPath, content, 0o644); err != nil {
		return fmt.Errorf("writing runner file: %w", err)
	}

	return nil
}

func buildRunner(cwd, runnerPath, binaryPath string) error {
	buildCmd := exec.Command("go", "build", "-o", binaryPath, runnerPath)
	buildCmd.Dir = cwd
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	return nil
}

func executeRunner(binaryPath, command string) error {
	fmt.Printf("Running migrations (%s)...\n", command)

	runCmd := exec.Command(binaryPath, command)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("runner execution failed: %w", err)
	}

	return nil
}
