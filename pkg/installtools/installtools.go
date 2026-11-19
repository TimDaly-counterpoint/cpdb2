package installtools

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// FindProjectRoot locates the root of the project by searching upwards for go.mod.
func FindProjectRoot(startDir string) (string, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			startDir = cwd
		}
	}

	// 1. Search upwards from startDir / cwd
	if startDir != "" {
		dir := startDir
		for i := 0; i < 10; i++ {
			candidate := filepath.Join(dir, "go.mod")
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 2. Search upwards from executable directory
	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		for i := 0; i < 10; i++ {
			candidate := filepath.Join(dir, "go.mod")
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", fmt.Errorf("could not find project root containing go.mod")
}

// CopyFile copies a single file from src to dst.
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", src, err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source file %q: %w", src, err)
	}

	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("create destination file %q: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy from %q to %q: %w", src, dst, err)
	}

	return nil
}

// InstallTools compiles all available tools in cmd/ (except install_tools),
// places their executables into the tools/ folder, and copies template.sql,
// voces.sql, and README.md into tools/.
func InstallTools(ctx context.Context, rootDir string) error {
	projectRoot, err := FindProjectRoot(rootDir)
	if err != nil {
		return err
	}

	toolsDir := filepath.Join(projectRoot, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory %q: %w", toolsDir, err)
	}

	cmdDir := filepath.Join(projectRoot, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return fmt.Errorf("failed to read cmd directory %q: %w", cmdDir, err)
	}

	var toolNames []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "install_tools" {
			continue
		}
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	fmt.Printf("Compiling %d tool(s) to %s...\n", len(toolNames), toolsDir)
	for _, tool := range toolNames {
		targetBin := filepath.Join(toolsDir, tool)
		pkgPath := fmt.Sprintf("./cmd/%s", tool)

		fmt.Printf("  -> Building %s (%s)...\n", tool, pkgPath)
		cmd := exec.CommandContext(ctx, "go", "build", "-o", targetBin, pkgPath)
		cmd.Dir = projectRoot
		cmd.Env = os.Environ()

		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to build tool %q: %w\nOutput: %s", tool, err, strings.TrimSpace(string(out)))
		}

		if err := os.Chmod(targetBin, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions on %q: %w", targetBin, err)
		}
	}

	// Copy template.sql, voces.sql, and README.md into tools/
	filesToCopy := []string{"template.sql", "voces.sql", "README.md"}
	for _, fileName := range filesToCopy {
		srcPath := filepath.Join(projectRoot, fileName)
		dstPath := filepath.Join(toolsDir, fileName)

		if _, err := os.Stat(srcPath); err != nil {
			return fmt.Errorf("required file %q not found in project root: %w", fileName, err)
		}

		fmt.Printf("  -> Copying %s to %s...\n", fileName, dstPath)
		if err := CopyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	fmt.Println("All tools and support files installed successfully in:", toolsDir)
	return nil
}
