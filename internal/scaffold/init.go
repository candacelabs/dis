package scaffold

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/candacelabs/dis/internal/config"
)

// lookPath is a package-level variable so tests can override it.
var lookPath = func(name string) error {
	_, err := exec.LookPath(name)
	return err
}

// InitOptions controls Init behaviour.
type InitOptions struct {
	Force       bool
	ProjectType ProjectType // zero value ("") = auto-detect
}

// Init scaffolds a new dis project inside projectRoot.
//
// It creates .dis/Dockerfile (from a template matched to the detected or
// overridden project type) and a Justfile with standard dev recipes.
// If .dis/Dockerfile already exists and opts.Force is false, Init returns an error.
func Init(projectRoot string, opts InitOptions) error {
	// 1. Guard: refuse to overwrite an existing setup unless --force.
	if config.DockerfileExists(projectRoot) && !opts.Force {
		return errors.New("dis is already initialized. Use --force to overwrite.")
	}

	// 2. Resolve project type.
	pt := opts.ProjectType
	if pt == "" {
		pt = DetectProjectType(projectRoot)
	}

	// 3. Generate Dockerfile content.
	dockerfile := DockerfileTemplate(pt)

	// 4. Ensure .dis/ directory exists.
	disDir := filepath.Join(projectRoot, ".dis")
	if err := os.MkdirAll(disDir, 0755); err != nil {
		return fmt.Errorf("creating .dis directory: %w", err)
	}

	// 5. Write .dis/Dockerfile.
	dockerfilePath := filepath.Join(disDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("writing .dis/Dockerfile: %w", err)
	}

	// 6. Handle Justfile — only generate if just is installed.
	justfilePath := filepath.Join(projectRoot, "Justfile")
	createdJustfile := false
	hasJust := lookPath("just") == nil

	if hasJust {
		justfileExists := false
		if info, err := os.Stat(justfilePath); err == nil && !info.IsDir() {
			justfileExists = true
		}

		if justfileExists && !opts.Force {
			fmt.Fprintln(os.Stderr, "Justfile already exists, skipping.")
		} else {
			justfileContent := GenerateJustfile()
			if err := os.WriteFile(justfilePath, []byte(justfileContent), 0644); err != nil {
				return fmt.Errorf("writing Justfile: %w", err)
			}
			createdJustfile = true
		}
	}

	// 7. Print summary.
	fmt.Printf("✅ dis initialized!\n")
	fmt.Printf("   Project type: %s\n", pt)
	fmt.Printf("   Created: .dis/Dockerfile\n")
	if createdJustfile {
		fmt.Printf("   Created: Justfile\n")
	}
	fmt.Println()
	fmt.Println("   Next steps:")
	if hasJust {
		fmt.Println("     just dev     # start dev environment")
		fmt.Println("     just shell   # enter container")
	} else {
		fmt.Println("     dis up       # start dev environment")
		fmt.Println("     dis run <cmd> # run a command in the container")
		fmt.Println("     dis shell    # enter container")
		fmt.Println()
		fmt.Println("   Tip: install just (https://github.com/casey/just) for convenient aliases.")
	}

	return nil
}
