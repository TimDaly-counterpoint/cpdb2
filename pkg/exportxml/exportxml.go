package exportxml

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"cpdb2/pkg/db"
	"cpdb2/pkg/patterns"
	"cpdb2/pkg/prompt"
)

// ParsedSequence represents a sequence pattern along with its occurrences.
type ParsedSequence struct {
	Sequence    []int
	Length      int
	Count       int
	Occurrences []patterns.SequenceOccurrence
}

// ParseSequences parses sequence blocks from an io.Reader.
func ParseSequences(r io.Reader) ([]ParsedSequence, error) {
	scanner := bufio.NewScanner(r)
	var sequences []ParsedSequence
	var current *ParsedSequence

	seqHeaderRegex := regexp.MustCompile(`(?i)^\s*Sequence:\s*\[(.*?)\]`)
	lengthRegex := regexp.MustCompile(`(?i)^\s*Length:\s*(\d+)`)
	countRegex := regexp.MustCompile(`(?i)^\s*Count:\s*(\d+)`)
	occRegex := regexp.MustCompile(`(?i)^\s*Piece:\s*(\d+),\s*Voice:\s*(\d+),\s*List:\s*(\w+),\s*Bar:\s*(\d+),\s*Index:\s*(\d+),\s*Pitch:\s*([^,]+),\s*Offset:\s*([0-9.]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "---") {
			if current != nil && len(current.Occurrences) > 0 {
				sequences = append(sequences, *current)
				current = nil
			}
			continue
		}

		if match := seqHeaderRegex.FindStringSubmatch(line); match != nil {
			if current != nil && len(current.Occurrences) > 0 {
				sequences = append(sequences, *current)
			}
			current = &ParsedSequence{}
			rawItems := strings.Fields(match[1])
			for _, item := range rawItems {
				item = strings.Trim(item, ",")
				if item == "" {
					continue
				}
				v, err := strconv.Atoi(item)
				if err == nil {
					current.Sequence = append(current.Sequence, v)
				}
			}
			current.Length = len(current.Sequence)
			continue
		}

		if current == nil {
			continue
		}

		if match := lengthRegex.FindStringSubmatch(line); match != nil {
			if v, err := strconv.Atoi(match[1]); err == nil {
				current.Length = v
			}
			continue
		}

		if match := countRegex.FindStringSubmatch(line); match != nil {
			if v, err := strconv.Atoi(match[1]); err == nil {
				current.Count = v
			}
			continue
		}

		if match := occRegex.FindStringSubmatch(line); match != nil {
			piece, _ := strconv.Atoi(match[1])
			voice, _ := strconv.Atoi(match[2])
			listType := match[3]
			bar, _ := strconv.Atoi(match[4])
			index, _ := strconv.Atoi(match[5])
			pitch := strings.TrimSpace(match[6])
			offset, _ := strconv.ParseFloat(match[7], 64)

			current.Occurrences = append(current.Occurrences, patterns.SequenceOccurrence{
				Piece:    piece,
				Voice:    voice,
				ListType: listType,
				Bar:      bar,
				Index:    index,
				Pitch:    pitch,
				Toffset:  offset,
			})
			continue
		}
	}

	if current != nil && len(current.Occurrences) > 0 {
		sequences = append(sequences, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read sequences: %w", err)
	}

	return sequences, nil
}

// ParseSequencesString parses sequence blocks from a string.
func ParseSequencesString(s string) ([]ParsedSequence, error) {
	return ParseSequences(strings.NewReader(s))
}

func extractOccurrenceNotes(allNotes []db.SequenceNote, mapping []int, intervSeq []int, listType string, sequence []int, startIndex int) []db.SequenceNote {
	seqLen := len(sequence)
	listPos := -1

	for i, idx := range mapping {
		if idx < len(allNotes) && allNotes[idx].Index == startIndex {
			if i+seqLen <= len(intervSeq) {
				match := true
				for k := 0; k < seqLen; k++ {
					if intervSeq[i+k] != sequence[k] {
						match = false
						break
					}
				}
				if match {
					listPos = i
					break
				}
			}
			if listPos == -1 {
				listPos = i
			}
		}
	}

	if listPos == -1 {
		return nil
	}

	var occNotes []db.SequenceNote
	for k := 0; k < seqLen; k++ {
		if listPos+k < len(mapping) {
			p := mapping[listPos+k]
			if sequence[k] == 0 {
				for p >= 0 && p < len(allNotes) && (allNotes[p].Pitch == "0" || allNotes[p].Midi == 0) {
					occNotes = append(occNotes, allNotes[p])
					p++
				}
			} else {
				if p >= 0 && p < len(allNotes) {
					occNotes = append(occNotes, allNotes[p])
				}
			}
		}
	}

	if seqLen > 0 && sequence[seqLen-1] != 0 {
		if listPos+seqLen < len(mapping) {
			pEnd := mapping[listPos+seqLen]
			if pEnd >= 0 && pEnd < len(allNotes) {
				occNotes = append(occNotes, allNotes[pEnd])
			}
		} else if len(occNotes) > 0 {
			lastP := mapping[listPos+seqLen-1]
			if listType == "retroList" || listType == "retroInvertList" {
				if lastP-1 >= 0 && lastP-1 < len(allNotes) {
					occNotes = append(occNotes, allNotes[lastP-1])
				}
			} else {
				if lastP+1 >= 0 && lastP+1 < len(allNotes) {
					occNotes = append(occNotes, allNotes[lastP+1])
				}
			}
		}
	}

	return occNotes
}

func sanitizePieceTitle(title string) string {
	var sb strings.Builder
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	s := sb.String()
	if len(s) > 10 {
		s = s[:10]
	}
	return s
}

func mapListTypeName(listType string) string {
	switch listType {
	case "intervList":
		return ""
	case "invertList":
		return "Inv"
	case "retroList":
		return "retro"
	case "retroInvertList":
		return "rI"
	default:
		return listType
	}
}

type noteDurationType struct {
	name   string
	dotted bool
}

func getNoteTypeAndDot(dur int) (string, bool) {
	durationMap := map[int]noteDurationType{
		384: {"long", false},
		288: {"breve", true},
		192: {"breve", false},
		144: {"whole", true},
		96:  {"whole", false},
		72:  {"half", true},
		64:  {"whole", false},
		48:  {"half", false},
		36:  {"quarter", true},
		32:  {"half", false},
		24:  {"quarter", false},
		18:  {"eighth", true},
		16:  {"quarter", false},
		12:  {"eighth", false},
		8:   {"eighth", false},
		6:   {"16th", false},
		4:   {"16th", false},
		3:   {"32nd", false},
		2:   {"32nd", false},
		1:   {"64th", false},
	}

	if dt, ok := durationMap[dur]; ok {
		return dt.name, dt.dotted
	}

	if dur >= 96 {
		return "whole", false
	} else if dur >= 48 {
		return "half", false
	} else if dur >= 24 {
		return "quarter", false
	} else if dur >= 12 {
		return "eighth", false
	} else if dur >= 6 {
		return "16th", false
	}
	return "32nd", false
}

func parsePitchComponents(pitchStr string, midi int) (step string, octave int, alter int) {
	pitchRegex := regexp.MustCompile(`^([A-G])(b{1,2}|#{1,2}|x)?(-?[0-9]+)$`)
	m := pitchRegex.FindStringSubmatch(strings.TrimSpace(pitchStr))
	if m != nil {
		step = m[1]
		acc := m[2]
		oct, _ := strconv.Atoi(m[3])
		octave = oct
		switch acc {
		case "b":
			alter = -1
		case "bb":
			alter = -2
		case "#":
			alter = 1
		case "##", "x":
			alter = 2
		}
		return step, octave, alter
	}

	if midi > 0 {
		stepNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
		semi := midi % 12
		octave = (midi / 12) - 1
		name := stepNames[semi]
		if len(name) == 1 {
			step = name
		} else {
			step = string(name[0])
			alter = 1
		}
		return step, octave, alter
	}

	return "C", 4, 0
}

// GenerateMusicXML creates a MusicXML 4.0 string for the given sequence and extracted notes.
func GenerateMusicXML(workTitle string, occs []patterns.SequenceOccurrence, occNotesList [][]db.SequenceNote) string {
	const divisions = 24
	const measureCapacity = 12 * divisions

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")
	sb.WriteString("<!DOCTYPE score-partwise PUBLIC \"-//Recordare//DTD MusicXML 4.0 Partwise//EN\" \"http://www.musicxml.org/dtds/partwise.dtd\">\n\n")
	sb.WriteString("<score-partwise version=\"4.0\">\n")
	sb.WriteString("  <work>\n")
	fmt.Fprintf(&sb, "    <work-title>%s</work-title>\n", workTitle)
	sb.WriteString("  </work>\n")
	sb.WriteString("  <part-list>\n")

	for j, occ := range occs {
		partID := fmt.Sprintf("P%d", j+1)
		mappedList := mapListTypeName(occ.ListType)
		var labelParts []string
		labelParts = append(labelParts, fmt.Sprintf("Occ %d", j+1))
		labelParts = append(labelParts, fmt.Sprintf("P%d", occ.Piece))
		labelParts = append(labelParts, fmt.Sprintf("V%d", occ.Voice))
		labelParts = append(labelParts, fmt.Sprintf("Idx %d", occ.Index))
		if mappedList != "" {
			labelParts = append(labelParts, mappedList)
		}
		partName := "(" + strings.Join(labelParts, " ") + ")"
		fmt.Fprintf(&sb, "    <score-part id=\"%s\">\n", partID)
		fmt.Fprintf(&sb, "      <part-name>%s</part-name>\n", partName)
		sb.WriteString("    </score-part>\n")
	}
	sb.WriteString("  </part-list>\n")

	partTotalDurations := make([]int, len(occs))
	for j, notes := range occNotesList {
		leadingRest := 0
		if len(notes) > 0 {
			leadingRest = int(math.Mod(notes[0].Toffset, 96))
		}
		dur := leadingRest
		for _, n := range notes {
			dur += int(n.Duration)
		}
		partTotalDurations[j] = dur
	}
	maxTotalDuration := 0
	for _, d := range partTotalDurations {
		if d > maxTotalDuration {
			maxTotalDuration = d
		}
	}

	for j, occ := range occs {
		partID := fmt.Sprintf("P%d", j+1)
		notes := occNotesList[j]
		fmt.Fprintf(&sb, "  <part id=\"%s\">\n", partID)

		currentMeasureNum := 1
		currentMeasurePos := 0

		writeMeasureStart := func(mNum int) {
			fmt.Fprintf(&sb, "    <measure number=\"%d\">\n", mNum)
			if mNum == 1 {
				sb.WriteString("      <attributes>\n")
				fmt.Fprintf(&sb, "        <divisions>%d</divisions>\n", divisions)
				sb.WriteString("        <time>\n")
				sb.WriteString("          <beats>12</beats>\n")
				sb.WriteString("          <beat-type>4</beat-type>\n")
				sb.WriteString("        </time>\n")
				sb.WriteString("        <clef>\n")
				sb.WriteString("          <sign>G</sign>\n")
				sb.WriteString("          <line>2</line>\n")
				sb.WriteString("        </clef>\n")
				sb.WriteString("      </attributes>\n")
			}
		}

		writeMeasureEnd := func() {
			sb.WriteString("    </measure>\n")
		}

		writeNote := func(n *db.SequenceNote, dur int, isTieStart, isTieStop bool) {
			isRest := (n == nil || n.Pitch == "0" || n.Midi == 0)
			noteType, isDotted := getNoteTypeAndDot(dur)

			sb.WriteString("      <note>\n")
			if isRest {
				sb.WriteString("        <rest/>\n")
			} else {
				step, octave, alter := parsePitchComponents(n.Pitch, n.Midi)
				sb.WriteString("        <pitch>\n")
				fmt.Fprintf(&sb, "          <step>%s</step>\n", step)
				if alter != 0 {
					fmt.Fprintf(&sb, "          <alter>%d</alter>\n", alter)
				}
				fmt.Fprintf(&sb, "          <octave>%d</octave>\n", octave)
				sb.WriteString("        </pitch>\n")
			}

			fmt.Fprintf(&sb, "        <duration>%d</duration>\n", dur)

			if !isRest {
				if isTieStop {
					sb.WriteString("        <tie type=\"stop\"/>\n")
					sb.WriteString("        <tied type=\"stop\"/>\n")
				}
				if isTieStart {
					sb.WriteString("        <tie type=\"start\"/>\n")
					sb.WriteString("        <tied type=\"start\"/>\n")
				}
			}

			if noteType != "" {
				fmt.Fprintf(&sb, "        <type>%s</type>\n", noteType)
			}
			if isDotted {
				sb.WriteString("        <dot/>\n")
			}
			sb.WriteString("      </note>\n")
		}

		writeMeasureStart(currentMeasureNum)

		// Leading rest
		if len(notes) > 0 {
			leadingRest := int(math.Mod(notes[0].Toffset, 96))
			remRest := leadingRest
			for remRest > 0 {
				durToAdd := remRest
				space := measureCapacity - currentMeasurePos
				if durToAdd > space {
					durToAdd = space
				}
				writeNote(nil, durToAdd, false, false)
				remRest -= durToAdd
				currentMeasurePos += durToAdd
				if currentMeasurePos >= measureCapacity {
					writeMeasureEnd()
					currentMeasureNum++
					writeMeasureStart(currentMeasureNum)
					currentMeasurePos = 0
				}
			}
		}

		// Notes
		for i := range notes {
			n := &notes[i]
			totalDur := int(n.Duration)
			if totalDur <= 0 {
				continue
			}

			space := measureCapacity - currentMeasurePos
			if totalDur <= space {
				writeNote(n, totalDur, false, false)
				currentMeasurePos += totalDur
				if currentMeasurePos >= measureCapacity {
					writeMeasureEnd()
					currentMeasureNum++
					writeMeasureStart(currentMeasureNum)
					currentMeasurePos = 0
				}
			} else {
				firstDur := space
				secondDur := totalDur - space

				writeNote(n, firstDur, true, false)
				writeMeasureEnd()

				currentMeasureNum++
				writeMeasureStart(currentMeasureNum)
				currentMeasurePos = 0

				remTail := secondDur
				for remTail > 0 {
					durChunk := remTail
					spaceChunk := measureCapacity - currentMeasurePos
					if durChunk > spaceChunk {
						durChunk = spaceChunk
					}
					isMore := (remTail-durChunk > 0)
					writeNote(n, durChunk, isMore, true)
					remTail -= durChunk
					currentMeasurePos += durChunk
					if currentMeasurePos >= measureCapacity {
						writeMeasureEnd()
						currentMeasureNum++
						writeMeasureStart(currentMeasureNum)
						currentMeasurePos = 0
					}
				}
			}
		}

		// Padding rests
		paddingNeeded := maxTotalDuration - partTotalDurations[j]
		for paddingNeeded > 0 {
			durToAdd := paddingNeeded
			space := measureCapacity - currentMeasurePos
			if durToAdd > space {
				durToAdd = space
			}
			writeNote(nil, durToAdd, false, false)
			paddingNeeded -= durToAdd
			currentMeasurePos += durToAdd
			if currentMeasurePos >= measureCapacity {
				writeMeasureEnd()
				currentMeasureNum++
				writeMeasureStart(currentMeasureNum)
				currentMeasurePos = 0
			}
		}

		writeMeasureEnd()
		sb.WriteString("  </part>\n")
		_ = occ
	}

	sb.WriteString("</score-partwise>\n")
	return sb.String()
}

// ExportSequencesToMusicXML exports a list of parsed sequences to MusicXML 4.0 files.
func ExportSequencesToMusicXML(ctx context.Context, database *db.DB, sequences []ParsedSequence, outDir string) ([]string, error) {
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	var generatedFiles []string

	for seqIdx, seq := range sequences {
		if len(seq.Occurrences) == 0 {
			continue
		}

		firstOcc := seq.Occurrences[0]
		var pieceTitle string
		_ = database.Pool.QueryRow(ctx, "SELECT title FROM pieces WHERE seq = $1", firstOcc.Piece).Scan(&pieceTitle)
		shortTitle := sanitizePieceTitle(pieceTitle)
		if shortTitle == "" {
			shortTitle = fmt.Sprintf("Piece%d", firstOcc.Piece)
		}

		mappedList := mapListTypeName(firstOcc.ListType)
		workTitle := fmt.Sprintf("%s_V%d_%s_%d", shortTitle, firstOcc.Voice, mappedList, seqIdx+1)
		fileName := fmt.Sprintf("%s.musicxml", workTitle)
		outPath := filepath.Join(outDir, fileName)

		var occNotesList [][]db.SequenceNote
		for _, occ := range seq.Occurrences {
			allNotes, err := database.FetchNotesForPieceVoice(ctx, occ.Piece, occ.Voice)
			if err != nil {
				return nil, err
			}

			seqObj, err := patterns.NewIntervalSequence(ctx, database, occ.Piece, occ.Voice, false)
			if err != nil {
				return nil, err
			}

			var mapping []int
			var intervSeq []int
			switch occ.ListType {
			case "intervList":
				mapping = seqObj.IntervMapping
				intervSeq = seqObj.IntervList
			case "invertList":
				mapping = seqObj.InvertMapping
				intervSeq = seqObj.InvertList
			case "retroList":
				mapping = seqObj.RetroMapping
				intervSeq = seqObj.RetroList
			case "retroInvertList":
				mapping = seqObj.RetroInvertMapping
				intervSeq = seqObj.RetroInvertList
			default:
				mapping = seqObj.IntervMapping
				intervSeq = seqObj.IntervList
			}

			notes := extractOccurrenceNotes(allNotes, mapping, intervSeq, occ.ListType, seq.Sequence, occ.Index)
			occNotesList = append(occNotesList, notes)
		}

		xmlContent := GenerateMusicXML(workTitle, seq.Occurrences, occNotesList)
		if err := os.WriteFile(outPath, []byte(xmlContent), 0644); err != nil {
			return nil, fmt.Errorf("write file %s: %w", outPath, err)
		}
		generatedFiles = append(generatedFiles, outPath)
	}

	return generatedFiles, nil
}

// ExportSequencesFromFile reads sequences from a file and exports MusicXML files.
func ExportSequencesFromFile(ctx context.Context, database *db.DB, filePath, outDir string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", filePath, err)
	}
	defer file.Close()

	sequences, err := ParseSequences(file)
	if err != nil {
		return nil, fmt.Errorf("parse sequences from %s: %w", filePath, err)
	}

	if len(sequences) == 0 {
		return nil, fmt.Errorf("no sequences found in %s", filePath)
	}

	return ExportSequencesToMusicXML(ctx, database, sequences, outDir)
}

// RunInteractive prompts the user and exports sequences from file to MusicXML.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader, defaultSeqFile string) error {
	if defaultSeqFile == "" {
		defaultSeqFile = "patterns.txt"
	}

	filePath, err := prompt.PromptString(reader, "Enter relative or absolute filepath to sequence file", defaultSeqFile)
	if err != nil {
		return err
	}

	outDir, err := prompt.PromptString(reader, "Enter output directory", "xmlseq")
	if err != nil {
		return err
	}

	files, err := ExportSequencesFromFile(ctx, database, filePath, outDir)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully exported %d MusicXML files to %s\n", len(files), outDir)
	return nil
}

