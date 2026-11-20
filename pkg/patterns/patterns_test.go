package patterns

import (
	"reflect"
	"testing"
)

func TestBuildIntervList(t *testing.T) {
	tests := []struct {
		name        string
		values      []int
		wantResult  []int
		wantMapping []int
	}{
		{
			name:        "simple ascending",
			values:      []int{1, 3, 5},
			wantResult:  []int{3, 3},
			wantMapping: []int{0, 1},
		},
		{
			name:        "simple descending",
			values:      []int{5, 3, 1},
			wantResult:  []int{-3, -3},
			wantMapping: []int{0, 1},
		},
		{
			name:        "unison step",
			values:      []int{4, 4},
			wantResult:  []int{1},
			wantMapping: []int{0},
		},
		{
			name:        "with rest (0)",
			values:      []int{1, 0, 0, 3},
			wantResult:  []int{0},
			wantMapping: []int{1},
		},
		{
			name:        "leading rest",
			values:      []int{0, 0, 2, 4},
			wantResult:  []int{0, 3},
			wantMapping: []int{0, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotMapping := BuildIntervList(tt.values)
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("BuildIntervList() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if !reflect.DeepEqual(gotMapping, tt.wantMapping) {
				t.Errorf("BuildIntervList() gotMapping = %v, want %v", gotMapping, tt.wantMapping)
			}
		})
	}
}

func TestNegate(t *testing.T) {
	in := []int{3, 1, -2, 4}
	want := []int{-3, 1, 2, -4}
	got := Negate(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Negate(%v) = %v, want %v", in, got, want)
	}
}

func TestContainsSubseq(t *testing.T) {
	haystack := []int{1, 2, 3, 4, 5, 6}
	if !containsSubseq(haystack, []int{3, 4, 5}) {
		t.Error("expected true for existing subseq")
	}
	if containsSubseq(haystack, []int{3, 5}) {
		t.Error("expected false for non-contiguous subseq")
	}
	if !containsSubseq(haystack, []int{}) {
		t.Error("expected true for empty needle")
	}
}

func TestFilterRepeatedResults(t *testing.T) {
	res := []SequenceResult{
		{
			Sequence: []int{2, 3, 4},
			Count:    2,
			Occurrences: []SequenceOccurrence{
				{Piece: 1, Voice: 1, Index: 1, EndIndex: 4, Toffset: 0, EndToffset: 96, ListType: "intervList"},
				{Piece: 2, Voice: 1, Index: 5, EndIndex: 8, Toffset: 96, EndToffset: 192, ListType: "intervList"},
			},
		},
	}
	filtered := FilterRepeatedResults(res, 0)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
}

func TestFilterRepeatedResultsPriority(t *testing.T) {
	// 4 results referring to the exact same range of notes, but from different list types
	res := []SequenceResult{
		{
			Sequence: []int{10, 20},
			Count:    2,
			Occurrences: []SequenceOccurrence{
				{Piece: 50, Voice: 4, Index: 154, EndIndex: 1, Toffset: 19296, EndToffset: 0, ListType: "retroList"},
				{Piece: 51, Voice: 3, Index: 285, EndIndex: 112, Toffset: 32448, EndToffset: 13248, ListType: "retroList"},
			},
		},
		{
			Sequence: []int{1, 2},
			Count:    2,
			Occurrences: []SequenceOccurrence{
				{Piece: 50, Voice: 4, Index: 1, EndIndex: 154, Toffset: 0, EndToffset: 19296, ListType: "intervList"},
				{Piece: 51, Voice: 3, Index: 112, EndIndex: 285, Toffset: 13248, EndToffset: 32448, ListType: "intervList"},
			},
		},
		{
			Sequence: []int{-1, -2},
			Count:    2,
			Occurrences: []SequenceOccurrence{
				{Piece: 50, Voice: 4, Index: 1, EndIndex: 154, Toffset: 0, EndToffset: 19296, ListType: "invertList"},
				{Piece: 51, Voice: 3, Index: 112, EndIndex: 285, Toffset: 13248, EndToffset: 32448, ListType: "invertList"},
			},
		},
		{
			Sequence: []int{-10, -20},
			Count:    2,
			Occurrences: []SequenceOccurrence{
				{Piece: 50, Voice: 4, Index: 154, EndIndex: 1, Toffset: 19296, EndToffset: 0, ListType: "retroInvertList"},
				{Piece: 51, Voice: 3, Index: 285, EndIndex: 112, Toffset: 32448, EndToffset: 13248, ListType: "retroInvertList"},
			},
		},
	}

	filtered := FilterRepeatedResults(res, 0)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Occurrences[0].ListType != "intervList" {
		t.Errorf("expected intervList to be chosen as highest priority, got %s", filtered[0].Occurrences[0].ListType)
	}

	// Test priority when intervList is not present (invertList should win)
	filteredInvert := FilterRepeatedResults([]SequenceResult{res[0], res[2], res[3]}, 0)
	if len(filteredInvert) != 1 || filteredInvert[0].Occurrences[0].ListType != "invertList" {
		t.Errorf("expected invertList to win when intervList is absent, got %v", filteredInvert)
	}

	// Test priority when invertList is absent (retroList should win over retroInvertList)
	filteredRetro := FilterRepeatedResults([]SequenceResult{res[0], res[3]}, 0)
	if len(filteredRetro) != 1 || filteredRetro[0].Occurrences[0].ListType != "retroList" {
		t.Errorf("expected retroList to win over retroInvertList, got %v", filteredRetro)
	}
}

