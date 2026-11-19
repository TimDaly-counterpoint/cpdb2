package score

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func findEndOfBrace(content string, startPos int) int {
	count := 0
	for i := startPos; i < len(content); i++ {
		if content[i] == '{' {
			count++
		} else if content[i] == '}' {
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

func LoadLilyPond(path string) (*Score, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	r, err := makeUTF8Reader(f)
	if err != nil {
		return nil, fmt.Errorf("reader: %w", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	content := string(data)

	score := &Score{
		Partwise: &ScorePartwise{
			Work:           &Work{},
			Identification: &Identification{},
			PartList:       PartList{},
		},
	}

	// Extract title
	reTitle := regexp.MustCompile(`title\s*=\s*"([^"]*)"`)
	if match := reTitle.FindStringSubmatch(content); len(match) > 1 {
		score.Partwise.Work.WorkTitle = match[1]
	}

	// Extract composer
	reComposer := regexp.MustCompile(`composer\s*=\s*"([^"]*)"`)
	if match := reComposer.FindStringSubmatch(content); len(match) > 1 {
		score.Partwise.Identification.Creators = append(score.Partwise.Identification.Creators, TypedText{Type: "composer", Value: match[1]})
	}

	isGerman := strings.Contains(content, "deutsch.ly")

	// Global defaults
	var globalKey *Key
	if km := regexp.MustCompile(`\\key\s+([a-h][is|es]*)\s+\\(\w+)`).FindStringSubmatch(content); len(km) > 2 {
		globalKey = &Key{Fifths: parseLPKey(km[1], km[2])}
	}
	var globalTime *Time
	if tm := regexp.MustCompile(`\\time\s+(\d+)/(\d+)`).FindStringSubmatch(content); len(tm) > 2 {
		globalTime = &Time{Beats: tm[1], BeatType: tm[2]}
	}

	// Find all variable definitions
	reVar := regexp.MustCompile(`(?m)^(\w+)\s*=\s*(\\relative\s+[a-h][',]*\s*)?\s*\{([\s\S]*?)\n\s*\}`)
	varMatches := reVar.FindAllStringSubmatch(content, -1)

	// Map to store measures for each variable
	varMeasures := make(map[string][]Measure)
	for _, m := range varMatches {
		varName := m[1]
		if strings.Contains(varName, "Lyrics") || varName == "global" {
			continue
		}

		relativePitch := ""
		if m[2] != "" {
			relPart := strings.TrimSpace(m[2])
			if strings.HasPrefix(relPart, "\\relative") {
				relativePitch = strings.TrimSpace(relPart[len("\\relative"):])
			}
		}
		notesContent := m[3]
		measures := parseLilyPondNotes(notesContent, relativePitch, isGerman, 96)

		if len(measures) > 0 {
			varMeasures[varName] = append(varMeasures[varName], measures...)
		}
	}

	var scoreContents []string
	pos := 0
	for {
		idx := strings.Index(content[pos:], "\\score")
		if idx == -1 {
			break
		}
		pos += idx + 6
		braceIdx := strings.Index(content[pos:], "{")
		if braceIdx == -1 {
			continue
		}
		startBrace := pos + braceIdx
		endBrace := findEndOfBrace(content, startBrace)
		if endBrace == -1 {
			break
		}
		scoreContents = append(scoreContents, content[startBrace+1:endBrace])
		pos = endBrace + 1
	}

	var orderedVoiceVars []string
	seenVoices := make(map[string]bool)

	for _, scoreContent := range scoreContents {
		reRef := regexp.MustCompile(`\\(\w+)`)
		refMatches := reRef.FindAllStringIndex(scoreContent, -1)
		for _, rmIndex := range refMatches {
			vName := scoreContent[rmIndex[0]+1 : rmIndex[1]]
			if measures, ok := varMeasures[vName]; ok && !seenVoices[vName] {
				prefix := scoreContent[:rmIndex[0]]

				var attr Attributes
				attr.Divisions = 24

				foundAttr := false
				if cm := regexp.MustCompile(`\\clef\s+"?([\w_]+)"?`).FindAllStringSubmatch(prefix, -1); len(cm) > 0 {
					attr.Clef = []Clef{parseLPClef(cm[len(cm)-1][1])}
					foundAttr = true
				}

				if km := regexp.MustCompile(`\\key\s+([a-h][is|es]*)\s+\\(\w+)`).FindAllStringSubmatch(prefix, -1); len(km) > 0 {
					last := km[len(km)-1]
					attr.Key = []Key{{Fifths: parseLPKey(last[1], last[2])}}
					foundAttr = true
				} else if globalKey != nil {
					attr.Key = []Key{*globalKey}
					foundAttr = true
				}

				if tm := regexp.MustCompile(`\\time\s+(\d+)/(\d+)`).FindAllStringSubmatch(prefix, -1); len(tm) > 0 {
					last := tm[len(tm)-1]
					attr.Time = []Time{{Beats: last[1], BeatType: last[2]}}
					foundAttr = true
				} else if globalTime != nil {
					attr.Time = []Time{*globalTime}
					foundAttr = true
				}

				if foundAttr && len(measures) > 0 {
					m0 := &measures[0]
					if m0.Attributes == nil {
						m0.Attributes = &attr
					} else {
						if len(attr.Clef) > 0 {
							m0.Attributes.Clef = attr.Clef
						}
						if len(attr.Key) > 0 {
							m0.Attributes.Key = attr.Key
						}
						if len(attr.Time) > 0 {
							m0.Attributes.Time = attr.Time
						}
					}
					m0.Elements = append([]any{attr}, m0.Elements...)
				}

				orderedVoiceVars = append(orderedVoiceVars, vName)
				seenVoices[vName] = true
			}
		}
	}

	if len(orderedVoiceVars) == 0 {
		reStaff := regexp.MustCompile(`\\new Staff = "([^"]+)"[\s\S]*?\\context Voice = "([^"]+)"`)
		staffMatches := reStaff.FindAllStringSubmatch(content, -1)
		for _, sm := range staffMatches {
			voiceVar := sm[2]
			if _, ok := varMeasures[voiceVar]; ok && !seenVoices[voiceVar] {
				orderedVoiceVars = append(orderedVoiceVars, voiceVar)
				seenVoices[voiceVar] = true
			}
		}
	}

	if len(orderedVoiceVars) == 0 {
		for _, m := range varMatches {
			vName := m[1]
			if _, ok := varMeasures[vName]; ok && !seenVoices[vName] {
				orderedVoiceVars = append(orderedVoiceVars, vName)
				seenVoices[vName] = true
			}
		}
	}

	for _, vName := range orderedVoiceVars {
		part := Part{ID: vName}
		part.Measures = varMeasures[vName]

		score.Partwise.Parts = append(score.Partwise.Parts, part)
		score.Partwise.PartList.ScoreParts = append(score.Partwise.PartList.ScoreParts, ScorePart{ID: vName})
	}

	return score, nil
}

type lpState struct {
	currentStep       int
	currentOctave     int
	currentDuration   int
	currentDots       int
	divisions         int
	timeBeats         int
	timeType          int
	measureLimit      int
	currentMeasure    int
	currentMeasureDur int
	previousHadTie    bool
	isGerman          bool
}

func parseLilyPondNotes(content string, relativePitch string, isGerman bool, defaultMeasureLimit int) []Measure {
	state := &lpState{
		divisions:       24,
		currentDuration: 4,
		timeBeats:       4,
		timeType:        4,
		measureLimit:    defaultMeasureLimit,
		currentMeasure:  1,
		isGerman:        isGerman,
	}

	if relativePitch != "" {
		state.currentStep, state.currentOctave = parseLPPitch(relativePitch, isGerman)
	} else {
		state.currentStep = 0
		state.currentOctave = 4
	}

	var measures []Measure
	currentMeasure := Measure{Number: "1"}

	reComment := regexp.MustCompile(`%.*`)
	content = reComment.ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "\n", " ")

	tokens := strings.Fields(content)
	i := 0
	for i < len(tokens) {
		token := tokens[i]

		if strings.HasPrefix(token, "\\clef") {
			if i+1 < len(tokens) {
				clefName := strings.Trim(tokens[i+1], "\"")
				i += 2
				attr := getAttributes(state)
				attr.Clef = []Clef{parseLPClef(clefName)}
				addAttributesToMeasure(&currentMeasure, attr)
			} else {
				i++
			}
			continue
		}

		if strings.HasPrefix(token, "\\time") {
			if i+1 < len(tokens) {
				timeStr := tokens[i+1]
				i += 2
				b, t := parseLPTime(timeStr)
				state.timeBeats = b
				state.timeType = t
				state.measureLimit = (b * 96) / t

				attr := getAttributes(state)
				attr.Time = []Time{{Beats: strconv.Itoa(b), BeatType: strconv.Itoa(t)}}
				addAttributesToMeasure(&currentMeasure, attr)
			} else {
				i++
			}
			continue
		}

		if strings.HasPrefix(token, "\\key") {
			if i+2 < len(tokens) {
				keyStep := tokens[i+1]
				keyMode := tokens[i+2]
				i += 3
				fifths := parseLPKey(keyStep, keyMode)

				attr := getAttributes(state)
				attr.Key = []Key{{Fifths: fifths, Mode: keyMode}}
				addAttributesToMeasure(&currentMeasure, attr)
			} else {
				i += len(tokens) - i
			}
			continue
		}

		if token == "|" || strings.HasPrefix(token, "\\barNumberCheck") || strings.HasPrefix(token, "\\bar") {
			if strings.HasPrefix(token, "\\bar") && !strings.HasPrefix(token, "\\barNumberCheck") {
				if len(currentMeasure.Elements) > 0 {
					currentMeasure.Elements = append(currentMeasure.Elements, Barline{})
				} else if len(measures) > 0 {
					measures[len(measures)-1].Elements = append(measures[len(measures)-1].Elements, Barline{})
				}
			}

			if state.currentMeasureDur > 0 {
				measures = append(measures, currentMeasure)
				state.currentMeasure++
				currentMeasure = Measure{Number: strconv.Itoa(state.currentMeasure)}
				state.currentMeasureDur = 0
			}
			if strings.HasPrefix(token, "\\barNumberCheck") {
				if i+1 < len(tokens) {
					numStr := strings.TrimPrefix(tokens[i+1], "#")
					if n, err := strconv.Atoi(numStr); err == nil {
						state.currentMeasure = n
						currentMeasure.Number = numStr
					}
					i += 2
				} else {
					i++
				}
			} else if strings.HasPrefix(token, "\\bar") {
				i += 2
			} else {
				i++
			}
			continue
		}

		isFicta := false
		if token == "\\ficta" {
			isFicta = true
			i++
			if i < len(tokens) {
				token = tokens[i]
			} else {
				continue
			}
		}

		if strings.HasPrefix(token, "\\") && !isFicta {
			i++
			continue
		}

		consumedSeparateTie := false
		if i+1 < len(tokens) && tokens[i+1] == "~" {
			token += "~"
			consumedSeparateTie = true
		}

		if note, ok := parseLPNote(token, state); ok {
			if isFicta {
				note.AccidentalEditorial = true
			}
			if consumedSeparateTie {
				i++
			}
			currentMeasure.Elements = append(currentMeasure.Elements, note)
			currentMeasure.Notes = append(currentMeasure.Notes, note)
			state.currentMeasureDur += note.Duration

			for state.currentMeasureDur >= state.measureLimit {
				measures = append(measures, currentMeasure)
				state.currentMeasure++
				currentMeasure = Measure{Number: strconv.Itoa(state.currentMeasure)}
				state.currentMeasureDur -= state.measureLimit
			}
			i++
			continue
		}

		i++
	}

	if len(currentMeasure.Elements) > 0 {
		measures = append(measures, currentMeasure)
	}

	return measures
}

func getAttributes(state *lpState) *Attributes {
	return &Attributes{Divisions: state.divisions}
}

func addAttributesToMeasure(m *Measure, attr *Attributes) {
	if m.Attributes == nil {
		m.Attributes = attr
	} else {
		if len(attr.Clef) > 0 {
			m.Attributes.Clef = attr.Clef
		}
		if len(attr.Time) > 0 {
			m.Attributes.Time = attr.Time
		}
		if len(attr.Key) > 0 {
			m.Attributes.Key = attr.Key
		}
	}
	m.Elements = append(m.Elements, *attr)
}

func parseLPPitch(p string, isGerman bool) (step int, octave int) {
	s := p[0]
	if isGerman && (s == 'h' || s == 'H') {
		step = 6
	} else {
		step = int(strings.ToUpper(string(s))[0]) - int('C')
		if step < 0 {
			step += 7
		}
	}

	octave = 3
	for _, char := range p[1:] {
		if char == '\'' {
			octave++
		}
		if char == ',' {
			octave--
		}
	}
	return step, octave
}

func parseLPClef(name string) Clef {
	switch name {
	case "treble", "violin":
		return Clef{Sign: "G", Line: 2}
	case "alto":
		return Clef{Sign: "C", Line: 3}
	case "tenor":
		return Clef{Sign: "C", Line: 4}
	case "bass":
		return Clef{Sign: "F", Line: 4}
	case "G_8":
		return Clef{Sign: "G", Line: 2}
	default:
		return Clef{Sign: "G", Line: 2}
	}
}

func parseLPTime(s string) (beats int, beatType int) {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		beats, _ = strconv.Atoi(parts[0])
		beatType, _ = strconv.Atoi(parts[1])
		return beats, beatType
	}
	return 4, 4
}

func parseLPKey(step string, mode string) int {
	keys := map[string]int{
		"c": 0, "g": 1, "d": 2, "a": 3, "e": 4, "b": 5, "fis": 6,
		"f": -1, "bes": -2, "ees": -3, "aes": -4, "des": -5, "ges": -6,
	}
	fifths := keys[strings.ToLower(step)]
	if mode == "\\minor" {
		fifths -= 3
	}
	return fifths
}

var lpNoteRegex = regexp.MustCompile(`^([a-hRr](?:is|es|isis|eses|s)?)[!?]*([',]*)(\d+|\\breve|\\longa)?(\.*)(~)?`)

func parseLPNote(token string, state *lpState) (Note, bool) {
	match := lpNoteRegex.FindStringSubmatch(token)
	if match == nil {
		return Note{}, false
	}

	matchedLen := len(match[0])
	if matchedLen < len(token) {
		remainder := token[matchedLen:]
		if !strings.HasPrefix(remainder, "-") && !strings.HasPrefix(remainder, "^") && !strings.HasPrefix(remainder, "_") {
			return Note{}, false
		}
	}

	stepName := match[1]
	octaveMarks := match[2]
	durationStr := match[3]
	dotsStr := match[4]
	hasTie := match[5] == "~"

	if stepName == "R" || stepName == "r" {
		dur := parseLPDuration(durationStr, state.currentDuration)
		state.currentDuration = dur
		dots := len(dotsStr)

		actualDur := calculateDuration(dur, dots)
		if stepName == "R" && dur == 1 && dots == 1 {
			actualDur = state.measureLimit
		}

		return Note{Rest: &Rest{}, Duration: actualDur}, true
	}

	step, alter := parseLPStep(stepName, state.isGerman)

	currentAbs := (state.currentOctave+1)*7 + state.currentStep
	diff := step - state.currentStep
	for diff > 3 {
		diff -= 7
	}
	for diff < -3 {
		diff += 7
	}
	newAbs := currentAbs + diff

	for _, char := range octaveMarks {
		if char == '\'' {
			newAbs += 7
		}
		if char == ',' {
			newAbs -= 7
		}
	}

	state.currentOctave = (newAbs / 7) - 1
	state.currentStep = newAbs % 7

	dur := parseLPDuration(durationStr, state.currentDuration)
	state.currentDuration = dur
	dots := len(dotsStr)
	actualDur := calculateDuration(dur, dots)

	var ties []Tie
	if state.previousHadTie {
		ties = append(ties, Tie{Type: "stop"})
	}
	if hasTie {
		ties = append(ties, Tie{Type: "start"})
	}
	state.previousHadTie = hasTie

	note := Note{
		Pitch:    &Pitch{Step: strings.ToUpper(string("CDEFGAB"[step])), Alter: alter, Octave: state.currentOctave},
		Duration: actualDur,
		Tie:      ties,
	}

	return note, true
}

func parseLPStep(s string, isGerman bool) (step int, alter int) {
	lower := strings.ToLower(s)
	stepChar := lower[0]

	if isGerman && stepChar == 'h' {
		step = 6
		alter = 0
	} else if isGerman && stepChar == 'b' && len(lower) == 1 {
		step = 6
		alter = -1
	} else {
		step = int(strings.ToUpper(string(stepChar))[0]) - int('C')
		if step < 0 {
			step += 7
		}
	}

	if strings.HasSuffix(lower, "isis") {
		alter = 2
	} else if strings.HasSuffix(lower, "eses") {
		alter = -2
	} else if strings.HasSuffix(lower, "is") {
		alter = 1
	} else if strings.HasSuffix(lower, "es") || (isGerman && strings.HasSuffix(lower, "s") && len(lower) > 1) {
		alter = -1
	}
	return step, alter
}

func calculateDuration(dur int, dots int) int {
	var base int
	if dur == -1 {
		base = 192
	} else if dur == -2 {
		base = 384
	} else if dur > 0 {
		base = 96 / dur
	} else {
		return 0
	}
	res := base
	for i := 0; i < dots; i++ {
		base /= 2
		res += base
	}
	return res
}

func parseLPDuration(durStr string, currentDur int) int {
	if durStr == "" {
		return currentDur
	}
	if durStr == "\\breve" {
		return -1
	}
	if durStr == "\\longa" {
		return -2
	}
	d, _ := strconv.Atoi(durStr)
	return d
}