// ConsonantSonority represents a single sonority row from the consonant table.
type ConsonantSonority struct {
	Offset     int
	Duration   int
	Pitches    []string
	MidiValues []int
	Voices     []int
}

func writePartMeasures(sb *strings.Builder, notes []db.SequenceNote) {
	const divisions = 24
	const measureCapacity = 12 * divisions

	currentMeasureNum := 1
	currentMeasurePos := 0
	measureOpened := false

	writeMeasureStart := func(mNum int) {
		fmt.Fprintf(sb, "    <measure number=\"%d\">\n", mNum)
		if mNum == 1 {
			sb.WriteString("      <attributes>\n")
			fmt.Fprintf(sb, "        <divisions>%d</divisions>\n", divisions)
			sb.WriteString("        <time>\n")
			sb.WriteString("          <beats>12</beats>\n")
			sb.WriteString("          <beat-type>4</beat-type>\n")
			sb.WriteString("        </time>\n")
			sb.WriteString("        <clef>\n")
			sb.WriteString("          <sign>G</sign>\n")
			sb.WriteString("          <line>2</line>\n")
			sb.WriteString("        </clef>\n")
			sb.WriteString("      </attributes>\n")
		}
		measureOpened = true
	}

	writeMeasureEnd := func() {
		if measureOpened {
			sb.WriteString("    </measure>\n")
			measureOpened = false
		}
	}

	writeNote := func(n *db.SequenceNote, dur int, isTieStart, isTieStop bool) {
		if !measureOpened {
			writeMeasureStart(currentMeasureNum)
		}
		isRest := (n == nil || n.Pitch == "0" || n.Pitch == "" || n.Midi == 0)
		noteType, isDotted := getNoteTypeAndDot(dur)

		sb.WriteString("      <note>\n")
		if isRest {
			sb.WriteString("        <rest/>\n")
		} else {
			step, octave, alter := parsePitchComponents(n.Pitch, n.Midi)
			sb.WriteString("        <pitch>\n")
			fmt.Fprintf(sb, "          <step>%s</step>\n", step)
			if alter != 0 {
				fmt.Fprintf(sb, "          <alter>%d</alter>\n", alter)
			}
			fmt.Fprintf(sb, "          <octave>%d</octave>\n", octave)
			sb.WriteString("        </pitch>\n")
		}

		fmt.Fprintf(sb, "        <duration>%d</duration>\n", dur)

		if !isRest {
			if isTieStop {
				sb.WriteString("        <tie type=\"stop\"/>\n")
				sb.WriteString("        <tied type=\"stop\"/>\n")
			}
			if isTieStart {
				sb.WriteString("        <tie type=\"start\"/>\n")
				sb.WriteString("        <tied type=\"start\"/>\n")
			}
		}

		if noteType != "" {
			fmt.Fprintf(sb, "        <type>%s</type>\n", noteType)
		}
		if isDotted {
			sb.WriteString("        <dot/>\n")
		}
		sb.WriteString("      </note>\n")
	}

	writeMeasureStart(currentMeasureNum)

	for i := range notes {
		n := &notes[i]
		totalDur := int(n.Duration)
		if totalDur <= 0 {
			continue
		}

		space := measureCapacity - currentMeasurePos
		if totalDur <= space {
			writeNote(n, totalDur, false, false)
			currentMeasurePos += totalDur
			if currentMeasurePos >= measureCapacity {
				writeMeasureEnd()
				currentMeasureNum++
				currentMeasurePos = 0
			}
		} else {
			firstDur := space
			secondDur := totalDur - space

			writeNote(n, firstDur, true, false)
			writeMeasureEnd()

			currentMeasureNum++
			currentMeasurePos = 0

			remTail := secondDur
			for remTail > 0 {
				durChunk := remTail
				spaceChunk := measureCapacity - currentMeasurePos
				if durChunk > spaceChunk {
					durChunk = spaceChunk
				}
				isMore := (remTail-durChunk > 0)
				writeNote(n, durChunk, isMore, true)
				remTail -= durChunk
				currentMeasurePos += durChunk
				if currentMeasurePos >= measureCapacity {
					writeMeasureEnd()
					currentMeasureNum++
					currentMeasurePos = 0
				}
			}
		}
	}

	writeMeasureEnd()
}

