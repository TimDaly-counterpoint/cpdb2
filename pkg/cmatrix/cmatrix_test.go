package cmatrix

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cpdb2/pkg/db"
	"cpdb2/pkg/setdb"
	"cpdb2/pkg/sonority"
)

func TestCalculateVoiceIntervals(t *testing.T) {
	notes := []db.SequenceNote{
		{Voice: 1, Bar: 1, Index: 1, Pitch: "0", Duration: 24, Toffset: 0},
		{Voice: 1, Bar: 1, Index: 2, Pitch: "C4", Duration: 24, Toffset: 24},
		{Voice: 1, Bar: 1, Index: 3, Pitch: "E4", Duration: 48, Toffset: 48},
		{Voice: 1, Bar: 2, Index: 4, Pitch: "G4", Duration: 96, Toffset: 96},
	}

	melodic := CalculateVoiceIntervals(notes)
	if len(melodic) != 3 {
		t.Fatalf("expected 3 melodic notes, got %d", len(melodic))
	}

	if melodic[0].Interval != 0 {
		t.Errorf("note 0 interval = %d, want 0", melodic[0].Interval)
	}
	if melodic[1].Interval != 2 { // C4 (14) to E4 (16) -> 16 - 14 = 2
		t.Errorf("note 1 interval = %d, want 2", melodic[1].Interval)
	}
	if melodic[2].Interval != 4 { // C4 (14) to G4 (18) -> 18 - 14 = 4
		t.Errorf("note 2 interval = %d, want 4", melodic[2].Interval)
	}
}

func TestFindVoiceDifferences(t *testing.T) {
	v1 := []MelodicNote{
		{Voice: 1, Pitch: "C4", Interval: 0},
		{Voice: 1, Pitch: "D4", Interval: 1},
	}
	v2 := []MelodicNote{
		{Voice: 2, Pitch: "C4", Interval: 0},
		{Voice: 2, Pitch: "E4", Interval: 2},
	}

	voiceNotes := map[int][]MelodicNote{
		1: v1,
		2: v2,
	}

	diffs := FindVoiceDifferences(voiceNotes, []int{1, 2})
	if len(diffs) != 1 {
		t.Fatalf("expected 1 difference, got %d", len(diffs))
	}
	if diffs[0].Position != 2 {
		t.Errorf("difference position = %d, want 2", diffs[0].Position)
	}
}

func TestGenerateIntervalMatrix(t *testing.T) {
	v1 := []MelodicNote{
		{Voice: 1, Bar: 1, Toffset: 0, Duration: 48, Pitch: "C4", Interval: 0},
	}
	v2 := []MelodicNote{
		{Voice: 2, Bar: 1, Toffset: 0, Duration: 48, Pitch: "E4", Interval: 0},
	}

	voiceNotes := map[int][]MelodicNote{
		1: v1,
		2: v2,
	}

	matrix, toffsets := GenerateIntervalMatrix(voiceNotes, []int{1, 2})
	if len(toffsets) != 1 {
		t.Fatalf("expected 1 toffset, got %d", len(toffsets))
	}
	if len(matrix) == 0 {
		t.Fatal("expected non-empty matrix")
	}
	if matrix[0][0] != "Offset" {
		t.Errorf("matrix header[0] = %q, want 'Offset'", matrix[0][0])
	}
}

func TestFormatVoiceDifferences(t *testing.T) {
	s := FormatVoiceDifferences(nil)
	if s == "" {
		t.Error("expected non-empty message for nil differences")
	}
}

