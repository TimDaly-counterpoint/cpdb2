package score

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"cpdb2/pkg/db"
	"cpdb2/pkg/setdb"
)

func TestLoadMEIFiles(t *testing.T) {
	files, err := filepath.Glob("../../Mente/Mei/*.mei")
	if err != nil {
		t.Fatalf("glob MEI files: %v", err)
	}
	if len(files) == 0 {
		files, _ = filepath.Glob("../../../Mente/Mei/*.mei")
	}
	if len(files) == 0 {
		t.Skip("no MEI files found")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			mei, err := LoadMEI(file)
			if err != nil {
				t.Fatalf("LoadMEI: %v", err)
			}

			score := mei.ToScore()
			if score.Partwise == nil {
				t.Fatal("expected partwise score")
			}
			if score.Partwise.Work == nil || score.Partwise.Work.WorkTitle == "" {
				t.Fatal("expected title metadata")
			}
			if len(score.Partwise.Parts) == 0 {
				t.Fatal("expected at least one part")
			}
			if len(score.Partwise.Parts[0].Measures) == 0 {
				t.Fatal("expected at least one measure")
			}
		})
	}
}

func TestMEIKeySigToFifths(t *testing.T) {
	tests := map[string]int{
		"":   0,
		"0":  0,
		"1f": -1,
		"2f": -2,
		"1s": 1,
		"3s": 3,
	}

	for keySig, want := range tests {
		if got := meiKeySigToFifths(keySig); got != want {
			t.Fatalf("meiKeySigToFifths(%q) = %d, want %d", keySig, got, want)
		}
	}
}

func TestMEIAttributesUseStaffDefKeySig(t *testing.T) {
	path := "../../Mente/Mei/CRIM_Mass_0016_1.mei"
	mei, err := LoadMEI(path)
	if err != nil {
		t.Skipf("LoadMEI %s: %v", path, err)
	}

	attrs := mei.AttributesForStaff("1")
	if attrs == nil || len(attrs.Key) == 0 {
		t.Fatal("expected MEI staff attributes to include key signature")
	}
	if attrs.Key[0].Fifths != -1 {
		t.Fatalf("MEI staff key = %d, want -1 for key.sig=\"1f\"", attrs.Key[0].Fifths)
	}
}

func TestMEIIgnoresSuppliedAccidentals(t *testing.T) {
	note := meiNoteToMusicXML(MEINote{
		PName:  "f",
		Octave: 4,
		Supplied: []MEISupplied{{Accids: []MEIAccid{{
			Accid: "s",
		}}}},
	}, 256)

	if !note.AccidentalEditorial {
		t.Fatal("expected accidental in <supplied> to be marked as editorial")
	}
	if note.Pitch.Alter != 1 {
		t.Fatalf("supplied accidental alter = %d, want 1", note.Pitch.Alter)
	}
}

func TestNoteDurationFromType(t *testing.T) {
	if got := NoteDurationFromType("whole", 0); got != 96.0 {
		t.Errorf("whole note duration = %v, want 96.0", got)
	}
	if got := NoteDurationFromType("half", 1); got != 72.0 {
		t.Errorf("dotted half note duration = %v, want 72.0", got)
	}
	if got := NoteDurationFromType("quarter", 0); got != 24.0 {
		t.Errorf("quarter note duration = %v, want 24.0", got)
	}
}

func TestFindFilesByPatterns(t *testing.T) {
	files, err := FindFilesByPatterns("../../test.mei", "../../Rtest.mei")
	if err != nil {
		t.Fatalf("find files: %v", err)
	}
	if len(files) == 0 {
		t.Log("No test files found at relative path")
	}
}

func TestImportScores_Integration(t *testing.T) {
	files, err := filepath.Glob("../../input/*.musicxml")
	if err != nil || len(files) == 0 {
		files, _ = filepath.Glob("input/*.musicxml")
	}
	if len(files) == 0 {
		t.Skip("no input musicxml files found")
	}

	testDBName := "test_import_scores_temp"
	ctx := context.Background()

	postgresConn := db.ResolveConnString("postgres")
	adminDB, err := db.NewDB(ctx, "postgres")
	if err != nil {
		t.Skipf("PostgreSQL server not available: %v", err)
	}
	defer adminDB.Close()

	_, _ = adminDB.Pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", testDBName))
	if _, err := adminDB.Pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", testDBName)); err != nil {
		t.Skipf("cannot create test database: %v", err)
	}
	defer func() {
		cleanupDB, err := db.NewDB(ctx, "postgres")
		if err == nil {
			_, _ = cleanupDB.Pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", testDBName))
			cleanupDB.Close()
		}
	}()

	_ = postgresConn
	if err := setdb.SetDB(ctx, testDBName); err != nil {
		t.Fatalf("SetDB failed: %v", err)
	}

	targetDB, err := db.NewDB(ctx, testDBName)
	if err != nil {
		t.Fatalf("Connect to %s: %v", testDBName, err)
	}
	defer targetDB.Close()

	// Import first 2 score files
	testPatterns := files[:min(2, len(files))]
	count, err := ImportScores(ctx, targetDB, testPatterns...)
	if err != nil {
		t.Fatalf("ImportScores failed: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected at least 1 imported score, got 0")
	}

	var noteCount int
	err = targetDB.Pool.QueryRow(ctx, "SELECT count(*) FROM notes").Scan(&noteCount)
	if err != nil {
		t.Fatalf("query notes count: %v", err)
	}
	if noteCount == 0 {
		t.Fatalf("expected notes in database, got 0")
	}

	var pieceCount int
	err = targetDB.Pool.QueryRow(ctx, "SELECT count(*) FROM pieces").Scan(&pieceCount)
	if err != nil {
		t.Fatalf("query pieces count: %v", err)
	}
	if pieceCount != count {
		t.Fatalf("expected %d pieces, got %d", count, pieceCount)
	}
}
