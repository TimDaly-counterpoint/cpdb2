package patterns

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cpdb2/pkg/db"
	"cpdb2/pkg/music"
)

// NoteRow mirrors note data for interval analysis.
type NoteRow struct {
	Midi    int
	Bar     int
	Index   int
	Pitch   string
	Toffset float64
}

// IntervalSequence holds the four interval representations for a single
// (piece, voice) pair, plus the mappings back to the original notes.
type IntervalSequence struct {
	Piece int
	Voice int

	NotesData []NoteRow

	IntervList         []int
	InvertList         []int
	RetroList          []int
	RetroInvertList    []int
	IntervMapping      []int
	InvertMapping      []int
	RetroMapping       []int
	RetroInvertMapping []int
}

// PieceIntervalSequences groups all voices for a piece.
type PieceIntervalSequences struct {
	Piece             int
	IntervalSequences []*IntervalSequence
}

// QueryBlock is the top-level collection of pieces to analyse.
type QueryBlock struct {
	Blocks  []*PieceIntervalSequences
	UseMidi bool
}

// NewIntervalSequence loads the notes for (piece, voice) from the database
// and builds the four interval lists.
func NewIntervalSequence(ctx context.Context, database *db.DB, piece, voice int, useMidi bool) (*IntervalSequence, error) {
	rows, err := database.Pool.Query(ctx,
		`SELECT midi, bar, index, pitch, toffset FROM notes WHERE piece = $1 AND voice = $2 ORDER BY index`,
		piece, voice)
	if err != nil {
		return nil, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var notes []NoteRow
	for rows.Next() {
		var n NoteRow
		if err := rows.Scan(&n.Midi, &n.Bar, &n.Index, &n.Pitch, &n.Toffset); err != nil {
			return nil, fmt.Errorf("scan notes: %w", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes: %w", err)
	}

	values := make([]int, len(notes))
	for i, n := range notes {
		if useMidi {
			values[i] = n.Midi
		} else if n.Pitch == "0" {
			values[i] = 0
		} else {
			values[i] = music.Voces[n.Pitch]
		}
	}

	seq := &IntervalSequence{
		Piece:     piece,
		Voice:     voice,
		NotesData: notes,
	}

	seq.IntervList, seq.IntervMapping = BuildIntervList(values)
	seq.InvertList = Negate(seq.IntervList)
	seq.InvertMapping = seq.IntervMapping

	reversed := make([]int, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}
	seq.RetroList, seq.RetroMapping = BuildIntervList(reversed)
	n := len(values)
	for i, idx := range seq.RetroMapping {
		seq.RetroMapping[i] = (n - 1) - idx
	}
	seq.RetroInvertList = Negate(seq.RetroList)
	seq.RetroInvertMapping = seq.RetroMapping

	return seq, nil
}

// Negate negates each interval in the slice (except 1).
func Negate(in []int) []int {
	out := make([]int, len(in))
	for i, v := range in {
		if v != 1 {
			out[i] = -v
		} else {
			out[i] = v
		}
	}
	return out
}

// BuildIntervList converts note values to interval sequences and tracks note mappings.
func BuildIntervList(values []int) (result []int, mapping []int) {
	n := len(values)
	i := 0
	for i < n {
		if values[i] == 0 {
			result = append(result, 0)
			mapping = append(mapping, i)
			for i < n && values[i] == 0 {
				i++
			}
			continue
		}
		j := i + 1
		if j < n {
			if values[j] == 0 {
				i++
			} else {
				diff := values[j] - values[i]
				var r int
				if diff >= 0 {
					r = diff + 1
				} else {
					r = diff - 1
				}
				result = append(result, r)
				mapping = append(mapping, i)
				i++
			}
		} else {
			i++
		}
	}
	return result, mapping
}

// NewPieceIntervalSequences builds an IntervalSequence for each distinct voice in piece.
func NewPieceIntervalSequences(ctx context.Context, database *db.DB, piece int, useMidi bool) (*PieceIntervalSequences, error) {
	rows, err := database.Pool.Query(ctx,
		`SELECT DISTINCT voice FROM notes WHERE piece = $1 ORDER BY voice`, piece)
	if err != nil {
		return nil, fmt.Errorf("query voices: %w", err)
	}
	var voices []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, err
		}
		voices = append(voices, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	p := &PieceIntervalSequences{Piece: piece}
	for _, v := range voices {
		seq, err := NewIntervalSequence(ctx, database, piece, v, useMidi)
		if err != nil {
			return nil, err
		}
		p.IntervalSequences = append(p.IntervalSequences, seq)
	}
	return p, nil
}

// NewQueryBlock builds a query block over pieces (or all pieces when pieces is empty).
func NewQueryBlock(ctx context.Context, database *db.DB, pieces []int, useMidi bool) (*QueryBlock, error) {
	if len(pieces) == 0 {
		rows, err := database.Pool.Query(ctx, `SELECT seq FROM pieces ORDER BY seq`)
		if err != nil {
			return nil, fmt.Errorf("query pieces: %w", err)
		}
		for rows.Next() {
			var p int
			if err := rows.Scan(&p); err != nil {
				rows.Close()
				return nil, err
			}
			pieces = append(pieces, p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	qb := &QueryBlock{UseMidi: useMidi}
	for _, p := range pieces {
		ps, err := NewPieceIntervalSequences(ctx, database, p, useMidi)
		if err != nil {
			return nil, err
		}
		qb.Blocks = append(qb.Blocks, ps)
	}
	return qb, nil
}

// SequenceOccurrence describes a single occurrence of a repeated pattern.
type SequenceOccurrence struct {
	Piece      int
	Voice      int
	ListType   string
	Offset     int
	Bar        int
	Index      int
	Pitch      string
	Toffset    float64
	EndBar     int
	EndIndex   int
	EndPitch   string
	EndToffset float64
}

// SequenceResult is one repeated maximal substring with its occurrences.
type SequenceResult struct {
	Sequence    []int
	Count       int
	Occurrences []SequenceOccurrence
}

type stringEntry struct {
	piece    int
	voice    int
	listType string
	values   []int
	mapping  []int
	notes    []NoteRow
}

// SequenceFinder searches for repeated interval substrings.
type SequenceFinder struct {
	QueryBlock  *QueryBlock
	VoiceFilter map[int]bool
	entries     []stringEntry
}

// NewSequenceFinder constructs a SequenceFinder.
func NewSequenceFinder(qb *QueryBlock, voiceNumbers []int) *SequenceFinder {
	var filter map[int]bool
	if voiceNumbers != nil {
		filter = make(map[int]bool, len(voiceNumbers))
		for _, v := range voiceNumbers {
			filter[v] = true
		}
	}
	f := &SequenceFinder{QueryBlock: qb, VoiceFilter: filter}
	f.prepare()
	return f
}

func (f *SequenceFinder) prepare() {
	for _, piece := range f.QueryBlock.Blocks {
		for _, seq := range piece.IntervalSequences {
			if f.VoiceFilter != nil && !f.VoiceFilter[seq.Voice] {
				continue
			}
			lists := []struct {
				name    string
				values  []int
				mapping []int
			}{
				{"intervList", seq.IntervList, seq.IntervMapping},
				{"invertList", seq.InvertList, seq.InvertMapping},
				{"retroList", seq.RetroList, seq.RetroMapping},
				{"retroInvertList", seq.RetroInvertList, seq.RetroInvertMapping},
			}
			for _, l := range lists {
				if len(l.values) == 0 {
					continue
				}
				f.entries = append(f.entries, stringEntry{
					piece:    piece.Piece,
					voice:    seq.Voice,
					listType: l.name,
					values:   l.values,
					mapping:  l.mapping,
					notes:    seq.NotesData,
				})
			}
		}
	}
}

type seqKey struct {
	s string
}

func keyOf(seq []int) seqKey {
	buf := make([]byte, 0, len(seq)*4)
	for _, v := range seq {
		buf = strconv.AppendInt(buf, int64(v), 10)
		buf = append(buf, ',')
	}
	return seqKey{s: string(buf)}
}

// FindRepeatedSubstrings returns all maximal repeated substrings of length >= x that occur >= y times.
func (f *SequenceFinder) FindRepeatedSubstrings(x, y int, excludeSamePiece bool) []SequenceResult {
	type occRef struct {
		entryIdx int
		offset   int
	}
	substringMap := make(map[seqKey][]occRef)
	canon := make(map[seqKey][]int)

	for ei, e := range f.entries {
		L := len(e.values)
		for start := 0; start+x <= L; start++ {
			for length := x; start+length <= L; length++ {
				sub := e.values[start : start+length]
				k := keyOf(sub)
				if _, ok := canon[k]; !ok {
					c := make([]int, length)
					copy(c, sub)
					canon[k] = c
				}
				substringMap[k] = append(substringMap[k], occRef{ei, start})
			}
		}
	}

	var results []SequenceResult
	for k, occs := range substringMap {
		if len(occs) < y {
			continue
		}
		occurrences := make([]SequenceOccurrence, 0, len(occs))
		pieces := make(map[int]struct{})
		for _, o := range occs {
			e := f.entries[o.entryIdx]
			startNote, endNote := getOccurrenceBoundaryNotes(e, o.offset, len(canon[k]))
			occurrences = append(occurrences, SequenceOccurrence{
				Piece:      e.piece,
				Voice:      e.voice,
				ListType:   e.listType,
				Offset:     o.offset,
				Bar:        startNote.Bar,
				Index:      startNote.Index,
				Pitch:      startNote.Pitch,
				Toffset:    startNote.Toffset,
				EndBar:     endNote.Bar,
				EndIndex:   endNote.Index,
				EndPitch:   endNote.Pitch,
				EndToffset: endNote.Toffset,
			})
			pieces[e.piece] = struct{}{}
		}
		if excludeSamePiece && len(pieces) < 2 {
			continue
		}
		if len(occurrences) < y {
			continue
		}
		results = append(results, SequenceResult{
			Sequence:    canon[k],
			Count:       len(occurrences),
			Occurrences: occurrences,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return len(results[i].Sequence) > len(results[j].Sequence)
	})

	maximal := make([]SequenceResult, 0, len(results))
	for i := range results {
		isMax := true
		cur := results[i].Sequence
		for j := range results {
			if i == j {
				continue
			}
			other := results[j].Sequence
			if len(other) <= len(cur) {
				continue
			}
			if containsSubseq(other, cur) && results[j].Count >= results[i].Count {
				isMax = false
				break
			}
		}
		if isMax {
			maximal = append(maximal, results[i])
		}
	}
	return maximal
}

// FindCrossPieceSubstrings returns maximal substrings from sourcePiece occurring in other pieces.
func (f *SequenceFinder) FindCrossPieceSubstrings(sourcePiece int, x int) []SequenceResult {
	type occRef struct {
		entryIdx int
		offset   int
	}
	sourceOccs := make(map[seqKey][]occRef)
	canon := make(map[seqKey][]int)

	for ei, e := range f.entries {
		if e.piece != sourcePiece {
			continue
		}
		L := len(e.values)
		for start := 0; start+x <= L; start++ {
			for length := x; start+length <= L; length++ {
				sub := e.values[start : start+length]
				k := keyOf(sub)
				if _, ok := canon[k]; !ok {
					c := make([]int, length)
					copy(c, sub)
					canon[k] = c
				}
				sourceOccs[k] = append(sourceOccs[k], occRef{ei, start})
			}
		}
	}

	if len(sourceOccs) == 0 {
		return nil
	}

	targetOccs := make(map[seqKey][]occRef)
	for ei, e := range f.entries {
		if e.piece == sourcePiece {
			continue
		}
		L := len(e.values)
		for start := 0; start+x <= L; start++ {
			for length := x; start+length <= L; length++ {
				sub := e.values[start : start+length]
				k := keyOf(sub)
				if _, exists := sourceOccs[k]; !exists {
					break
				}
				targetOccs[k] = append(targetOccs[k], occRef{ei, start})
			}
		}
	}

	var results []SequenceResult
	for k, tOccs := range targetOccs {
		if len(tOccs) == 0 {
			continue
		}
		sOccs := sourceOccs[k]
		occurrences := make([]SequenceOccurrence, 0, len(sOccs)+len(tOccs))
		for _, o := range sOccs {
			e := f.entries[o.entryIdx]
			startNote, endNote := getOccurrenceBoundaryNotes(e, o.offset, len(canon[k]))
			occurrences = append(occurrences, SequenceOccurrence{
				Piece:      e.piece,
				Voice:      e.voice,
				ListType:   e.listType,
				Offset:     o.offset,
				Bar:        startNote.Bar,
				Index:      startNote.Index,
				Pitch:      startNote.Pitch,
				Toffset:    startNote.Toffset,
				EndBar:     endNote.Bar,
				EndIndex:   endNote.Index,
				EndPitch:   endNote.Pitch,
				EndToffset: endNote.Toffset,
			})
		}
		for _, o := range tOccs {
			e := f.entries[o.entryIdx]
			startNote, endNote := getOccurrenceBoundaryNotes(e, o.offset, len(canon[k]))
			occurrences = append(occurrences, SequenceOccurrence{
				Piece:      e.piece,
				Voice:      e.voice,
				ListType:   e.listType,
				Offset:     o.offset,
				Bar:        startNote.Bar,
				Index:      startNote.Index,
				Pitch:      startNote.Pitch,
				Toffset:    startNote.Toffset,
				EndBar:     endNote.Bar,
				EndIndex:   endNote.Index,
				EndPitch:   endNote.Pitch,
				EndToffset: endNote.Toffset,
			})
		}
		results = append(results, SequenceResult{
			Sequence:    canon[k],
			Count:       len(occurrences),
			Occurrences: occurrences,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return len(results[i].Sequence) > len(results[j].Sequence)
	})

	maximal := make([]SequenceResult, 0, len(results))
	for i := range results {
		isMax := true
		cur := results[i].Sequence
		for j := range results {
			if i == j {
				continue
			}
			other := results[j].Sequence
			if len(other) <= len(cur) {
				continue
			}
			if containsSubseq(other, cur) && results[j].Count >= results[i].Count {
				isMax = false
				break
			}
		}
		if isMax {
			maximal = append(maximal, results[i])
		}
	}
	return maximal
}

func containsSubseq(haystack, needle []int) bool {
	if len(needle) == 0 {
		return true
	}
	for start := 0; start+len(needle) <= len(haystack); start++ {
		match := true
		for k := range needle {
			if haystack[start+k] != needle[k] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func getOccurrenceBoundaryNotes(e stringEntry, offset, length int) (startNote, endNote NoteRow) {
	startIdx := e.mapping[offset]
	startNote = e.notes[startIdx]

	lastIdx := offset + length - 1
	lastNoteIdx := e.mapping[lastIdx]
	lastVal := e.values[lastIdx]

	if e.listType == "intervList" || e.listType == "invertList" {
		if lastVal != 0 && lastNoteIdx+1 < len(e.notes) {
			endNote = e.notes[lastNoteIdx+1]
		} else {
			endNote = e.notes[lastNoteIdx]
		}
	} else {
		if lastVal != 0 && lastNoteIdx-1 >= 0 {
			endNote = e.notes[lastNoteIdx-1]
		} else {
			endNote = e.notes[lastNoteIdx]
		}
	}
	return startNote, endNote
}

type occurrenceSpan struct {
	piece      int
	voice      int
	minNoteIdx int
	maxNoteIdx int
	minToffset float64
	maxToffset float64
}

func resultSpanKey(r SequenceResult) string {
	spans := make([]occurrenceSpan, len(r.Occurrences))
	for i, occ := range r.Occurrences {
		minIdx := occ.Index
		maxIdx := occ.EndIndex
		if minIdx > maxIdx {
			minIdx, maxIdx = maxIdx, minIdx
		}
		minToff := occ.Toffset
		maxToff := occ.EndToffset
		if minToff > maxToff {
			minToff, maxToff = maxToff, minToff
		}
		spans[i] = occurrenceSpan{
			piece:      occ.Piece,
			voice:      occ.Voice,
			minNoteIdx: minIdx,
			maxNoteIdx: maxIdx,
			minToffset: minToff,
			maxToffset: maxToff,
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].piece != spans[j].piece {
			return spans[i].piece < spans[j].piece
		}
		if spans[i].voice != spans[j].voice {
			return spans[i].voice < spans[j].voice
		}
		if spans[i].minNoteIdx != spans[j].minNoteIdx {
			return spans[i].minNoteIdx < spans[j].minNoteIdx
		}
		if spans[i].maxNoteIdx != spans[j].maxNoteIdx {
			return spans[i].maxNoteIdx < spans[j].maxNoteIdx
		}
		if spans[i].minToffset != spans[j].minToffset {
			return spans[i].minToffset < spans[j].minToffset
		}
		return spans[i].maxToffset < spans[j].maxToffset
	})

	var buf strings.Builder
	for _, s := range spans {
		fmt.Fprintf(&buf, "%d:%d:%d-%d:%.2f-%.2f;", s.piece, s.voice, s.minNoteIdx, s.maxNoteIdx, s.minToffset, s.maxToffset)
	}
	return buf.String()
}

func listTypePriority(listType string) int {
	switch listType {
	case "intervList":
		return 0
	case "invertList":
		return 1
	case "retroList":
		return 2
	case "retroInvertList":
		return 3
	default:
		return 4
	}
}

func resultPriority(r SequenceResult, sourcePiece int) int {
	best := 999
	for _, occ := range r.Occurrences {
		if sourcePiece > 0 && occ.Piece == sourcePiece {
			p := listTypePriority(occ.ListType)
			if p < best {
				best = p
			}
		}
	}
	if best != 999 {
		return best
	}
	for _, occ := range r.Occurrences {
		p := listTypePriority(occ.ListType)
		if p < best {
			best = p
		}
	}
	if best == 999 {
		return 4
	}
	return best
}

func compareResults(a, b SequenceResult, sourcePiece int) bool {
	pA := resultPriority(a, sourcePiece)
	pB := resultPriority(b, sourcePiece)
	if pA != pB {
		return pA < pB
	}
	minLen := len(a.Occurrences)
	if len(b.Occurrences) < minLen {
		minLen = len(b.Occurrences)
	}
	for i := 0; i < minLen; i++ {
		pOccA := listTypePriority(a.Occurrences[i].ListType)
		pOccB := listTypePriority(b.Occurrences[i].ListType)
		if pOccA != pOccB {
			return pOccA < pOccB
		}
	}
	return false
}

// FilterRepeatedResults filters results that share the same piece identifiers and boundary notes.
func FilterRepeatedResults(results []SequenceResult, sourcePiece int) []SequenceResult {
	if len(results) == 0 {
		return results
	}
	groups := make(map[string]SequenceResult)
	var groupOrder []string

	for _, r := range results {
		key := resultSpanKey(r)
		existing, ok := groups[key]
		if !ok {
			groups[key] = r
			groupOrder = append(groupOrder, key)
		} else {
			if compareResults(r, existing, sourcePiece) {
				groups[key] = r
			}
		}
	}

	filtered := make([]SequenceResult, 0, len(groupOrder))
	for _, key := range groupOrder {
		filtered = append(filtered, groups[key])
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if len(filtered[i].Sequence) != len(filtered[j].Sequence) {
			return len(filtered[i].Sequence) > len(filtered[j].Sequence)
		}
		return filtered[i].Count > filtered[j].Count
	})

	return filtered
}

// PrintResults writes results to stdout and optional file.
func (f *SequenceFinder) PrintResults(results []SequenceResult, toFile string) error {
	var out *os.File
	if toFile != "" {
		var err error
		out, err = os.OpenFile(toFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	emit := func(line string) {
		fmt.Print(line)
		if out != nil {
			_, _ = out.WriteString(line)
		}
	}
	for _, r := range results {
		emit(fmt.Sprintf("Sequence: %v\n", r.Sequence))
		emit(fmt.Sprintf("Length: %d\n", len(r.Sequence)))
		emit(fmt.Sprintf("Count: %d\n", r.Count))
		for _, occ := range r.Occurrences {
			emit(fmt.Sprintf("  Piece: %d, Voice: %d, List: %s, Bar: %d, Index: %d, Pitch: %s, Offset: %v\n",
				occ.Piece, occ.Voice, occ.ListType, occ.Bar, occ.Index, occ.Pitch, occ.Toffset))
		}
		emit("--------------------\n")
	}
	return nil
}

// EstimateTime estimates computation time for search.
func (f *SequenceFinder) EstimateTime(x int) time.Duration {
	const secPerOp = 5.8e-6
	total := 0.0
	for _, e := range f.entries {
		L := len(e.values)
		if L >= x {
			n := L - x + 1
			total += float64(n) * float64(n+1) / 2.0
		}
	}
	return time.Duration(total * secPerOp * float64(time.Second))
}
