package cmatrix

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"cpdb2/pkg/db"
	"cpdb2/pkg/music"
	"cpdb2/pkg/prompt"
	"github.com/jackc/pgx/v5"
)

const (
	// VersionFlorid specifies analysis of the florid version (from sonority table).
	VersionFlorid = "f"
	// VersionConsonant specifies analysis of the smoothed consonant version (from consonant table).
	VersionConsonant = "c"
)

// MelodicNote represents a sounding note with its melodic interval relative to the first note in the voice.
type MelodicNote struct {
	Voice     int
	Bar       int
	Index     int
	Pitch     string
	Duration  float64
	Toffset   float64
	Interval  int
	BarOffset float64
}

// VoiceDifference records a position where voices differ in pitch.
type VoiceDifference struct {
	Position int
	Notes    []MelodicNote
}

// Result contains the analysis result from ProcessCMatrix.
type Result struct {
	Piece          int
	Version        string
	Voices         []int
	VoiceNotes     map[int][]MelodicNote
	Differences    []VoiceDifference
	Matrix         [][]string
	CSVFilename    string
	UniqueToffsets []float64
}

// CalculateVoiceIntervals converts a sequence of note entries for a single voice into melodic intervals.
func CalculateVoiceIntervals(notes []db.SequenceNote) []MelodicNote {
	return CalculateVoiceIntervalsWithBarStarts(notes, nil)
}

// CalculateVoiceIntervalsWithBarStarts converts a sequence of note entries for a single voice into melodic intervals,
// using the provided bar start offsets mapping if available.
func CalculateVoiceIntervalsWithBarStarts(notes []db.SequenceNote, customBarStarts map[int]float64) []MelodicNote {
	var melodicNotes []MelodicNote
	var firstPitch string
	hasSoundingNote := false

	barStartOffsets := make(map[int]float64)
	if customBarStarts != nil {
		for k, v := range customBarStarts {
			barStartOffsets[k] = v
		}
	}
	for _, n := range notes {
		if _, exists := barStartOffsets[n.Bar]; !exists {
			barStartOffsets[n.Bar] = n.Toffset
		}
	}

	for _, n := range notes {
		if n.Pitch == "0" || n.Pitch == "" {
			continue
		}

		var interval int
		if !hasSoundingNote {
			interval = 0
			firstPitch = n.Pitch
			hasSoundingNote = true
		} else {
			currVal := music.Voces[n.Pitch]
			firstVal := music.Voces[firstPitch]
			interval = currVal - firstVal
		}

		barOffset := n.Toffset - barStartOffsets[n.Bar]

		melodicNotes = append(melodicNotes, MelodicNote{
			Voice:     n.Voice,
			Bar:       n.Bar,
			Index:     n.Index,
			Pitch:     n.Pitch,
			Duration:  n.Duration,
			Toffset:   n.Toffset,
			Interval:  interval,
			BarOffset: barOffset,
		})
	}

	return melodicNotes
}

// FindVoiceDifferences compares the sounding note sequences of the surveyed voices.
func FindVoiceDifferences(voiceNotes map[int][]MelodicNote, voiceOrder []int) []VoiceDifference {
	if len(voiceOrder) < 2 {
		return nil
	}

	maxNotes := 0
	for _, v := range voiceOrder {
		if len(voiceNotes[v]) > maxNotes {
			maxNotes = len(voiceNotes[v])
		}
	}

	var diffs []VoiceDifference

	for pos := 0; pos < maxNotes; pos++ {
		var notesAtPos []MelodicNote
		var firstPitch string
		hasDifference := false
		count := 0

		for _, v := range voiceOrder {
			notes := voiceNotes[v]
			if pos < len(notes) {
				n := notes[pos]
				notesAtPos = append(notesAtPos, n)
				if count == 0 {
					firstPitch = n.Pitch
				} else if n.Pitch != firstPitch {
					hasDifference = true
				}
				count++
			} else {
				hasDifference = true
			}
		}

		if hasDifference && len(notesAtPos) > 0 {
			diffs = append(diffs, VoiceDifference{
				Position: pos + 1,
				Notes:    notesAtPos,
			})
		}
	}

	return diffs
}

