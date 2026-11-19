package sonority

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"

	"cpdb2/pkg/db"
	"cpdb2/pkg/music"
	"cpdb2/pkg/prompt"
)

// SonorityData represents overlapping sonority interval between two voices.
type SonorityData struct {
	StartOffset float64
	EndOffset   float64
	Duration    float64
	Sonority    int
	Bar         int
}

// CalculateSonorities calculates the vertical interval / sonority between two voice note slices.
func CalculateSonorities(v1Notes, v2Notes []db.SequenceNote) []SonorityData {
	var results []SonorityData
	i1, i2 := 0, 0

	for i1 < len(v1Notes) && i2 < len(v2Notes) {
		n1 := v1Notes[i1]
		n2 := v2Notes[i2]

		start := max(n1.Toffset, n2.Toffset)
		end := min(n1.Toffset+n1.Duration, n2.Toffset+n2.Duration)

		if end > start {
			if n1.Midi != 0 && n2.Midi != 0 {
				rawSonority := int(math.Abs(float64(n1.Midi - n2.Midi)))
				results = append(results, SonorityData{
					StartOffset: start,
					EndOffset:   end,
					Duration:    end - start,
					Sonority:    music.Sonor[rawSonority],
					Bar:         n1.Bar,
				})
			}
		}

		if n1.Toffset+n1.Duration < n2.Toffset+n2.Duration {
			i1++
		} else if n2.Toffset+n2.Duration < n1.Toffset+n1.Duration {
			i2++
		} else {
			i1++
			i2++
		}
	}
	return results
}

// PrintAndSaveSonorities prints sonority data and writes to CSV.
func PrintAndSaveSonorities(data []SonorityData, csvFile string) error {
	if csvFile == "" {
		csvFile = "son_data.csv"
	}

	file, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("create csv file %s: %w", csvFile, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Starting offset", "Ending offset", "Duration", "Sonority", "Bar"})
	fmt.Printf("%-15s %-15s %-10s %-10s %-10s\n", "Start Offset", "End Offset", "Duration", "Sonority", "Bar")

	totalDuration := 0.0
	sonorityTotals := make(map[int]float64)

	for _, d := range data {
		row := []string{
			strconv.FormatFloat(d.StartOffset, 'f', -1, 64),
			strconv.FormatFloat(d.EndOffset, 'f', -1, 64),
			strconv.FormatFloat(d.Duration, 'f', -1, 64),
			strconv.Itoa(d.Sonority),
			strconv.Itoa(d.Bar),
		}
		writer.Write(row)
		fmt.Printf("%-15.5f %-15.5f %-10.5f %-10d %-10d\n", d.StartOffset, d.EndOffset, d.Duration, d.Sonority, d.Bar)

		totalDuration += d.Duration
		sonorityTotals[d.Sonority] += d.Duration
	}

	writer.Write([]string{})
	writer.Write([]string{"Sonority", "Total Duration", "Percentage"})
	fmt.Printf("\n%-10s %-15s %-10s\n", "Sonority", "Total Duration", "Percentage")

	sortedSonorities := slices.Sorted(maps.Keys(sonorityTotals))
	for _, s := range sortedSonorities {
		dur := sonorityTotals[s]
		var percentage float64
		if totalDuration > 0 {
			percentage = (dur / totalDuration) * 100.0
		}
		row := []string{
			strconv.Itoa(s),
			strconv.FormatFloat(dur, 'f', -1, 64),
			strconv.FormatFloat(percentage, 'f', 5, 64),
		}
		writer.Write(row)
		fmt.Printf("%-10d %-15.5f %-10.5f\n", s, dur, percentage)
	}

	writer.Write([]string{"TOTAL", strconv.FormatFloat(totalDuration, 'f', -1, 64), "100.00000"})
	fmt.Printf("%-10s %-15.5f %-10.5f\n", "TOTAL", totalDuration, 100.0)

	return nil
}

// ProcessSonority fetches notes for piece and specified voices, calculates sonorities, and saves output.
func ProcessSonority(ctx context.Context, database *db.DB, pieceSeq int, voices []int, csvFile string) ([]SonorityData, error) {
	if len(voices) < 2 {
		voicesCount, err := database.GetPieceVoices(ctx, pieceSeq)
		if err != nil {
			return nil, fmt.Errorf("error getting voice count for piece %d: %w", pieceSeq, err)
		}

		if voicesCount == 3 {
			voices = []int{1, 3}
		} else {
			voices = []int{1, 2}
		}
	}

	voiceNotes, err := database.FetchNotesForVoices(ctx, pieceSeq, voices)
	if err != nil {
		return nil, fmt.Errorf("error fetching notes: %w", err)
	}

	results := CalculateSonorities(voiceNotes[voices[0]], voiceNotes[voices[1]])

	if err := PrintAndSaveSonorities(results, csvFile); err != nil {
		return nil, fmt.Errorf("error saving results: %w", err)
	}

	return results, nil
}

// RunInteractive prompts the user and executes sonority processing.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	piece, err := prompt.PromptInt(reader, "Enter piece identity", 1)
	if err != nil {
		return err
	}

	voices, err := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated, empty for auto 1,2 or 1,3)")
	if err != nil {
		return err
	}

	csvFile, err := prompt.PromptString(reader, "Enter CSV output filename", "son_data.csv")
	if err != nil {
		return err
	}

	_, err = ProcessSonority(ctx, database, piece, voices, csvFile)
	return err
}
