package smooth

import (
	"testing"
)

func TestSmoothPiece(t *testing.T) {
	rows := []DissRow{
		{Piece: 1, Offset: 0, Duration: 24, PitchesDissonant: false, MidiDissonant: false, MidiValues: []int{60, 64}},
		{Piece: 1, Offset: 24, Duration: 12, PitchesDissonant: true, MidiDissonant: true, MidiValues: []int{60, 65}},
		{Piece: 1, Offset: 36, Duration: 24, PitchesDissonant: false, MidiDissonant: false, MidiValues: []int{60, 67}},
	}

	smoothed := SmoothPiece(rows)
	if len(smoothed) == 0 {
		t.Fatal("expected smoothed rows")
	}
	for _, r := range smoothed {
		if r.IsDissonant() {
			t.Errorf("expected no dissonant rows left in resolved sequence, got %+v", r)
		}
	}
}

func TestChangedVoices(t *testing.T) {
	a := []int{60, 64, 67}
	b := []int{60, 65, 67}
	if got := changedVoices(a, b); got != 1 {
		t.Errorf("changedVoices got %d, want 1", got)
	}
}