// GenerateConsonantMusicXML creates a multi-voice MusicXML 4.0 score from consonant sonorities.
func GenerateConsonantMusicXML(pieceTitle string, pieceSeq int, sonorities []ConsonantSonority) (string, error) {
	if len(sonorities) == 0 {
		return "", fmt.Errorf("no consonant sonorities provided for piece %d", pieceSeq)
	}

	// Sort sonorities by offset ascending
	sortedSonorities := make([]ConsonantSonority, len(sonorities))
	copy(sortedSonorities, sonorities)
	slices.SortFunc(sortedSonorities, func(a, b ConsonantSonority) int {
		if a.Offset != b.Offset {
			return a.Offset - b.Offset
		}
		return a.Duration - b.Duration
	})

	// Collect unique voices and calculate total duration
	voiceSet := make(map[int]bool)
	totalDuration := 0
	for _, s := range sortedSonorities {
		for _, v := range s.Voices {
			voiceSet[v] = true
		}
		if end := s.Offset + s.Duration; end > totalDuration {
			totalDuration = end
		}
	}

	var uniqueVoices []int
	for v := range voiceSet {
		uniqueVoices = append(uniqueVoices, v)
	}
	sort.Ints(uniqueVoices)
	if len(uniqueVoices) == 0 {
		return "", fmt.Errorf("no voices found in consonant sonorities for piece %d", pieceSeq)
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")
	sb.WriteString("<!DOCTYPE score-partwise PUBLIC \"-//Recordare//DTD MusicXML 4.0 Partwise//EN\" \"http://www.musicxml.org/dtds/partwise.dtd\">\n\n")
	sb.WriteString("<score-partwise version=\"4.0\">\n")
	sb.WriteString("  <work>\n")
	if strings.TrimSpace(pieceTitle) == "" {
		pieceTitle = fmt.Sprintf("Piece %d", pieceSeq)
	}
	fmt.Fprintf(&sb, "    <work-title>%s</work-title>\n", pieceTitle)
	sb.WriteString("  </work>\n")
	sb.WriteString("  <part-list>\n")

	for _, v := range uniqueVoices {
		partID := fmt.Sprintf("P%d", v)
		partName := fmt.Sprintf("Voice %d", v)
		fmt.Fprintf(&sb, "    <score-part id=\"%s\">\n", partID)
		fmt.Fprintf(&sb, "      <part-name>%s</part-name>\n", partName)
		sb.WriteString("    </score-part>\n")
	}
	sb.WriteString("  </part-list>\n")

	for _, v := range uniqueVoices {
		partID := fmt.Sprintf("P%d", v)
		fmt.Fprintf(&sb, "  <part id=\"%s\">\n", partID)

		// Build note stream for voice v
		var voiceNotes []db.SequenceNote
		currOffset := 0

		for _, s := range sortedSonorities {
			if s.Offset > currOffset {
				gapDur := s.Offset - currOffset
				voiceNotes = append(voiceNotes, db.SequenceNote{
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
				if vID == v {
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
				voiceNotes = append(voiceNotes, db.SequenceNote{
					Pitch:    pitch,
					Duration: float64(s.Duration),
					Midi:     midi,
					Toffset:  float64(currOffset),
				})
				currOffset += s.Duration
			}
		}

		if currOffset < totalDuration {
			voiceNotes = append(voiceNotes, db.SequenceNote{
				Pitch:    "0",
				Duration: float64(totalDuration - currOffset),
				Midi:     0,
				Toffset:  float64(currOffset),
			})
		}

		writePartMeasures(&sb, voiceNotes)
		sb.WriteString("  </part>\n")
	}

	sb.WriteString("</score-partwise>\n")
	return sb.String(), nil
}

// FetchConsonantSonorities queries the consonant table for a piece and returns sonorities sorted by offset.
func FetchConsonantSonorities(ctx context.Context, database *db.DB, pieceSeq int) ([]ConsonantSonority, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT "offset", duration, pitches, midi_values, voices
		FROM consonant
		WHERE piece = $1
		ORDER BY "offset"
	`, pieceSeq)
	if err != nil {
		return nil, fmt.Errorf("query consonant table for piece %d: %w", pieceSeq, err)
	}
	defer rows.Close()

	var sonorities []ConsonantSonority
	for rows.Next() {
		var s ConsonantSonority
		if err := rows.Scan(&s.Offset, &s.Duration, &s.Pitches, &s.MidiValues, &s.Voices); err != nil {
			return nil, fmt.Errorf("scan consonant row: %w", err)
		}
		sonorities = append(sonorities, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate consonant rows: %w", err)
	}

	return sonorities, nil
}

// ExportConsonantToMusicXML queries the consonant table for a piece and exports it to a MusicXML file.
func ExportConsonantToMusicXML(ctx context.Context, database *db.DB, pieceSeq int, outFile string) error {
	sonorities, err := FetchConsonantSonorities(ctx, database, pieceSeq)
	if err != nil {
		return err
	}
	if len(sonorities) == 0 {
		return fmt.Errorf("no sonorities found in consonant table for piece %d", pieceSeq)
	}

	var pieceTitle string
	_ = database.Pool.QueryRow(ctx, "SELECT title FROM pieces WHERE seq = $1", pieceSeq).Scan(&pieceTitle)
	if strings.TrimSpace(pieceTitle) == "" {
		pieceTitle = fmt.Sprintf("Piece %d", pieceSeq)
	}

	xmlContent, err := GenerateConsonantMusicXML(pieceTitle, pieceSeq, sonorities)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(outFile, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("write musicxml file %s: %w", outFile, err)
	}

	return nil
}

// RunInteractiveSmooth prompts the user for piece.seq and output filename and exports consonant sonorities to MusicXML.
func RunInteractiveSmooth(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	pieceSeq, err := prompt.PromptInt(reader, "Enter piece ID (piece.seq)", 1)
	if err != nil {
		return err
	}

	var pieceTitle string
	_ = database.Pool.QueryRow(ctx, "SELECT title FROM pieces WHERE seq = $1", pieceSeq).Scan(&pieceTitle)
	cleanTitle := sanitizePieceTitle(pieceTitle)
	if cleanTitle == "" {
		cleanTitle = fmt.Sprintf("Piece_%d", pieceSeq)
	}
	defaultOutFile := fmt.Sprintf("%s_smooth.musicxml", cleanTitle)

	outFile, err := prompt.PromptString(reader, "Enter output MusicXML filename", defaultOutFile)
	if err != nil {
		return err
	}

	if err := ExportConsonantToMusicXML(ctx, database, pieceSeq, outFile); err != nil {
		return err
	}

	fmt.Printf("Successfully exported smoothed MusicXML for piece %d to %s\n", pieceSeq, outFile)
	return nil
}