func TestRepeatedSubstringsWithRestsDeduplication(t *testing.T) {
	// Piece 1 and Piece 2 have the same notes with a leading rest and trailing notes
	notes1 := []NoteRow{
		{Midi: 0, Bar: 1, Index: 1, Pitch: "0", Toffset: 0},
		{Midi: 0, Bar: 1, Index: 2, Pitch: "0", Toffset: 24},
		{Midi: 60, Bar: 2, Index: 3, Pitch: "C4", Toffset: 48},
		{Midi: 62, Bar: 2, Index: 4, Pitch: "D4", Toffset: 72},
		{Midi: 64, Bar: 3, Index: 5, Pitch: "E4", Toffset: 96},
		{Midi: 65, Bar: 3, Index: 6, Pitch: "F4", Toffset: 120},
		{Midi: 67, Bar: 4, Index: 7, Pitch: "G4", Toffset: 144},
	}
	notes2 := []NoteRow{
		{Midi: 0, Bar: 1, Index: 1, Pitch: "0", Toffset: 0},
		{Midi: 0, Bar: 1, Index: 2, Pitch: "0", Toffset: 24},
		{Midi: 60, Bar: 2, Index: 3, Pitch: "C4", Toffset: 48},
		{Midi: 62, Bar: 2, Index: 4, Pitch: "D4", Toffset: 72},
		{Midi: 64, Bar: 3, Index: 5, Pitch: "E4", Toffset: 96},
		{Midi: 65, Bar: 3, Index: 6, Pitch: "F4", Toffset: 120},
		{Midi: 67, Bar: 4, Index: 7, Pitch: "G4", Toffset: 144},
	}

	qb := &QueryBlock{
		Blocks: []*PieceIntervalSequences{
			{Piece: 50, IntervalSequences: []*IntervalSequence{mockIntervalSequence(50, 4, notes1)}},
			{Piece: 51, IntervalSequences: []*IntervalSequence{mockIntervalSequence(51, 3, notes2)}},
		},
	}

	finder := NewSequenceFinder(qb, nil)
	rawResults := finder.FindRepeatedSubstrings(5, 2, false)
	filtered := FilterRepeatedResults(rawResults, 0)

	if len(filtered) != 1 {
		t.Fatalf("expected exactly 1 filtered result (deduplicating recte and retro), got %d: %+v", len(filtered), filtered)
	}
	if filtered[0].Occurrences[0].ListType != "intervList" {
		t.Errorf("expected intervList result, got %s", filtered[0].Occurrences[0].ListType)
	}
}

func mockIntervalSequence(piece, voice int, notes []NoteRow) *IntervalSequence {
	values := make([]int, len(notes))
	for i, n := range notes {
		if n.Pitch == "0" {
			values[i] = 0
		} else {
			values[i] = n.Midi
		}
	}
	seq := &IntervalSequence{
		Piece:     piece,
		Voice:     voice,
		NotesData: notes,
	}
	seq.IntervList, seq.IntervMapping = BuildIntervList(values)
	seq.InvertList = Negate(seq.IntervList)
	seq.InvertMapping = seq.IntervMapping

	reversed := make([]int, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}
	seq.RetroList, seq.RetroMapping = BuildIntervList(reversed)
	n := len(values)
	for i, idx := range seq.RetroMapping {
		seq.RetroMapping[i] = (n - 1) - idx
	}
	seq.RetroInvertList = Negate(seq.RetroList)
	seq.RetroInvertMapping = seq.RetroMapping
	return seq
}

