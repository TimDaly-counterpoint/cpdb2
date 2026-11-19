package cmatrix

import (
	"testing"

	"cpdb2/pkg/db"
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
