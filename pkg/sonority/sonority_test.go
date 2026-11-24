package sonority

import (
	"context"
	"fmt"
	"testing"

	"cpdb2/pkg/db"
	"cpdb2/pkg/setdb"
)

func TestRecreateSonorityTables_Integration(t *testing.T) {
	testDBName := "test_sonority_temp"
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

	// 1. Initialize test database schema and voces
	if err := setdb.SetDB(ctx, testDBName); err != nil {
		t.Fatalf("SetDB failed: %v", err)
	}

	database, err := db.NewDB(ctx, testDBName)
	if err != nil {
		t.Fatalf("Connect to test db failed: %v", err)
	}
	defer database.Close()

	// 2. Insert test pieces, staves, and notes
	insertPiecesSQL := `
		INSERT INTO public.pieces (title, composer, voices) VALUES
		('Test Piece 1', 'Composer 1', 2),
		('Test Piece 2', 'Composer 2', 2);
	`
	if _, err := database.Pool.Exec(ctx, insertPiecesSQL); err != nil {
		t.Fatalf("Insert pieces failed: %v", err)
	}

	insertStavesSQL := `
		INSERT INTO public.staves (num, denom, modus, tempus, prolatio, dimin, propd, propn, modusct, key, clef) VALUES
		(4, 4, 2, 2, 2, false, 1, 1, 1, 0, 'C1');
	`
	if _, err := database.Pool.Exec(ctx, insertStavesSQL); err != nil {
		t.Fatalf("Insert staves failed: %v", err)
	}

	// Piece 1:
	// Voice 1: C4 (midi 60, toffset 0, dur 48), D4 (midi 62, toffset 48, dur 48), B3 (midi 58, toffset 552, dur 48)
	// Voice 2: E4 (midi 64, toffset 0, dur 48), F4 (midi 65, toffset 48, dur 48), E3 (midi 52, toffset 552, dur 48)
	// Piece 2:
	// Voice 1: G4 (midi 67, toffset 0, dur 48)
	// Voice 2: C5 (midi 72, toffset 0, dur 48)
	insertNotesSQL := `
		INSERT INTO public.notes (piece, mvt, staffchange, voice, index, pitch, duration, midi, inflec, ficta, colour, bar, islast, toffset, dimlevel) VALUES
		(1, 1, 1, 1, 1, 'C4', 48, 60, 0, 0, false, 1, 0, 0, 0),
		(1, 1, 1, 1, 2, 'D4', 48, 62, 0, 0, false, 2, 0, 48, 0),
		(1, 1, 1, 1, 3, 'B3', 48, 58, 0, 0, false, 3, 1, 552, 0),
		(1, 1, 1, 2, 1, 'E4', 48, 64, 0, 0, false, 1, 0, 0, 0),
		(1, 1, 1, 2, 2, 'F4', 48, 65, 0, 0, false, 2, 0, 48, 0),
		(1, 1, 1, 2, 3, 'E3', 48, 52, 0, 0, false, 3, 1, 552, 0),
		(2, 1, 1, 1, 1, 'G4', 48, 67, 0, 0, false, 1, 1, 0, 0),
		(2, 1, 1, 2, 1, 'C5', 48, 72, 0, 0, false, 1, 1, 0, 0);
	`
	if _, err := database.Pool.Exec(ctx, insertNotesSQL); err != nil {
		t.Fatalf("Insert notes failed: %v", err)
	}

	// 3. Drop views sonority_view and sonority_dissonance_view to guarantee process operates without views
	dropViewsSQL := `
		DROP VIEW IF EXISTS public.sonority_dissonance_view CASCADE;
		DROP VIEW IF EXISTS public.sonority_view CASCADE;
	`
	if _, err := database.Pool.Exec(ctx, dropViewsSQL); err != nil {
		t.Fatalf("Drop views failed: %v", err)
	}

	// 4. First run of RecreateSonorityTables
	summary, err := RecreateSonorityTables(ctx, database)
	if err != nil {
		t.Fatalf("RecreateSonorityTables failed: %v", err)
	}

	if summary.SonorityCount != 4 {
		t.Errorf("expected 4 sonority rows, got %d", summary.SonorityCount)
	}
	if summary.DissonanceCount != 4 {
		t.Errorf("expected 4 dissonance rows, got %d", summary.DissonanceCount)
	}
	if summary.ConsonantCount == 0 {
		t.Errorf("expected > 0 consonant rows, got %d", summary.ConsonantCount)
	}

	// Verify row counts in the database
	var sonCount, dissCount, consCount int
	if err := database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.sonority").Scan(&sonCount); err != nil {
		t.Fatalf("query sonority count failed: %v", err)
	}
	if err := database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.dissonance").Scan(&dissCount); err != nil {
		t.Fatalf("query dissonance count failed: %v", err)
	}
	if err := database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.consonant").Scan(&consCount); err != nil {
		t.Fatalf("query consonant count failed: %v", err)
	}

	if sonCount != 4 || dissCount != 4 || consCount != int(summary.ConsonantCount) {
		t.Errorf("database counts mismatch: sonority=%d, dissonance=%d, consonant=%d", sonCount, dissCount, consCount)
	}

	// Verify specific MIDI dissonance behavior for B3 and E3 (midi diff 6, voces diff 4) at offset 552
	var pitchesDiss, midiDiss bool
	err = database.Pool.QueryRow(ctx, `
		SELECT pitches_dissonant, midi_dissonant
		FROM public.dissonance
		WHERE piece = 1 AND "offset" = 552
	`).Scan(&pitchesDiss, &midiDiss)
	if err != nil {
		t.Fatalf("query dissonance for piece 1 offset 552 failed: %v", err)
	}
	if pitchesDiss != false {
		t.Errorf("expected pitches_dissonant=false for B3 and E3, got %v", pitchesDiss)
	}
	if midiDiss != true {
		t.Errorf("expected midi_dissonant=true for MIDI diff 6 (58 - 52), got %v", midiDiss)
	}

	// 5. Second run to verify tables are cleared and repopulated without duplicate accumulation
	summary2, err := RecreateSonorityTables(ctx, database)
	if err != nil {
		t.Fatalf("Second RecreateSonorityTables failed: %v", err)
	}

	if summary2.SonorityCount != 4 || summary2.DissonanceCount != 4 || summary2.ConsonantCount != summary.ConsonantCount {
		t.Errorf("unexpected counts on second run: %+v", summary2)
	}

	// Verify database row counts after second run
	if err := database.Pool.QueryRow(ctx, "SELECT count(*) FROM public.sonority").Scan(&sonCount); err != nil {
		t.Fatalf("query sonority count failed: %v", err)
	}
	if sonCount != 4 {
		t.Errorf("expected 4 sonority rows after repopulation, got %d", sonCount)
	}
}