func TestBuildVoiceNotesFromSonorities(t *testing.T) {
	sonRows := []SonorityRow{
		{
			Offset:     0,
			Duration:   120,
			Pitches:    []string{"C4", "C3"},
			MidiValues: []int{60, 48},
			Voices:     []int{1, 2},
		},
		{
			Offset:     120,
			Duration:   24,
			Pitches:    []string{"A3", "D3"},
			MidiValues: []int{57, 50},
			Voices:     []int{1, 2},
		},
		{
			Offset:     144,
			Duration:   24,
			Pitches:    []string{"0", "F3"},
			MidiValues: []int{0, 53},
			Voices:     []int{1, 2},
		},
		{
			Offset:     168,
			Duration:   12,
			Pitches:    []string{"D4", "0"},
			MidiValues: []int{62, 0},
			Voices:     []int{1, 2},
		},
		{
			Offset:     180,
			Duration:   12,
			Pitches:    []string{"C4", "0"},
			MidiValues: []int{60, 0},
			Voices:     []int{1, 2},
		},
		{
			Offset:     192,
			Duration:   12,
			Pitches:    []string{"C4", "F3"},
			MidiValues: []int{60, 53},
			Voices:     []int{1, 2},
		},
	}

	barStarts := map[int]float64{
		1: 0,
		3: 96,
		4: 144,
		5: 192,
	}
	barOrder := []int{1, 3, 4, 5}

	v1Notes := BuildVoiceNotesFromSonorities(sonRows, 1, barStarts, barOrder)
	// V1 expected notes:
	// 1: C4 (0-120, dur 120, bar 1)
	// 2: A3 (120-144, dur 24, bar 3)
	// 3: 0 (144-168, dur 24, bar 4)
	// 4: D4 (168-180, dur 12, bar 4)
	// 5: C4 (180-204, dur 24, bar 4, merged from 180 and 192)
	if len(v1Notes) != 5 {
		t.Fatalf("expected 5 merged notes for voice 1, got %d", len(v1Notes))
	}
	if v1Notes[0].Pitch != "C4" || v1Notes[0].Duration != 120 || v1Notes[0].Bar != 1 {
		t.Errorf("v1 note 0 mismatch: %+v", v1Notes[0])
	}
	if v1Notes[1].Pitch != "A3" || v1Notes[1].Duration != 24 || v1Notes[1].Bar != 3 {
		t.Errorf("v1 note 1 mismatch: %+v", v1Notes[1])
	}
	if v1Notes[2].Pitch != "0" || v1Notes[2].Duration != 24 || v1Notes[2].Bar != 4 {
		t.Errorf("v1 note 2 mismatch: %+v", v1Notes[2])
	}
	if v1Notes[3].Pitch != "D4" || v1Notes[3].Duration != 12 || v1Notes[3].Bar != 4 {
		t.Errorf("v1 note 3 mismatch: %+v", v1Notes[3])
	}
	if v1Notes[4].Pitch != "C4" || v1Notes[4].Duration != 24 || v1Notes[4].Bar != 4 {
		t.Errorf("v1 note 4 mismatch: %+v", v1Notes[4])
	}
}

func TestGetBarForOffset(t *testing.T) {
	barStarts := map[int]float64{
		1: 0,
		3: 96,
		4: 144,
		5: 192,
	}
	barOrder := []int{1, 3, 4, 5}

	bar, barOffset := getBarForOffset(0, barStarts, barOrder)
	if bar != 1 || barOffset != 0 {
		t.Errorf("offset 0: got bar %d, barOffset %f; want 1, 0", bar, barOffset)
	}

	bar, barOffset = getBarForOffset(120, barStarts, barOrder)
	if bar != 3 || barOffset != 24 {
		t.Errorf("offset 120: got bar %d, barOffset %f; want 3, 24", bar, barOffset)
	}

	bar, barOffset = getBarForOffset(180, barStarts, barOrder)
	if bar != 4 || barOffset != 36 {
		t.Errorf("offset 180: got bar %d, barOffset %f; want 4, 36", bar, barOffset)
	}

	bar, barOffset = getBarForOffset(192, barStarts, barOrder)
	if bar != 5 || barOffset != 0 {
		t.Errorf("offset 192: got bar %d, barOffset %f; want 5, 0", bar, barOffset)
	}
}