// FormatVoiceDifferences formats the list of differences into a readable string.
func FormatVoiceDifferences(diffs []VoiceDifference) string {
	if len(diffs) == 0 {
		return "No pitch differences found across the surveyed voices (disregarding initial rests).\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d position(s) with pitch differences across voices (disregarding initial rests):\n", len(diffs)))
	for _, d := range diffs {
		sb.WriteString(fmt.Sprintf("Position %d:\n", d.Position))
		for _, n := range d.Notes {
			sb.WriteString(fmt.Sprintf("  Voice %d: Pitch = %-3s | Bar = %-4d | Index = %-4d | Interval = %-3d\n",
				n.Voice, n.Pitch, n.Bar, n.Index, n.Interval))
		}
	}
	return sb.String()
}

// GenerateIntervalMatrix creates a 2D string grid of melodic intervals and pairwise voice intervals aligned by toffset.
func GenerateIntervalMatrix(voiceNotes map[int][]MelodicNote, voiceOrder []int) ([][]string, []float64) {
	if len(voiceOrder) == 0 {
		return nil, nil
	}

	toffsetSet := make(map[float64]bool)
	for _, v := range voiceOrder {
		for _, n := range voiceNotes[v] {
			toffsetSet[n.Toffset] = true
		}
	}

	toffsetList := make([]float64, 0, len(toffsetSet))
	for t := range toffsetSet {
		toffsetList = append(toffsetList, t)
	}
	slices.Sort(toffsetList)

	numCols := 1 + len(toffsetList)

	makeEmptyRow := func() []string {
		return make([]string, numCols)
	}

	makeDividerRow := func() []string {
		row := make([]string, numCols)
		for i := 0; i < numCols; i++ {
			row[i] = "_____"
		}
		return row
	}

	var matrix [][]string

	minBarToffset := make(map[int]float64)
	for _, v := range voiceOrder {
		for _, n := range voiceNotes[v] {
			if minT, exists := minBarToffset[n.Bar]; !exists || n.Toffset < minT {
				minBarToffset[n.Bar] = n.Toffset
			}
		}
	}

	type colInfo struct {
		bar       int
		barOffset float64
	}
	colMap := make(map[float64]colInfo)
	for _, v := range voiceOrder {
		for _, n := range voiceNotes[v] {
			if _, exists := colMap[n.Toffset]; !exists {
				barOffset := n.BarOffset
				if barOffset == 0 && n.Toffset > minBarToffset[n.Bar] {
					barOffset = n.Toffset - minBarToffset[n.Bar]
				}
				colMap[n.Toffset] = colInfo{
					bar:       n.Bar,
					barOffset: barOffset,
				}
			}
		}
	}

	// Header row
	headerRow := make([]string, numCols)
	headerRow[0] = "Offset"
	for i, t := range toffsetList {
		info := colMap[t]
		headerRow[i+1] = fmt.Sprintf("%d/%s", info.bar, strconv.FormatFloat(info.barOffset, 'f', -1, 64))
	}
	matrix = append(matrix, headerRow)

	findSoundingNote := func(notes []MelodicNote, t float64) *MelodicNote {
		for i := range notes {
			if notes[i].Toffset <= t && t < notes[i].Toffset+notes[i].Duration-1e-6 {
				return &notes[i]
			}
		}
		return nil
	}

	createVoiceRow := func(v int) []string {
		row := make([]string, numCols)
		row[0] = fmt.Sprintf("Voice %d", v)
		for colIdx, t := range toffsetList {
			col := colIdx + 1
			n := findSoundingNote(voiceNotes[v], t)
			if n != nil {
				if n.Toffset == t {
					row[col] = strconv.Itoa(n.Interval)
				} else {
					row[col] = "-"
				}
			}
		}
		return row
	}

	createIntervalRow := func(vUpper, vLower int) []string {
		row := make([]string, numCols)
		row[0] = fmt.Sprintf("v%d+v%d", vLower, vUpper)
		for colIdx, t := range toffsetList {
			col := colIdx + 1
			nUpper := findSoundingNote(voiceNotes[vUpper], t)
			nLower := findSoundingNote(voiceNotes[vLower], t)
			if nUpper != nil && nLower != nil {
				interval := music.Voces[nUpper.Pitch] - music.Voces[nLower.Pitch]
				row[col] = strconv.Itoa(interval)
			}
		}
		return row
	}

	type voicePair struct {
		v1, v2 int
	}
	calculatedPairs := make(map[voicePair]bool)

	for currIdx, currVoice := range voiceOrder {
		matrix = append(matrix, makeEmptyRow())
		matrix = append(matrix, createVoiceRow(currVoice))
		matrix = append(matrix, makeEmptyRow())

		for i := 0; i < currIdx; i++ {
			otherVoice := voiceOrder[i]
			pair := voicePair{v1: min(otherVoice, currVoice), v2: max(otherVoice, currVoice)}
			if !calculatedPairs[pair] {
				vUpper := min(otherVoice, currVoice)
				vLower := max(otherVoice, currVoice)
				matrix = append(matrix, createIntervalRow(vUpper, vLower))
				matrix = append(matrix, makeEmptyRow())
				calculatedPairs[pair] = true
			}
		}

		for i := currIdx + 1; i < len(voiceOrder); i++ {
			otherVoice := voiceOrder[i]
			pair := voicePair{v1: min(currVoice, otherVoice), v2: max(currVoice, otherVoice)}
			if !calculatedPairs[pair] {
				vUpper := min(currVoice, otherVoice)
				vLower := max(currVoice, otherVoice)
				matrix = append(matrix, createIntervalRow(vUpper, vLower))
				matrix = append(matrix, makeEmptyRow())
				calculatedPairs[pair] = true
			}
		}

		matrix = append(matrix, makeDividerRow())
	}

	return matrix, toffsetList
}

