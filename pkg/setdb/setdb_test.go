package setdb

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"cpdb2/pkg/db"
)

func TestFindSQLFile(t *testing.T) {
	templatePath, err := FindSQLFile("template.sql")
	if err != nil {
		t.Fatalf("FindSQLFile(template.sql) failed: %v", err)
	}
	if info, err := os.Stat(templatePath); err != nil || info.IsDir() {
		t.Fatalf("template.sql path invalid: %s (%v)", templatePath, err)
	}

	vocesPath, err := FindSQLFile("voces.sql")
	if err != nil {
		t.Fatalf("FindSQLFile(voces.sql) failed: %v", err)
	}
	if info, err := os.Stat(vocesPath); err != nil || info.IsDir() {
		t.Fatalf("voces.sql path invalid: %s (%v)", vocesPath, err)
	}

	_, err = FindSQLFile("nonexistent_file_12345.sql")
	if err == nil {
		t.Fatalf("expected error for nonexistent file, got nil")
	}
}

func TestSetDB_EmptyArg(t *testing.T) {
	ctx := context.Background()
	if err := SetDB(ctx, "   "); err == nil {
		t.Fatalf("expected error for empty db name, got nil")
	}
}

func TestSetDB_NonExistentDB(t *testing.T) {
	ctx := context.Background()
	err := SetDB(ctx, "nonexistent_db_987654321")
	if err == nil {
		t.Fatalf("expected error for non-existent database, got nil")
	}
}

func TestSetDB_Integration(t *testing.T) {
	testDBName := "test_set_db_temp"
	ctx := context.Background()

	// Helper to run psql command against postgres DB
	postgresConn := db.ResolveConnString("postgres")
	_ = exec.Command("psql", postgresConn, "-c", "DROP DATABASE IF EXISTS "+testDBName+";").Run()

	cmdCreate := exec.Command("psql", postgresConn, "-c", "CREATE DATABASE "+testDBName+";")
	if out, err := cmdCreate.CombinedOutput(); err != nil {
		t.Skipf("Skipping integration test: cannot create test database: %v (%s)", err, string(out))
	}
	defer func() {
		_ = exec.Command("psql", postgresConn, "-c", "DROP DATABASE IF EXISTS "+testDBName+";").Run()
	}()

	// 1. First run on empty DB should succeed
	if err := SetDB(ctx, testDBName); err != nil {
		t.Fatalf("SetDB on empty database failed: %v", err)
	}

	// Verify table population
	database, err := db.NewDB(ctx, testDBName)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	defer database.Close()

	var vocesCount int
	if err := database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.voces").Scan(&vocesCount); err != nil {
		t.Fatalf("failed to query voces count: %v", err)
	}
	if vocesCount != 30 {
		t.Errorf("expected 30 voces rows, got %d", vocesCount)
	}

	var intervalaCount int
	if err := database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.intervala").Scan(&intervalaCount); err != nil {
		t.Fatalf("failed to query intervala count: %v", err)
	}
	if intervalaCount != 25 {
		t.Errorf("expected 25 intervala rows, got %d", intervalaCount)
	}

	// 2. Second run on non-empty DB should return error
	if err := SetDB(ctx, testDBName); err == nil {
		t.Fatalf("expected error when running SetDB on non-empty database, got nil")
	}
}
