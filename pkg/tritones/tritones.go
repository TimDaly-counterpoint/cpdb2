package tritones

import (
	"bufio"
	"context"
	"fmt"

	"cpdb2/pkg/db"
	"cpdb2/pkg/prompt"
)

// FindTritones finds and returns offsets with tritones in dissonant sonorities for a piece.
func FindTritones(ctx context.Context, database *db.DB, piece int) ([]float64, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT "offset", midi_values
		FROM disstable
		WHERE piece = $1 AND midi_dissonant = true
	`, piece)
	if err != nil {
		return nil, fmt.Errorf("query dissonant MIDI values for piece %d: %w", piece, err)
	}
	defer rows.Close()

	var tritoneOffsets []float64
	for rows.Next() {
		var offset float64
		var midiValues []int
		if err := rows.Scan(&offset, &midiValues); err != nil {
			return nil, fmt.Errorf("scan dissonant MIDI values: %w", err)
		}

		if ContainsTritone(midiValues) {
			tritoneOffsets = append(tritoneOffsets, offset)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dissonant MIDI values: %w", err)
	}

	return tritoneOffsets, nil
}

// ContainsTritone checks whether any pair of non-zero midi values forms a tritone interval (diff % 12 == 6).
func ContainsTritone(midiValues []int) bool {
	for i, a := range midiValues {
		if a == 0 {
			continue
		}
		for _, b := range midiValues[i+1:] {
			if b != 0 && absInt(a-b)%12 == 6 {
				return true
			}
		}
	}
	return false
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// RunInteractive prompts the user for piece identity and prints tritone offsets.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	piece, err := prompt.PromptInt(reader, "Enter piece identity", 1)
	if err != nil {
		return err
	}

	offsets, err := FindTritones(ctx, database, piece)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d tritone occurrence(s) in dissonant sonorities for piece %d:\n", len(offsets), piece)
	for _, o := range offsets {
		fmt.Println(o)
	}

	return nil
}
