package tritones

import (
	"context"
	"fmt"
	"testing"

	"cpdb2/pkg/db"
	"cpdb2/pkg/setdb"
)

func TestContainsTritone(t *testing.T) {
	// 60 (C) and 66 (F#) is a tritone (difference 6)
	if !ContainsTritone([]int{60, 66}) {
		t.Error("expected true for [60, 66]")
	}

	// 60 (C) and 67 (G) is a fifth (difference 7)
	if ContainsTritone([]int{60, 67}) {
		t.Error("expected false for [60, 67]")
	}

	// with zero (rest)
	if ContainsTritone([]int{0, 60}) {
		t.Error("expected false for [0, 60]")
	}

	// Multiple voices containing tritone across octaves (e.g. 58 Bb and 52 E = diff 6, or 48 C and 66 F# = diff 18 % 12 == 6)
	if !ContainsTritone([]int{48, 66}) {
		t.Error("expected true for [48, 66]")
	}

	// Single note or empty
	if ContainsTritone([]int{60}) {
		t.Error("expected false for single note [60]")
	}
	if ContainsTritone([]int{}) {
		t.Error("expected false for empty slice")
	}
}

func TestFindTritones_Integration(t *testing.T) {
	testDBName := "test_tritones_temp"
	ctx := context.Background()

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

	if err := setdb.SetDB(ctx, testDBName); err != nil {
		t.Fatalf("SetDB failed: %v", err)
	}

	database, err := db.NewDB(ctx, testDBName)
	if err != nil {
		t.Fatalf("Connect to test db failed: %v", err)
	}
	defer database.Close()

	// 1. Insert test data into dissonance table
	_, err = database.Pool.Exec(ctx, `
		INSERT INTO public.dissonance (piece, "offset", duration, pitches, midi_values, note_ids, voices, pitches_dissonant, midi_dissonant)
		VALUES
			(1, 100, 24, ARRAY['C4','F#4'], ARRAY[60, 66], ARRAY[1, 2], ARRAY[1, 2], true, true),
			(1, 200, 24, ARRAY['C4','G4'], ARRAY[60, 67], ARRAY[3, 4], ARRAY[1, 2], false, false),
			(1, 300, 24, ARRAY['Bb3','E3'], ARRAY[58, 52], ARRAY[5, 6], ARRAY[1, 2], true, true),
			(2, 400, 24, ARRAY['C4','F#4'], ARRAY[60, 66], ARRAY[7, 8], ARRAY[1, 2], true, true);
	`)
	if err != nil {
		t.Fatalf("Insert into dissonance table failed: %v", err)
	}

	// 2. Test FindTritones on piece 1
	offsets, err := FindTritones(ctx, database, 1)
	if err != nil {
		t.Fatalf("FindTritones failed: %v", err)
	}

	expectedOffsets := []float64{100, 300}
	if len(offsets) != len(expectedOffsets) {
		t.Fatalf("expected %d offsets, got %d (%v)", len(expectedOffsets), len(offsets), offsets)
	}
	for i, o := range offsets {
		if o != expectedOffsets[i] {
			t.Errorf("offset %d: got %f, want %f", i, o, expectedOffsets[i])
		}
	}

	// 3. Test when table is named 'dissonant'
	_, err = database.Pool.Exec(ctx, `
		CREATE TABLE public.dissonant (LIKE public.dissonance INCLUDING ALL);
		INSERT INTO public.dissonant SELECT * FROM public.dissonance;
		DROP TABLE public.dissonance;
	`)
	if err != nil {
		t.Fatalf("Create and populate 'dissonant' table failed: %v", err)
	}

	tableResolved := ResolveDissonanceTable(ctx, database)
	if tableResolved != "dissonant" {
		t.Errorf("expected resolved table 'dissonant', got %q", tableResolved)
	}

	offsetsDiss, err := FindTritones(ctx, database, 1)
	if err != nil {
		t.Fatalf("FindTritones with 'dissonant' table failed: %v", err)
	}
	if len(offsetsDiss) != len(expectedOffsets) {
		t.Fatalf("expected %d offsets from 'dissonant' table, got %d (%v)", len(expectedOffsets), len(offsetsDiss), offsetsDiss)
	}
}
