package sonority

import (
	"bufio"
	"context"
	"fmt"

	"cpdb2/pkg/db"
	"cpdb2/pkg/smooth"
	"github.com/jackc/pgx/v5"
)

const (
	// TableSonority is the name of the sonority table.
	TableSonority = "sonority"
	// TableDissonance is the name of the dissonance table.
	TableDissonance = "dissonance"
	// TableConsonant is the name of the smoothed consonant table.
	TableConsonant = "consonant"
)

// ResultSummary holds record counts for created/populated tables.
type ResultSummary struct {
	SonorityCount   int64
	DissonanceCount int64
	ConsonantCount  int64
}

// RecreateSonorityTables creates or cleans sonority, dissonance, and consonant tables,
// populates sonority and dissonance from notes and voces, and generates smoothed consonant rows.
func RecreateSonorityTables(ctx context.Context, database *db.DB) (*ResultSummary, error) {
	// 1. Ensure tables exist
	createTablesSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS public.%s (
			piece integer,
			"offset" integer,
			duration integer,
			pitches character varying[],
			midi_values integer[],
			note_ids integer[],
			voices integer[]
		);

		CREATE TABLE IF NOT EXISTS public.%s (
			piece integer,
			"offset" integer,
			duration integer,
			pitches character varying[],
			midi_values integer[],
			note_ids integer[],
			voices integer[],
			pitches_dissonant boolean,
			midi_dissonant boolean
		);

		CREATE TABLE IF NOT EXISTS public.%s (
			piece integer,
			"offset" integer,
			duration integer,
			pitches character varying[],
			midi_values integer[],
			note_ids integer[],
			voices integer[],
			pitches_dissonant boolean,
			midi_dissonant boolean
		);
	`, pgx.Identifier{TableSonority}.Sanitize(), pgx.Identifier{TableDissonance}.Sanitize(), pgx.Identifier{TableConsonant}.Sanitize())

	if _, err := database.Pool.Exec(ctx, createTablesSQL); err != nil {
		return nil, fmt.Errorf("ensure tables exist: %w", err)
	}

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Delete existing data from the tables
	cleanTablesSQL := fmt.Sprintf(`
		DELETE FROM public.%s;
		DELETE FROM public.%s;
		DELETE FROM public.%s;
	`, pgx.Identifier{TableConsonant}.Sanitize(), pgx.Identifier{TableDissonance}.Sanitize(), pgx.Identifier{TableSonority}.Sanitize())

	if _, err := tx.Exec(ctx, cleanTablesSQL); err != nil {
		return nil, fmt.Errorf("clear existing table data: %w", err)
	}

	// 3. Populate sonority from notes table
	insertSonoritySQL := fmt.Sprintf(`
		INSERT INTO public.%s (piece, "offset", duration, pitches, midi_values, note_ids, voices)
		WITH change_points AS (
			SELECT notes.piece,
				notes.toffset AS point
			FROM public.notes
			UNION
			SELECT notes.piece,
				(notes.toffset + notes.duration) AS point
			FROM public.notes
		), intervals AS (
			SELECT change_points.piece,
				change_points.point AS startoffset,
				lead(change_points.point) OVER (PARTITION BY change_points.piece ORDER BY change_points.point) AS endoffset
			FROM change_points
		), sonority_notes AS (
			SELECT i.piece,
				i.startoffset,
				i.endoffset,
				(i.endoffset - i.startoffset) AS duration,
				n.id AS note_id,
				n.voice,
				n.pitch,
				n.midi,
				n.toffset
			FROM intervals i
				JOIN public.notes n ON (((n.piece = i.piece) AND (n.toffset <= i.startoffset) AND ((n.toffset + n.duration) >= i.endoffset)))
			WHERE (i.startoffset < i.endoffset)
		), elementary_sonorities AS (
			SELECT sonority_notes.piece,
				sonority_notes.startoffset,
				sonority_notes.endoffset,
				sonority_notes.duration,
				max(sonority_notes.toffset) AS newest_note_offset,
				array_agg(sonority_notes.pitch ORDER BY sonority_notes.voice) AS pitches,
				array_agg(sonority_notes.midi ORDER BY sonority_notes.voice) AS midi_values,
				array_agg(sonority_notes.note_id ORDER BY sonority_notes.voice) AS note_ids,
				array_agg(sonority_notes.voice ORDER BY sonority_notes.voice) AS voices
			FROM sonority_notes
			GROUP BY sonority_notes.piece, sonority_notes.startoffset, sonority_notes.endoffset, sonority_notes.duration
		), islands AS (
			SELECT elementary_sonorities.piece,
				elementary_sonorities.startoffset,
				elementary_sonorities.endoffset,
				elementary_sonorities.duration,
				elementary_sonorities.newest_note_offset,
				elementary_sonorities.pitches,
				elementary_sonorities.midi_values,
				elementary_sonorities.note_ids,
				elementary_sonorities.voices,
				CASE
					WHEN (lag(elementary_sonorities.pitches) OVER (PARTITION BY elementary_sonorities.piece ORDER BY elementary_sonorities.startoffset) IS DISTINCT FROM elementary_sonorities.pitches) THEN 1
					ELSE 0
				END AS is_new_sonority
			FROM elementary_sonorities
		), groups AS (
			SELECT islands.piece,
				islands.startoffset,
				islands.endoffset,
				islands.duration,
				islands.newest_note_offset,
				islands.pitches,
				islands.midi_values,
				islands.note_ids,
				islands.voices,
				islands.is_new_sonority,
				sum(islands.is_new_sonority) OVER (PARTITION BY islands.piece ORDER BY islands.startoffset) AS group_id
			FROM islands
		), merged_groups AS (
			SELECT groups.piece,
				groups.group_id,
				min(groups.startoffset) AS "offset",
				(max(groups.endoffset) - min(groups.startoffset)) AS duration,
				groups.pitches,
				groups.midi_values,
				groups.voices
			FROM groups
			GROUP BY groups.piece, groups.group_id, groups.pitches, groups.midi_values, groups.voices
		), merged_note_ids AS (
			SELECT g.piece,
				g.group_id,
				array_agg(DISTINCT u.note_id ORDER BY u.note_id) AS note_ids
			FROM groups g,
				LATERAL unnest(g.note_ids) u(note_id)
			GROUP BY g.piece, g.group_id
		)
		SELECT mg.piece,
			mg."offset",
			mg.duration,
			mg.pitches,
			mg.midi_values,
			mn.note_ids,
			mg.voices
		FROM (merged_groups mg
			JOIN merged_note_ids mn ON (((mg.piece = mn.piece) AND (mg.group_id = mn.group_id))))
		ORDER BY mg.piece, mg."offset";
	`, pgx.Identifier{TableSonority}.Sanitize())

	tagSonority, err := tx.Exec(ctx, insertSonoritySQL)
	if err != nil {
		return nil, fmt.Errorf("populate %s table: %w", TableSonority, err)
	}

	// 4. Populate dissonance from sonority and voces tables
	insertDissonanceSQL := fmt.Sprintf(`
		INSERT INTO public.%s (piece, "offset", duration, pitches, midi_values, note_ids, voices, pitches_dissonant, midi_dissonant)
		SELECT s.piece,
			s."offset",
			s.duration,
			s.pitches,
			s.midi_values,
			s.note_ids,
			s.voices,
			flags.pitches_dissonant,
			flags.midi_dissonant
		FROM (public.%s s
			CROSS JOIN LATERAL ( WITH sonority_notes AS (
					SELECT n.pitch,
						n.midi,
						n.ordinality
					FROM UNNEST(s.pitches, s.midi_values) WITH ORDINALITY n(pitch, midi, ordinality)
					WHERE ((n.pitch IS NOT NULL) AND ((n.pitch)::text <> ''::text) AND ((n.pitch)::text <> '0'::text) AND (n.midi IS NOT NULL) AND (n.midi <> 0))
				), pair_values AS (
					SELECT a.pitch AS pitch_a,
						b.pitch AS pitch_b,
						LEAST(a.midi, b.midi) AS lower_midi,
						CASE
							WHEN (a.midi < b.midi) THEN a.pitch
							ELSE b.pitch
						END AS lower_pitch,
						(abs((a.midi - b.midi)) %% 12) AS semitone_interval,
						((to_jsonb(va.*) ->> (b.pitch)::text))::integer AS interval_value
					FROM ((sonority_notes a
						JOIN sonority_notes b ON ((a.ordinality < b.ordinality)))
						JOIN public.voces va ON (((va.pitch)::text = (a.pitch)::text)))
				), pair_support AS (
					SELECT p.pitch_a,
						p.pitch_b,
						p.lower_midi,
						p.lower_pitch,
						p.semitone_interval,
						p.interval_value,
						COALESCE((EXISTS ( SELECT 1
							FROM (sonority_notes support_note
								JOIN public.voces lower_v ON (((lower_v.pitch)::text = (p.lower_pitch)::text)))
							WHERE ((support_note.midi < p.lower_midi) AND ((((to_jsonb(lower_v.*) ->> (support_note.pitch)::text))::integer %% '-7'::integer) = ANY (ARRAY['-2'::integer, '-4'::integer]))))), false) AS has_lower_voces_support,
						COALESCE((NOT (EXISTS ( SELECT 1
							FROM sonority_notes support_note
							WHERE ((support_note.midi < p.lower_midi) AND (((p.lower_midi - support_note.midi) %% 12) = ANY (ARRAY[3, 4, 7])))))), true) AS has_no_midi_support
					FROM pair_values p
				), pair_flags AS (
					SELECT p.pitch_a,
						p.pitch_b,
						p.lower_midi,
						p.lower_pitch,
						p.semitone_interval,
						p.interval_value,
						p.has_lower_voces_support,
						p.has_no_midi_support,
						(((abs(p.interval_value) %% 7) = ANY (ARRAY[1, 6])) OR (((abs(p.interval_value) %% 7) = 3) AND (NOT p.has_lower_voces_support))) AS is_voces_dissonant
					FROM pair_support p
				)
			SELECT COALESCE(bool_or(pair_flags.is_voces_dissonant), false) AS pitches_dissonant,
				COALESCE(bool_or((pair_flags.semitone_interval = 6) OR (pair_flags.is_voces_dissonant AND ((pair_flags.semitone_interval = ANY (ARRAY[1, 2, 6, 10, 11])) OR ((pair_flags.semitone_interval = 5) AND pair_flags.has_no_midi_support)))), false) AS midi_dissonant
			FROM pair_flags) flags)
		ORDER BY s.piece, s."offset";
	`, pgx.Identifier{TableDissonance}.Sanitize(), pgx.Identifier{TableSonority}.Sanitize())

	tagDissonance, err := tx.Exec(ctx, insertDissonanceSQL)
	if err != nil {
		return nil, fmt.Errorf("populate %s table: %w", TableDissonance, err)
	}

	// 5. Query dissonance rows for smoothing into consonant table
	queryDissSQL := fmt.Sprintf(`
		SELECT piece, "offset", duration, pitches, midi_values, note_ids, voices, pitches_dissonant, midi_dissonant
		FROM public.%s
		ORDER BY piece, "offset"
	`, pgx.Identifier{TableDissonance}.Sanitize())

	rows, err := tx.Query(ctx, queryDissSQL)
	if err != nil {
		return nil, fmt.Errorf("query %s for smoothing: %w", TableDissonance, err)
	}
	defer rows.Close()

	var allDissRows []smooth.DissRow
	for rows.Next() {
		var r smooth.DissRow
		if err := rows.Scan(&r.Piece, &r.Offset, &r.Duration, &r.Pitches, &r.MidiValues, &r.NoteIDs, &r.Voices, &r.PitchesDissonant, &r.MidiDissonant); err != nil {
			return nil, fmt.Errorf("scan dissonance row: %w", err)
		}
		allDissRows = append(allDissRows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dissonance rows: %w", err)
	}
	rows.Close()

	// 6. Smooth piece by piece
	var allSmoothedRows []smooth.DissRow
	if len(allDissRows) > 0 {
		startIdx := 0
		for i := 1; i <= len(allDissRows); i++ {
			if i == len(allDissRows) || allDissRows[i].Piece != allDissRows[startIdx].Piece {
				pieceSlice := allDissRows[startIdx:i]
				smoothedPiece := smooth.SmoothPiece(pieceSlice)
				allSmoothedRows = append(allSmoothedRows, smoothedPiece...)
				startIdx = i
			}
		}
	}

	// 7. Insert smoothed rows into consonant table
	columns := []string{"piece", "offset", "duration", "pitches", "midi_values", "note_ids", "voices", "pitches_dissonant", "midi_dissonant"}
	tagConsonant, err := tx.CopyFrom(ctx, pgx.Identifier{TableConsonant}, columns, pgx.CopyFromSlice(len(allSmoothedRows), func(i int) ([]any, error) {
		r := allSmoothedRows[i]
		return []any{r.Piece, r.Offset, r.Duration, r.Pitches, r.MidiValues, r.NoteIDs, r.Voices, r.PitchesDissonant, r.MidiDissonant}, nil
	}))
	if err != nil {
		return nil, fmt.Errorf("copy rows into %s: %w", TableConsonant, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &ResultSummary{
		SonorityCount:   tagSonority.RowsAffected(),
		DissonanceCount: tagDissonance.RowsAffected(),
		ConsonantCount:  tagConsonant,
	}, nil
}

// RunInteractive executes table recreation for sonority, dissonance, and consonant tables.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	_ = reader
	fmt.Println("Recreating sonority, dissonance, and consonant tables across all pieces...")
	summary, err := RecreateSonorityTables(ctx, database)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully updated tables:\n")
	fmt.Printf("  - %s: %d rows inserted\n", TableSonority, summary.SonorityCount)
	fmt.Printf("  - %s: %d rows inserted\n", TableDissonance, summary.DissonanceCount)
	fmt.Printf("  - %s: %d rows inserted\n", TableConsonant, summary.ConsonantCount)
	return nil
}