func TestFindRepeatedSubstrings(t *testing.T) {
	// Create notes that produce a matching interval sequence
	// Piece 1 notes: 60, 62, 64, 65, 67 -> intervals +3, +3, +2, +3
	notes1 := []NoteRow{
		{Midi: 60, Bar: 1, Index: 1, Pitch: "C4", Toffset: 0},
		{Midi: 62, Bar: 1, Index: 2, Pitch: "D4", Toffset: 24},
		{Midi: 64, Bar: 2, Index: 3, Pitch: "E4", Toffset: 48},
		{Midi: 65, Bar: 2, Index: 4, Pitch: "F4", Toffset: 72},
		{Midi: 67, Bar: 3, Index: 5, Pitch: "G4", Toffset: 96},
	}
	// Piece 2 notes: same interval pattern +3, +3, +2, +3
	notes2 := []NoteRow{
		{Midi: 65, Bar: 1, Index: 1, Pitch: "F4", Toffset: 0},
		{Midi: 67, Bar: 1, Index: 2, Pitch: "G4", Toffset: 24},
		{Midi: 69, Bar: 2, Index: 3, Pitch: "A4", Toffset: 48},
		{Midi: 70, Bar: 2, Index: 4, Pitch: "Bb4", Toffset: 72},
		{Midi: 72, Bar: 3, Index: 5, Pitch: "C5", Toffset: 96},
	}

	qb := &QueryBlock{
		Blocks: []*PieceIntervalSequences{
			{Piece: 1, IntervalSequences: []*IntervalSequence{mockIntervalSequence(1, 1, notes1)}},
			{Piece: 2, IntervalSequences: []*IntervalSequence{mockIntervalSequence(2, 1, notes2)}},
		},
	}

	finder := NewSequenceFinder(qb, nil)
	results := finder.FindRepeatedSubstrings(4, 2, false)

	if len(results) == 0 {
		t.Fatalf("expected at least 1 repeated substring of length >= 4, got 0")
	}

	// Verify occurrence counts
	found := false
	for _, r := range results {
		if len(r.Sequence) >= 4 && r.Count >= 2 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding matching pattern occurring across pieces")
	}

	// Test excludeSamePiece = true
	resultsExclude := finder.FindRepeatedSubstrings(4, 2, true)
	if len(resultsExclude) == 0 {
		t.Fatalf("expected cross-piece pattern with excludeSamePiece=true")
	}
}

func TestFindCrossPieceSubstrings(t *testing.T) {
	notes1 := []NoteRow{
		{Midi: 60, Bar: 1, Index: 1, Pitch: "C4", Toffset: 0},
		{Midi: 62, Bar: 1, Index: 2, Pitch: "D4", Toffset: 24},
		{Midi: 64, Bar: 2, Index: 3, Pitch: "E4", Toffset: 48},
		{Midi: 65, Bar: 2, Index: 4, Pitch: "F4", Toffset: 72},
		{Midi: 67, Bar: 3, Index: 5, Pitch: "G4", Toffset: 96},
	}
	notes2 := []NoteRow{
		{Midi: 65, Bar: 1, Index: 1, Pitch: "F4", Toffset: 0},
		{Midi: 67, Bar: 1, Index: 2, Pitch: "G4", Toffset: 24},
		{Midi: 69, Bar: 2, Index: 3, Pitch: "A4", Toffset: 48},
		{Midi: 70, Bar: 2, Index: 4, Pitch: "Bb4", Toffset: 72},
		{Midi: 72, Bar: 3, Index: 5, Pitch: "C5", Toffset: 96},
	}

	qb := &QueryBlock{
		Blocks: []*PieceIntervalSequences{
			{Piece: 1, IntervalSequences: []*IntervalSequence{mockIntervalSequence(1, 1, notes1)}},
			{Piece: 2, IntervalSequences: []*IntervalSequence{mockIntervalSequence(2, 1, notes2)}},
		},
	}

	finder := NewSequenceFinder(qb, nil)
	results := finder.FindCrossPieceSubstrings(1, 4)

	if len(results) == 0 {
		t.Fatalf("expected cross piece substrings from piece 1 in piece 2")
	}
}

func TestIntraPieceRepetition(t *testing.T) {
	// Pattern occurs twice in piece 1 only
	notes := []NoteRow{
		{Midi: 60, Bar: 1, Index: 1, Pitch: "C4", Toffset: 0},
		{Midi: 62, Bar: 1, Index: 2, Pitch: "D4", Toffset: 24},
		{Midi: 64, Bar: 2, Index: 3, Pitch: "E4", Toffset: 48},
		{Midi: 65, Bar: 2, Index: 4, Pitch: "F4", Toffset: 72},
		{Midi: 67, Bar: 3, Index: 5, Pitch: "G4", Toffset: 96},

		{Midi: 60, Bar: 4, Index: 6, Pitch: "C4", Toffset: 120},
		{Midi: 62, Bar: 4, Index: 7, Pitch: "D4", Toffset: 144},
		{Midi: 64, Bar: 5, Index: 8, Pitch: "E4", Toffset: 168},
		{Midi: 65, Bar: 5, Index: 9, Pitch: "F4", Toffset: 192},
		{Midi: 67, Bar: 6, Index: 10, Pitch: "G4", Toffset: 216},
	}

	qb := &QueryBlock{
		Blocks: []*PieceIntervalSequences{
			{Piece: 1, IntervalSequences: []*IntervalSequence{mockIntervalSequence(1, 1, notes)}},
		},
	}

	finder := NewSequenceFinder(qb, nil)

	// Intra-piece without excludeSamePiece -> should find results
	resInclude := finder.FindRepeatedSubstrings(4, 2, false)
	if len(resInclude) == 0 {
		t.Fatalf("expected intra-piece pattern to be found when excludeSamePiece=false")
	}

	// Intra-piece with excludeSamePiece -> should exclude because it only occurs in 1 piece
	resExclude := finder.FindRepeatedSubstrings(4, 2, true)
	if len(resExclude) != 0 {
		t.Fatalf("expected 0 results with excludeSamePiece=true for single piece, got %d", len(resExclude))
	}
}