// WriteIntervalMatrixCSV writes the interval matrix to a CSV file.
func WriteIntervalMatrixCSV(filename string, matrix [][]string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create csv file %s: %w", filename, err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close csv file %s: %w", filename, cerr)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.WriteAll(matrix); err != nil {
		return fmt.Errorf("write csv data: %w", err)
	}

	return nil
}

// SonorityRow represents a vertical slice row from sonority or consonant table.
type SonorityRow struct {
	Offset     int
	Duration   int
	Pitches    []string
	MidiValues []int
	Voices     []int
}

// FetchPieceBars retrieves bar start offsets from notes table for a piece.
func FetchPieceBars(ctx context.Context, database *db.DB, piece int) (map[int]float64, []int, error) {
	rows, err := database.Pool.Query(ctx, "SELECT bar, MIN(toffset) FROM notes WHERE piece = $1 GROUP BY bar ORDER BY MIN(toffset), bar", piece)
	if err != nil {
		return nil, nil, fmt.Errorf("query bars for piece %d: %w", piece, err)
	}
	defer rows.Close()

	barStarts := make(map[int]float64)
	var barOrder []int
	for rows.Next() {
		var bar int
		var start float64
		if err := rows.Scan(&bar, &start); err != nil {
			return nil, nil, fmt.Errorf("scan bar: %w", err)
		}
		barStarts[bar] = start
		barOrder = append(barOrder, bar)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate bars: %w", err)
	}
	return barStarts, barOrder, nil
}

func getBarForOffset(offset float64, barStarts map[int]float64, barOrder []int) (int, float64) {
	if len(barOrder) == 0 {
		return 1, offset
	}
	selectedBar := barOrder[0]
	for _, b := range barOrder {
		if barStarts[b] <= offset {
			selectedBar = b
		} else {
			break
		}
	}
	return selectedBar, offset - barStarts[selectedBar]
}

