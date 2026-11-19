package smooth

import (
	"bufio"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"cpdb2/pkg/db"
	"cpdb2/pkg/prompt"
)

// DissRow represents a row in the disstable.
type DissRow struct {
	Piece            int
	Offset           int
	Duration         int
	Pitches          []string
	MidiValues       []int
	NoteIDs          []int
	Voices           []int
	PitchesDissonant bool
	MidiDissonant    bool
	Bar              *int
}

func (r DissRow) End() int {
	return r.Offset + r.Duration
}

func (r DissRow) IsDissonant() bool {
	return r.PitchesDissonant || r.MidiDissonant
}

// SmoothPiece applies dissonance smoothing across rows for a single piece.
func SmoothPiece(rows []DissRow) []DissRow {
	var out []DissRow
	var current *DissRow

	i := 0
	for i < len(rows) {
		if !rows[i].IsDissonant() {
			if current != nil {
				out = append(out, *current)
			}
			row := rows[i]
			current = &row
			i++
			continue
		}

		j := i
		for j < len(rows) && rows[j].IsDissonant() {
			j++
		}
		block := rows[i:j]

		var following *DissRow
		if j < len(rows) {
			row := rows[j]
			following = &row
		}

		if current == nil && following == nil {
			out = append(out, block...)
			i = j
			continue
		}

		prev, next := resolveBlock(current, block, following)
		if next == nil {
			current = prev
		} else {
			if prev != nil {
				out = append(out, *prev)
			}
			current = next
		}
		i = j + 1
	}

	if current != nil {
		out = append(out, *current)
	}
	return out
}

func resolveBlock(prev *DissRow, block []DissRow, following *DissRow) (*DissRow, *DissRow) {
	bestSplit, bestChanges, bestMisses := -1, 0, 0
	for split := 0; split <= len(block); split++ {
		if (split > 0 && prev == nil) || (split < len(block) && following == nil) {
			continue
		}

		changes, misses := 0, 0
		for idx, row := range block {
			toPrev := idx < split
			target := following
			if toPrev {
				target = prev
			}
			changes += changedVoices(row.MidiValues, target.MidiValues)
			if toPrev == (row.Offset%48 < 24) {
				misses++
			}
		}

		if bestSplit < 0 || changes < bestChanges || (changes == bestChanges && misses < bestMisses) {
			bestSplit, bestChanges, bestMisses = split, changes, misses
		}
	}

	if bestSplit < 0 {
		return prev, following
	}

	var newPrev, newNext *DissRow
	if prev != nil {
		merged := mergeRows(*prev, block[:bestSplit])
		newPrev = &merged
	}
	if following != nil {
		merged := mergeRows(*following, block[bestSplit:])
		newNext = &merged
	}
	return newPrev, newNext
}

func mergeRows(anchor DissRow, rows []DissRow) DissRow {
	if len(rows) == 0 {
		return anchor
	}

	start, end := anchor.Offset, anchor.End()
	bar := anchor.Bar
	for _, row := range rows {
		if row.Offset < start {
			start = row.Offset
			bar = row.Bar
		}
		if row.End() > end {
			end = row.End()
		}
	}

	anchor.Offset = start
	anchor.Duration = end - start
	anchor.Bar = bar
	return anchor
}

func changedVoices(a, b []int) int {
	changes := 0
	for i := 0; i < len(a) || i < len(b); i++ {
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			changes++
		}
	}
	return changes
}

// WriteSmoothTable writes smoothed rows into smoothcp_<piece> table.
func WriteSmoothTable(ctx context.Context, database *db.DB, tableName string, rows []DissRow) error {
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (LIKE disstable)`, pgx.Identifier{tableName}.Sanitize()))
	if err != nil {
		return fmt.Errorf("create table %q: %w", tableName, err)
	}

	columns := []string{"piece", "offset", "duration", "pitches", "midi_values", "note_ids", "voices", "pitches_dissonant", "midi_dissonant", "bar"}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{tableName}, columns, pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
		r := rows[i]
		return []any{r.Piece, r.Offset, r.Duration, r.Pitches, r.MidiValues, r.NoteIDs, r.Voices, r.PitchesDissonant, r.MidiDissonant, r.Bar}, nil
	}))
	if err != nil {
		return fmt.Errorf("copy rows into %q: %w", tableName, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// SmoothDissTable runs dissonance smoothing for a piece and creates the smoothed table.
func SmoothDissTable(ctx context.Context, database *db.DB, piece int) (string, int, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT piece, "offset", duration, pitches, midi_values, note_ids, voices, pitches_dissonant, midi_dissonant, bar
		FROM disstable
		WHERE piece = $1
		ORDER BY "offset"
	`, piece)
	if err != nil {
		return "", 0, fmt.Errorf("query disstable for piece %d: %w", piece, err)
	}
	defer rows.Close()

	var pieceRows []DissRow
	for rows.Next() {
		var r DissRow
		if err := rows.Scan(&r.Piece, &r.Offset, &r.Duration, &r.Pitches, &r.MidiValues, &r.NoteIDs, &r.Voices, &r.PitchesDissonant, &r.MidiDissonant, &r.Bar); err != nil {
			return "", 0, fmt.Errorf("scan disstable row: %w", err)
		}
		pieceRows = append(pieceRows, r)
	}
	if err := rows.Err(); err != nil {
		return "", 0, fmt.Errorf("iterate disstable: %w", err)
	}
	rows.Close()

	if len(pieceRows) == 0 {
		return "", 0, fmt.Errorf("no disstable rows found for piece %d", piece)
	}

	smoothed := SmoothPiece(pieceRows)

	tableName := fmt.Sprintf("smoothcp_%d", piece)
	if err := WriteSmoothTable(ctx, database, tableName, smoothed); err != nil {
		return "", 0, err
	}

	return tableName, len(smoothed), nil
}

// RunInteractive prompts the user for piece identity and smooths the table.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	piece, err := prompt.PromptInt(reader, "Enter piece identity", 1)
	if err != nil {
		return err
	}

	tableName, count, err := SmoothDissTable(ctx, database, piece)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %d rows into %q\n", count, tableName)
	return nil
}
