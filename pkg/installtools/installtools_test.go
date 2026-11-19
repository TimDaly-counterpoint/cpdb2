package installtools

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	root, err := FindProjectRoot("")
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	goMod := filepath.Join(root, "go.mod")
	if info, err := os.Stat(goMod); err != nil || info.IsDir() {
		t.Fatalf("go.mod not found at located root %s: %v", root, err)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("hello install_tools test")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	copied, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read dst file: %v", err)
	}

	if !bytes.Equal(content, copied) {
		t.Fatalf("expected content %q, got %q", string(content), string(copied))
	}
}

func TestInstallTools(t *testing.T) {
	ctx := context.Background()
	root, err := FindProjectRoot("")
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	if err := InstallTools(ctx, root); err != nil {
		t.Fatalf("InstallTools failed: %v", err)
	}

	toolsDir := filepath.Join(root, "tools")
	info, err := os.Stat(toolsDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("tools directory was not created: %v", err)
	}

	expectedBinaries := []string{
		"cmatrix",
		"cpdb2",
		"disstable",
		"export_sequences",
		"export_smooth_xml",
		"find_tritones",
		"import_and_search",
		"import_scores",
		"process_sonority",
		"search_cross_piece",
		"search_sequences",
		"set_db",
		"smoothcp",
	}

	for _, bin := range expectedBinaries {
		binPath := filepath.Join(toolsDir, bin)
		st, err := os.Stat(binPath)
		if err != nil {
			t.Errorf("expected binary %s was not created: %v", bin, err)
			continue
		}
		// Verify executable permission
		if st.Mode()&0111 == 0 {
			t.Errorf("binary %s is not executable (mode %v)", bin, st.Mode())
		}
	}

	// Verify install_tools is NOT present in tools directory
	installToolsBin := filepath.Join(toolsDir, "install_tools")
	if _, err := os.Stat(installToolsBin); err == nil {
		t.Errorf("install_tools binary should not be present in tools directory")
	}

	// Verify copies of template.sql, voces.sql, and README.md
	expectedFiles := []string{"template.sql", "voces.sql", "README.md"}
	for _, f := range expectedFiles {
		fPath := filepath.Join(toolsDir, f)
		if _, err := os.Stat(fPath); err != nil {
			t.Errorf("expected support file %s was not copied to tools directory: %v", f, err)
		}
	}
}
