package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Grandbusta/jone/cmd/jone/templates"
	"github.com/Grandbusta/jone/internal/term"
)

type RunExecParams struct {
	Command string
	Flags   map[string]any
	Args    []string
}

// runMigrations generates a runner, builds it, and executes it with the given command.
func runMigrations(params RunExecParams) error {
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
		return fmt.Errorf("could not read module path. Ensure go.mod exists and contains a valid module declaration")
	}
	// Generate runner in .runner folder
	runnerDir := filepath.Join(cwd, JoneFolderPath, ".runner")
	if err := os.MkdirAll(runnerDir, 0o755); err != nil {
		return fmt.Errorf("creating .runner directory: %w", err)
	}

	// Cleanup .runner directory after execution
	defer func() {
		if err := os.RemoveAll(runnerDir); err != nil {
			fmt.Printf("Warning: failed to cleanup .runner directory: %v\n", err)
		}
	}()

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
	if err := executeRunner(binaryPath, params); err != nil {
		return fmt.Errorf("executing runner: %w", err)
	}

	return nil
}

func generateRunner(runnerPath, modulePath string) error {
	registryPackage := modulePath + "/" + MigrationsPath + "/registry"
	configPackage := modulePath + "/" + JoneFolderPath

	content, err := templates.RenderRunner(templates.RunnerData{
		RuntimePackage:  RuntimePackage,
		RegistryPackage: registryPackage,
		ConfigPackage:   configPackage,
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
		return fmt.Errorf("failed to compile migrations: %w. Try running 'go mod tidy' and check for syntax errors in your migration files", err)
	}

	return nil
}

func executeRunner(binaryPath string, params RunExecParams) error {
	fmt.Println(term.CyanText(fmt.Sprintf("Running migrations (%s)...", params.Command)))

	// Build command arguments: command + flags + args
	cmdArgs := []string{params.Command}

	// Add flags
	for key, value := range params.Flags {
		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--%s", key))
			}
		case string:
			if v != "" {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%s", key, v))
			}
		case int:
			cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%d", key, v))
		default:
			cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%v", key, v))
		}
	}

	// Add positional args
	cmdArgs = append(cmdArgs, params.Args...)

	runCmd := exec.Command(binaryPath, cmdArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("runner execution failed: %w", err)
	}

	return nil
}
