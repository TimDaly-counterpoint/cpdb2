package exportxml

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cpdb2/pkg/db"
	"cpdb2/pkg/setdb"
	"cpdb2/pkg/sonority"
)

func TestParseSequencesString(t *testing.T) {
	input := `Sequence: [2, 3, 4]
Length: 3
Count: 2
  Piece: 1, Voice: 1, List: intervList, Bar: 1, Index: 1, Pitch: C4, Offset: 0.00
  Piece: 2, Voice: 2, List: invertList, Bar: 3, Index: 10, Pitch: D4, Offset: 96.00
--------------------`

	seqs, err := ParseSequencesString(input)
	if err != nil {
		t.Fatalf("ParseSequencesString: %v", err)
	}
	if len(seqs) != 1 {
		t.Fatalf("expected 1 sequence, got %d", len(seqs))
	}
	if len(seqs[0].Sequence) != 3 {
		t.Errorf("expected sequence length 3, got %d", len(seqs[0].Sequence))
	}
	if len(seqs[0].Occurrences) != 2 {
		t.Errorf("expected 2 occurrences, got %d", len(seqs[0].Occurrences))
	}
}

func TestGetNoteTypeAndDot(t *testing.T) {
	name, dotted := getNoteTypeAndDot(24)
	if name != "quarter" || dotted {
		t.Errorf("duration 24: got %s, dotted %v, want quarter, false", name, dotted)
	}

	name, dotted = getNoteTypeAndDot(36)
	if name != "quarter" || !dotted {
		t.Errorf("duration 36: got %s, dotted %v, want quarter, true", name, dotted)
	}
}

func TestParsePitchComponents(t *testing.T) {
	step, oct, alt := parsePitchComponents("F#4", 66)
	if step != "F" || oct != 4 || alt != 1 {
		t.Errorf("parse F#4: got %s %d %d, want F 4 1", step, oct, alt)
	}

	step, oct, alt = parsePitchComponents("Bb3", 58)
	if step != "B" || oct != 3 || alt != -1 {
		t.Errorf("parse Bb3: got %s %d %d, want B 3 -1", step, oct, alt)
	}
}

func TestSanitizePieceTitle(t *testing.T) {
	if got := sanitizePieceTitle("Missa L'homme arme!"); got != "MissaLhomm" {
		t.Errorf("sanitizePieceTitle got %q, want 'MissaLhomm'", got)
	}
}

func TestGenerateConsonantMusicXML(t *testing.T) {
	sonorities := []ConsonantSonority{
		{
			Offset:     0,
			Duration:   48,
			Pitches:    []string{"C4", "E4"},
			MidiValues: []int{60, 64},
			Voices:     []int{1, 2},
		},
		{
			Offset:     48,
			Duration:   48,
			Pitches:    []string{"D4", "F4"},
			MidiValues: []int{62, 65},
			Voices:     []int{1, 2},
		},
		{
			Offset:     120,
			Duration:   24,
			Pitches:    []string{"G4"},
			MidiValues: []int{67},
			Voices:     []int{1},
		},
	}

	xmlStr, err := GenerateConsonantMusicXML("Test Work", 1, sonorities)
	if err != nil {
		t.Fatalf("GenerateConsonantMusicXML failed: %v", err)
	}

	if !strings.Contains(xmlStr, "<work-title>Test Work</work-title>") {
		t.Errorf("expected work title in XML")
	}
	if !strings.Contains(xmlStr, `<score-part id="P1">`) || !strings.Contains(xmlStr, `<part-name>Voice 1</part-name>`) {
		t.Errorf("expected Voice 1 part definition in XML")
	}
	if !strings.Contains(xmlStr, `<score-part id="P2">`) || !strings.Contains(xmlStr, `<part-name>Voice 2</part-name>`) {
		t.Errorf("expected Voice 2 part definition in XML")
	}
	if !strings.Contains(xmlStr, `<part id="P1">`) || !strings.Contains(xmlStr, `<part id="P2">`) {
		t.Errorf("expected part tags for P1 and P2 in XML")
	}
	if !strings.Contains(xmlStr, "<step>C</step>") || !strings.Contains(xmlStr, "<step>E</step>") {
		t.Errorf("expected C and E pitch steps in XML")
	}
	// Voice 2 should have rest for offset 96-144 (or offset 120-144)
	if !strings.Contains(xmlStr, "<rest/>") {
		t.Errorf("expected rest in XML for missing voice / gaps")
	}
}