// FetchSonorityRows retrieves all sonority/consonant slices for a piece from the specified table.
func FetchSonorityRows(ctx context.Context, database *db.DB, tableName string, piece int) ([]SonorityRow, error) {
	query := fmt.Sprintf(`
		SELECT "offset", duration, pitches, midi_values, voices
		FROM public.%s
		WHERE piece = $1
		ORDER BY "offset"
	`, pgx.Identifier{tableName}.Sanitize())

	rows, err := database.Pool.Query(ctx, query, piece)
	if err != nil {
		return nil, fmt.Errorf("query table %q for piece %d: %w", tableName, piece, err)
	}
	defer rows.Close()

	var sonRows []SonorityRow
	for rows.Next() {
		var r SonorityRow
		if err := rows.Scan(&r.Offset, &r.Duration, &r.Pitches, &r.MidiValues, &r.Voices); err != nil {
			return nil, fmt.Errorf("scan %s row: %w", tableName, err)
		}
		sonRows = append(sonRows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s rows: %w", tableName, err)
	}
	return sonRows, nil
}

// BuildVoiceNotesFromSonorities extracts sequence notes for a specific voice from sonority/consonant rows.
func BuildVoiceNotesFromSonorities(sonRows []SonorityRow, voice int, barStarts map[int]float64, barOrder []int) []db.SequenceNote {
	var rawNotes []db.SequenceNote
	currOffset := 0

	for _, s := range sonRows {
		if s.Offset > currOffset {
			gapDur := s.Offset - currOffset
			bar, _ := getBarForOffset(float64(currOffset), barStarts, barOrder)
			rawNotes = append(rawNotes, db.SequenceNote{
				Voice:    voice,
				Bar:      bar,
				Pitch:    "0",
				Duration: float64(gapDur),
				Midi:     0,
				Toffset:  float64(currOffset),
			})
			currOffset = s.Offset
		}

		pitch := "0"
		midi := 0
		for idx, vID := range s.Voices {
			if vID == voice {
				if idx < len(s.Pitches) {
					pitch = s.Pitches[idx]
				}
				if idx < len(s.MidiValues) {
					midi = s.MidiValues[idx]
				}
				break
			}
		}

		if s.Duration > 0 {
			bar, _ := getBarForOffset(float64(currOffset), barStarts, barOrder)
			rawNotes = append(rawNotes, db.SequenceNote{
				Voice:    voice,
				Bar:      bar,
				Pitch:    pitch,
				Duration: float64(s.Duration),
				Midi:     midi,
				Toffset:  float64(currOffset),
			})
			currOffset += s.Duration
		}
	}

	// Merge contiguous notes of identical pitch in the voice
	var mergedNotes []db.SequenceNote
	for _, n := range rawNotes {
		if len(mergedNotes) > 0 {
			last := &mergedNotes[len(mergedNotes)-1]
			if last.Pitch == n.Pitch && (last.Toffset+last.Duration) == n.Toffset {
				last.Duration += n.Duration
				continue
			}
		}
		mergedNotes = append(mergedNotes, n)
	}

	// Assign sequential 1-based index to notes
	for i := range mergedNotes {
		mergedNotes[i].Index = i + 1
	}

	return mergedNotes
}

// ProcessCMatrix runs the interval extraction, difference checking, and CSV generation for a piece and voices.
// The version parameter specifies either florid (VersionFlorid / "f", built from sonority table) or
// consonant (VersionConsonant / "c", built from consonant table).
func ProcessCMatrix(ctx context.Context, database *db.DB, piece int, voices []int, version string, customCSVFile string) (*Result, error) {
	normVersion := strings.ToLower(strings.TrimSpace(version))
	var tableName string
	var resolvedVersion string
	if normVersion == "" || normVersion == VersionFlorid || normVersion == "florid" {
		tableName = "sonority"
		resolvedVersion = VersionFlorid
	} else if normVersion == VersionConsonant || normVersion == "consonant" {
		tableName = "consonant"
		resolvedVersion = VersionConsonant
	} else {
		return nil, fmt.Errorf("invalid version %q (must be 'f' for florid or 'c' for consonant)", version)
	}

	if len(voices) == 0 {
		// Discover voices for this piece from the selected table
		rows, err := database.Pool.Query(ctx, fmt.Sprintf(`
			SELECT DISTINCT unnest(voices) AS v
			FROM public.%s
			WHERE piece = $1
			ORDER BY v
		`, pgx.Identifier{tableName}.Sanitize()), piece)
		if err == nil {
			for rows.Next() {
				var v int
				if err := rows.Scan(&v); err == nil {
					voices = append(voices, v)
				}
			}
			rows.Close()
		}

		if len(voices) == 0 {
			// Fallback to notes table
			rows, err := database.Pool.Query(ctx, "SELECT DISTINCT voice FROM notes WHERE piece = $1 ORDER BY voice", piece)
			if err == nil {
				for rows.Next() {
					var v int
					if err := rows.Scan(&v); err == nil {
						voices = append(voices, v)
					}
				}
				rows.Close()
			}
		}
	}

	if len(voices) == 0 {
		return nil, fmt.Errorf("no voices found for piece %d in table %q", piece, tableName)
	}

	voiceOrder := slices.Clone(voices)
	slices.Sort(voiceOrder)
	voiceOrder = slices.Compact(voiceOrder)

	barStarts, barOrder, err := FetchPieceBars(ctx, database, piece)
	if err != nil {
		barStarts = make(map[int]float64)
		barOrder = nil
	}

	sonRows, err := FetchSonorityRows(ctx, database, tableName, piece)
	if err != nil {
		return nil, err
	}
	if len(sonRows) == 0 {
		return nil, fmt.Errorf("no records found in table %q for piece %d", tableName, piece)
	}

	voiceMelodicNotes := make(map[int][]MelodicNote)
	for _, v := range voiceOrder {
		voiceNotes := BuildVoiceNotesFromSonorities(sonRows, v, barStarts, barOrder)
		voiceMelodicNotes[v] = CalculateVoiceIntervalsWithBarStarts(voiceNotes, barStarts)
	}

	diffs := FindVoiceDifferences(voiceMelodicNotes, voiceOrder)
	matrix, uniqueToffsets := GenerateIntervalMatrix(voiceMelodicNotes, voiceOrder)

	csvFilename := customCSVFile
	if csvFilename == "" {
		timestamp := time.Now().Format("20060102_150405")
		csvFilename = fmt.Sprintf("cmatrix_%d_%s.csv", piece, timestamp)
	}

	if err := WriteIntervalMatrixCSV(csvFilename, matrix); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}

	return &Result{
		Piece:          piece,
		Version:        resolvedVersion,
		Voices:         voiceOrder,
		VoiceNotes:     voiceMelodicNotes,
		Differences:    diffs,
		Matrix:         matrix,
		CSVFilename:    csvFilename,
		UniqueToffsets: uniqueToffsets,
	}, nil
}