func TestProcessCMatrix_FloridAndConsonant_Integration(t *testing.T) {
	testDBName := "test_cmatrix_temp"
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

	// 2. Insert piece, staves, and notes
	if _, err := database.Pool.Exec(ctx, "INSERT INTO public.pieces (title, composer, voices) VALUES ('Test Piece', 'Composer', 2)"); err != nil {
		t.Fatalf("Insert pieces failed: %v", err)
	}
	if _, err := database.Pool.Exec(ctx, "INSERT INTO public.staves (num, denom, modus, tempus, prolatio, dimin, propd, propn, modusct, key, clef) VALUES (4, 4, 2, 2, 2, false, 1, 1, 1, 0, 'C1')"); err != nil {
		t.Fatalf("Insert staves failed: %v", err)
	}

	insertNotesSQL := `
		INSERT INTO public.notes (piece, mvt, staffchange, voice, index, pitch, duration, midi, inflec, ficta, colour, bar, islast, toffset, dimlevel) VALUES
		(1, 1, 1, 1, 1, 'C4', 48, 60, 0, 0, false, 1, 0, 0, 0),
		(1, 1, 1, 1, 2, 'D4', 48, 62, 0, 0, false, 2, 0, 48, 0),
		(1, 1, 1, 1, 3, 'E4', 48, 64, 0, 0, false, 3, 1, 96, 0),
		(1, 1, 1, 2, 1, 'C3', 48, 48, 0, 0, false, 1, 0, 0, 0),
		(1, 1, 1, 2, 2, 'D3', 48, 50, 0, 0, false, 2, 0, 48, 0),
		(1, 1, 1, 2, 3, 'G3', 48, 55, 0, 0, false, 3, 1, 96, 0);
	`
	if _, err := database.Pool.Exec(ctx, insertNotesSQL); err != nil {
		t.Fatalf("Insert notes failed: %v", err)
	}

	// 3. Populate sonority and consonant tables
	if _, err := sonority.RecreateSonorityTables(ctx, database); err != nil {
		t.Fatalf("RecreateSonorityTables failed: %v", err)
	}

	// 4. Test Florid CMatrix process
	tmpFloridCSV := "test_cmatrix_florid_out.csv"
	defer os.Remove(tmpFloridCSV)

	resFlorid, err := ProcessCMatrix(ctx, database, 1, []int{1, 2}, VersionFlorid, tmpFloridCSV)
	if err != nil {
		t.Fatalf("ProcessCMatrix (florid) failed: %v", err)
	}
	if resFlorid.Version != VersionFlorid {
		t.Errorf("expected version %s, got %s", VersionFlorid, resFlorid.Version)
	}
	if len(resFlorid.Matrix) == 0 {
		t.Fatal("expected non-empty florid matrix")
	}
	if resFlorid.Matrix[0][0] != "Offset" {
		t.Errorf("header[0] = %q, want 'Offset'", resFlorid.Matrix[0][0])
	}

	// 5. Test Consonant CMatrix process
	tmpConsonantCSV := "test_cmatrix_consonant_out.csv"
	defer os.Remove(tmpConsonantCSV)

	resConsonant, err := ProcessCMatrix(ctx, database, 1, []int{1, 2}, VersionConsonant, tmpConsonantCSV)
	if err != nil {
		t.Fatalf("ProcessCMatrix (consonant) failed: %v", err)
	}
	if resConsonant.Version != VersionConsonant {
		t.Errorf("expected version %s, got %s", VersionConsonant, resConsonant.Version)
	}
	if len(resConsonant.Matrix) == 0 {
		t.Fatal("expected non-empty consonant matrix")
	}

	// 6. Test Interactive Prompt for both versions
	inputFlorid := "1\nf\n1,2\n\n"
	readerFlorid := bufio.NewReader(strings.NewReader(inputFlorid))
	if err := RunInteractive(ctx, database, readerFlorid); err != nil {
		t.Fatalf("RunInteractive with 'f' failed: %v", err)
	}

	inputConsonant := "1\nc\n1,2\n\n"
	readerConsonant := bufio.NewReader(strings.NewReader(inputConsonant))
	if err := RunInteractive(ctx, database, readerConsonant); err != nil {
		t.Fatalf("RunInteractive with 'c' failed: %v", err)
	}

	// Clean up any default csv files generated
	defer func() {
		files, _ := os.ReadDir(".")
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "cmatrix_1_") && strings.HasSuffix(f.Name(), ".csv") {
				_ = os.Remove(f.Name())
			}
		}
	}()

	// 7. Test invalid version returns error
	inputInvalid := "1\ninvalid_ver\n1,2\n\n"
	readerInvalid := bufio.NewReader(strings.NewReader(inputInvalid))
	if err := RunInteractive(ctx, database, readerInvalid); err == nil {
		t.Fatalf("expected error for invalid version, got nil")
	}
}
