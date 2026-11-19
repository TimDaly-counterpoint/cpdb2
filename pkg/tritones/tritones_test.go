package tritones

import (
	"testing"
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
}
