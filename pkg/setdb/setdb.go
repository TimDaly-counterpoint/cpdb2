package setdb

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cpdb2/pkg/db"
	"cpdb2/pkg/prompt"
)

// FindSQLFile searches for a SQL file by name in the working directory,
// executable directory, and parent directories.
func FindSQLFile(filename string) (string, error) {
	// 1. Check current working directory
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, filename)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 2. Check executable directory
	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, filename)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 3. Fallback to direct relative path
	if info, err := os.Stat(filename); err == nil && !info.IsDir() {
		return filename, nil
	}

	return "", fmt.Errorf("could not find required file %q in working directory or parent directories", filename)
}

// SetDB connects to the named database, validates that it exists and is empty,
// imports the schema from template.sql, and populates the voces and intervala tables from voces.sql.
func SetDB(ctx context.Context, dbName string) error {
	trimmed := strings.TrimSpace(dbName)
	if trimmed == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	templatePath, err := FindSQLFile("template.sql")
	if err != nil {
		return err
	}

	vocesPath, err := FindSQLFile("voces.sql")
	if err != nil {
		return err
	}

	// 1. Connect to named database
	database, err := db.NewDB(ctx, trimmed)
	if err != nil {
		return fmt.Errorf("failed to connect to database %q (does it exist?): %w", trimmed, err)
	}
	defer database.Close()

	// 2. Check if database is empty
	var tableCount int
	err = database.Pool.QueryRow(ctx, `
		SELECT count(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
	`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to inspect database %q tables: %w", trimmed, err)
	}

	if tableCount > 0 {
		return fmt.Errorf("database %q is not empty: contains %d existing table(s) in public schema", trimmed, tableCount)
	}

	// 3. Import schema from template.sql and populate voces and intervala from voces.sql
	connStr := db.ResolveConnString(trimmed)

	cmdTemplate := exec.CommandContext(ctx, "psql", connStr, "-v", "ON_ERROR_STOP=1", "-f", templatePath)
	if out, err := cmdTemplate.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to import schema from %s: %w\nOutput: %s", templatePath, err, strings.TrimSpace(string(out)))
	}

	cmdVoces := exec.CommandContext(ctx, "psql", connStr, "-f", vocesPath)
	if out, err := cmdVoces.CombinedOutput(); err != nil {
		// Log notice if needed but check table population below
		_ = out
	}

	// 4. Verify voces table population
	var vocesCount int
	err = database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.voces").Scan(&vocesCount)
	if err != nil {
		return fmt.Errorf("failed to verify voces table in database %q: %w", trimmed, err)
	}
	if vocesCount == 0 {
		return fmt.Errorf("voces table was not populated (0 rows found)")
	}

	// 5. Verify intervala table population
	var intervalaCount int
	err = database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.intervala").Scan(&intervalaCount)
	if err != nil {
		return fmt.Errorf("failed to verify intervala table in database %q: %w", trimmed, err)
	}
	if intervalaCount == 0 {
		return fmt.Errorf("intervala table was not populated (0 rows found)")
	}

	fmt.Printf("Database %q successfully initialized with %d voces and %d intervala records.\n", trimmed, vocesCount, intervalaCount)
	return nil
}

// RunInteractive prompts the user for a database name and runs SetDB.
func RunInteractive(ctx context.Context, reader *bufio.Reader) error {
	dbName, err := prompt.PromptString(reader, "Enter name of empty database to configure", "")
	if err != nil {
		return err
	}
	dbName = strings.TrimSpace(dbName)
	if dbName == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	return SetDB(ctx, dbName)
}
