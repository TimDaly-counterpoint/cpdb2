package exportxml

import (
	"testing"
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