// RunInteractive prompts the user and executes CMatrix analysis.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	piece, err := prompt.PromptInt(reader, "Enter piece ID to examine", 1)
	if err != nil {
		return err
	}

	versionStr, err := prompt.PromptString(reader, "Apply process to florid (f) or consonant (c) version", "f")
	if err != nil {
		return err
	}
	versionStr = strings.ToLower(strings.TrimSpace(versionStr))
	var version string
	if versionStr == "f" || versionStr == "florid" || versionStr == "" {
		version = VersionFlorid
	} else if versionStr == "c" || versionStr == "consonant" {
		version = VersionConsonant
	} else {
		return fmt.Errorf("invalid version %q (must be 'f' for florid or 'c' for consonant)", versionStr)
	}

	voices, err := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated or ranges e.g. 1-3, empty for all)")
	if err != nil {
		return err
	}

	csvFile, err := prompt.PromptString(reader, "Enter CSV output filename (optional, press Enter for default)", "")
	if err != nil {
		return err
	}

	res, err := ProcessCMatrix(ctx, database, piece, voices, version, csvFile)
	if err != nil {
		return err
	}

	fmt.Print(FormatVoiceDifferences(res.Differences))
	fmt.Printf("\nGenerated interval matrix with %d columns (offsets) and saved to %s\n",
		len(res.UniqueToffsets), res.CSVFilename)

	return nil
}
