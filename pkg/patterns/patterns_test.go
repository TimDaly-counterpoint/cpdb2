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
