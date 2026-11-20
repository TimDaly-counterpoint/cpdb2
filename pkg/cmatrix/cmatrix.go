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
	Voices         []int
	VoiceNotes     map[int][]MelodicNote
	Differences    []VoiceDifference
	Matrix         [][]string
	CSVFilename    string
	UniqueToffsets []float64
}

// CalculateVoiceIntervals converts a sequence of note entries for a single voice into melodic intervals.
func CalculateVoiceIntervals(notes []db.SequenceNote) []MelodicNote {
	var melodicNotes []MelodicNote
	var firstPitch string
	hasSoundingNote := false

	barStartOffsets := make(map[int]float64)
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

// ProcessCMatrix runs the interval extraction, difference checking, and CSV generation for a piece and voices.
func ProcessCMatrix(ctx context.Context, database *db.DB, piece int, voices []int, customCSVFile string) (*Result, error) {
	if len(voices) == 0 {
		// Discover voices for this piece
		rows, err := database.Pool.Query(ctx, "SELECT DISTINCT voice FROM notes WHERE piece = $1 ORDER BY voice", piece)
		if err != nil {
			return nil, fmt.Errorf("query voices: %w", err)
		}
		for rows.Next() {
			var v int
			if err := rows.Scan(&v); err == nil {
				voices = append(voices, v)
			}
		}
		rows.Close()
	}

	if len(voices) == 0 {
		return nil, fmt.Errorf("at least one voice must be specified for piece %d", piece)
	}

	voiceOrder := slices.Clone(voices)
	slices.Sort(voiceOrder)
	voiceOrder = slices.Compact(voiceOrder)

	rawNotes, err := database.FetchNotesForVoices(ctx, piece, voiceOrder)
	if err != nil {
		return nil, fmt.Errorf("fetch notes: %w", err)
	}

	voiceMelodicNotes := make(map[int][]MelodicNote)
	for _, v := range voiceOrder {
		voiceMelodicNotes[v] = CalculateVoiceIntervals(rawNotes[v])
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

	voices, err := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated or ranges e.g. 1-3, empty for all)")
	if err != nil {
		return err
	}

	csvFile, err := prompt.PromptString(reader, "Enter CSV output filename (optional, press Enter for default)", "")
	if err != nil {
		return err
	}

	res, err := ProcessCMatrix(ctx, database, piece, voices, csvFile)
	if err != nil {
		return err
	}

	fmt.Print(FormatVoiceDifferences(res.Differences))
	fmt.Printf("\nGenerated interval matrix with %d columns (offsets) and saved to %s\n",
		len(res.UniqueToffsets), res.CSVFilename)

	return nil
}