func TestExportConsonantToMusicXML_Integration(t *testing.T) {
	testDBName := "test_export_smooth_temp"
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

	targetDB, err := db.NewDB(ctx, testDBName)
	if err != nil {
		t.Fatalf("Connect to %s: %v", testDBName, err)
	}
	defer targetDB.Close()

	// Insert a test piece and notes
	var pieceSeq int
	err = targetDB.Pool.QueryRow(ctx, "INSERT INTO pieces (title, composer) VALUES ('Test Piece 1', 'Test Composer') RETURNING seq").Scan(&pieceSeq)
	if err != nil {
		t.Fatalf("insert piece failed: %v", err)
	}

	// Insert a test staff
	var staffSeq int
	err = targetDB.Pool.QueryRow(ctx, "INSERT INTO staves (key, num, denom, clef) VALUES (0, 4, 4, 'G2') RETURNING seq").Scan(&staffSeq)
	if err != nil {
		t.Fatalf("insert staff failed: %v", err)
	}

	insertNotesSQL := fmt.Sprintf(`
		INSERT INTO public.notes (piece, mvt, staffchange, voice, index, pitch, duration, midi, inflec, ficta, colour, bar, islast, toffset, dimlevel) VALUES
		(%%d, 1, %d, 1, 1, 'C4', 48, 60, 0, 0, false, 1, 0, 0, 0),
		(%%d, 1, %d, 1, 2, 'D4', 48, 62, 0, 0, false, 2, 0, 48, 0),
		(%%d, 1, %d, 1, 3, 'B3', 48, 58, 0, 0, false, 3, 1, 96, 0),
		(%%d, 1, %d, 2, 1, 'E4', 48, 64, 0, 0, false, 1, 0, 0, 0),
		(%%d, 1, %d, 2, 2, 'F4', 48, 65, 0, 0, false, 2, 0, 48, 0),
		(%%d, 1, %d, 2, 3, 'G4', 48, 67, 0, 0, false, 3, 1, 96, 0);
	`, staffSeq, staffSeq, staffSeq, staffSeq, staffSeq, staffSeq)
	insertNotesSQL = fmt.Sprintf(insertNotesSQL, pieceSeq, pieceSeq, pieceSeq, pieceSeq, pieceSeq, pieceSeq)
	if _, err := targetDB.Pool.Exec(ctx, insertNotesSQL); err != nil {
		t.Fatalf("insert test notes failed: %v", err)
	}

	// Populate sonority, dissonance, and consonant tables
	if _, err := sonority.RecreateSonorityTables(ctx, targetDB); err != nil {
		t.Fatalf("RecreateSonorityTables failed: %v", err)
	}

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "smooth_out.musicxml")

	if err := ExportConsonantToMusicXML(ctx, targetDB, pieceSeq, outFile); err != nil {
		t.Fatalf("ExportConsonantToMusicXML failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read exported musicxml failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<score-partwise") {
		t.Errorf("expected score-partwise root in exported musicxml")
	}
	if !strings.Contains(content, "<work-title>Test Piece 1</work-title>") {
		t.Errorf("expected piece title 'Test Piece 1' in exported musicxml")
	}
	if !strings.Contains(content, `<score-part id="P1">`) || !strings.Contains(content, `<score-part id="P2">`) {
		t.Errorf("expected P1 and P2 parts in exported musicxml")
	}
	if !strings.Contains(content, "<step>C</step>") || !strings.Contains(content, "<step>E</step>") {
		t.Errorf("expected pitch steps in exported musicxml")
	}
}
