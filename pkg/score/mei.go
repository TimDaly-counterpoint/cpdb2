package score

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

type MEI struct {
	XMLName     xml.Name `xml:"mei"`
	ID          string   `xml:"id,attr"`
	Version     string   `xml:"meiversion,attr"`
	MEIHead     MEIHead  `xml:"meiHead"`
	Music       MEIMusic `xml:"music"`
	RawInnerXML string   `xml:",innerxml"`
}

type MEIHead struct {
	FileDesc          MEIFileDesc          `xml:"fileDesc"`
	EncodingDesc      MEIEncodingDesc      `xml:"encodingDesc"`
	WorkList          MEIWorkList          `xml:"workList"`
	ManifestationList MEIManifestationList `xml:"manifestationList"`
}

type MEIFileDesc struct {
	TitleStmt MEITitleStmt `xml:"titleStmt"`
	PubStmt   MEIPubStmt   `xml:"pubStmt"`
}

type MEITitleStmt struct {
	Title    string        `xml:"title"`
	RespStmt []MEIRespStmt `xml:"respStmt"`
}

type MEIRespStmt struct {
	PersNames []MEIPersName `xml:"persName"`
}

type MEIPersName struct {
	Role    string `xml:"role,attr"`
	Auth    string `xml:"auth,attr"`
	AuthURI string `xml:"auth.uri,attr"`
	Value   string `xml:",chardata"`
}

type MEIPubStmt struct {
	Publisher    string   `xml:"publisher"`
	Distributors []string `xml:"distributor"`
	Date         MEIDate  `xml:"date"`
	Availability string   `xml:"availability"`
}

type MEIDate struct {
	ISODate string `xml:"isodate,attr"`
	Value   string `xml:",chardata"`
}

type MEIEncodingDesc struct {
	AppInfo MEIAppInfo `xml:"appInfo"`
}

type MEIAppInfo struct {
	Applications []MEIApplication `xml:"application"`
}

type MEIApplication struct {
	Version string `xml:"version,attr"`
	Name    string `xml:"name"`
}

type MEIWorkList struct {
	Works []MEIWork `xml:"work"`
}

type MEIWork struct {
	Title          string            `xml:"title"`
	Composers      []MEIComposer     `xml:"composer"`
	Classification MEIClassification `xml:"classification"`
}

type MEIComposer struct {
	PersName *MEIPersName `xml:"persName"`
	Value    string       `xml:",chardata"`
}

type MEIClassification struct {
	TermList MEITermList `xml:"termList"`
}

type MEITermList struct {
	Terms []string `xml:"term"`
}

type MEIManifestationList struct {
	Manifestations []MEIManifestation `xml:"manifestation"`
}

type MEIManifestation struct {
	TitleStmt MEITitleStmt `xml:"titleStmt"`
	PubStmt   MEIPubStmt   `xml:"pubStmt"`
	PhysLoc   MEIPhysLoc   `xml:"physLoc"`
}

type MEIPhysLoc struct {
	Repository MEIRepository `xml:"repository"`
}

type MEIRepository struct {
	CorpName   string `xml:"corpName"`
	Identifier string `xml:"identifier"`
	GeogName   string `xml:"geogName"`
}

type MEIMusic struct {
	Body MEIBody `xml:"body"`
}

type MEIBody struct {
	MDivs []MEIMDiv `xml:"mdiv"`
}

type MEIMDiv struct {
	Score MEIScore `xml:"score"`
}

type MEIScore struct {
	Elements []any `xml:"-"`
}

func (s *MEIScore) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch tt := t.(type) {
		case xml.StartElement:
			switch tt.Name.Local {
			case "scoreDef":
				var sd MEIScoreDef
				if err := d.DecodeElement(&sd, &tt); err != nil {
					return err
				}
				s.Elements = append(s.Elements, sd)
			case "section":
				var sec MEISection
				if err := d.DecodeElement(&sec, &tt); err != nil {
					return err
				}
				s.Elements = append(s.Elements, sec)
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if tt == start.End() {
				return nil
			}
		}
	}
}

type MEIScoreDef struct {
	MeterCount   string        `xml:"meter.count,attr"`
	MeterUnit    string        `xml:"meter.unit,attr"`
	KeySig       string        `xml:"key.sig,attr"`
	KeyMode      string        `xml:"key.mode,attr"`
	ClefLine     int           `xml:"clef.line,attr"`
	ClefShape    string        `xml:"clef.shape,attr"`
	ClefDis      int           `xml:"clef.dis,attr"`
	ClefDisPlace string        `xml:"clef.dis.place,attr"`
	PPQ          int           `xml:"ppq,attr"`
	StaffGrps    []MEIStaffGrp `xml:"staffGrp"`
}

type MEIStaffGrp struct {
	N         string        `xml:"n,attr"`
	Symbol    string        `xml:"symbol,attr"`
	StaffDefs []MEIStaffDef `xml:"staffDef"`
	StaffGrps []MEIStaffGrp `xml:"staffGrp"`
}

type MEIStaffDef struct {
	N            string       `xml:"n,attr"`
	Label        string       `xml:"label,attr"`
	Lines        int          `xml:"lines,attr"`
	MeterCount   string       `xml:"meter.count,attr"`
	MeterUnit    string       `xml:"meter.unit,attr"`
	ClefLine     int          `xml:"clef.line,attr"`
	ClefShape    string       `xml:"clef.shape,attr"`
	ClefDis      int          `xml:"clef.dis,attr"`
	ClefDisPlace string       `xml:"clef.dis.place,attr"`
	KeySig       string       `xml:"key.sig,attr"`
	KeyMode      string       `xml:"key.mode,attr"`
	TransDiat    int          `xml:"trans.diat,attr"`
	TransSemi    int          `xml:"trans.semi,attr"`
	LabelElement string       `xml:"label"`
	LabelAbbr    string       `xml:"labelAbbr"`
	InstrDef     *MEIInstrDef `xml:"instrDef"`
}

type MEIInstrDef struct {
	MIDIChannel int `xml:"midi.channel,attr"`
	MIDIPan     int `xml:"midi.pan,attr"`
	MIDIVolume  int `xml:"midi.volume,attr"`
}

type MEISection struct {
	Elements []any `xml:"-"`
}

func (s *MEISection) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch tt := t.(type) {
		case xml.StartElement:
			switch tt.Name.Local {
			case "measure":
				var m MEIMeasure
				if err := d.DecodeElement(&m, &tt); err != nil {
					return err
				}
				s.Elements = append(s.Elements, m)
			case "scoreDef":
				var sd MEIScoreDef
				if err := d.DecodeElement(&sd, &tt); err != nil {
					return err
				}
				s.Elements = append(s.Elements, sd)
			case "staffDef":
				var sd MEIStaffDef
				if err := d.DecodeElement(&sd, &tt); err != nil {
					return err
				}
				s.Elements = append(s.Elements, sd)
			case "section":
				var sec MEISection
				if err := d.DecodeElement(&sec, &tt); err != nil {
					return err
				}
				s.Elements = append(s.Elements, sec)
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if tt == start.End() {
				return nil
			}
		}
	}
}

type MEIMeasure struct {
	ID     string     `xml:"id,attr"`
	Label  string     `xml:"label,attr"`
	N      string     `xml:"n,attr"`
	Right  string     `xml:"right,attr"`
	Staffs []MEIStaff `xml:"staff"`
}

type MEIStaff struct {
	ID     string     `xml:"id,attr"`
	N      string     `xml:"n,attr"`
	Layers []MEILayer `xml:"layer"`
}

type MEILayer struct {
	ID       string     `xml:"id,attr"`
	N        string     `xml:"n,attr"`
	Notes    []MEINote  `xml:"note"`
	Rests    []MEIRest  `xml:"rest"`
	MRests   []MEIRest  `xml:"mRest"`
	Chords   []MEIChord `xml:"chord"`
	Beams    []MEIBeam  `xml:"beam"`
	Elements []any      `xml:"-"`
}

func (l *MEILayer) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "id":
			l.ID = attr.Value
		case "n":
			l.N = attr.Value
		}
	}

	for {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch tt := t.(type) {
		case xml.StartElement:
			var element any
			switch tt.Name.Local {
			case "note":
				var note MEINote
				if err := d.DecodeElement(&note, &tt); err != nil {
					return err
				}
				l.Notes = append(l.Notes, note)
				element = note
			case "rest", "mRest":
				var rest MEIRest
				if err := d.DecodeElement(&rest, &tt); err != nil {
					return err
				}
				if tt.Name.Local == "mRest" {
					l.MRests = append(l.MRests, rest)
				} else {
					l.Rests = append(l.Rests, rest)
				}
				element = rest
			case "chord":
				var chord MEIChord
				if err := d.DecodeElement(&chord, &tt); err != nil {
					return err
				}
				l.Chords = append(l.Chords, chord)
				element = chord
			case "beam":
				var beam MEIBeam
				if err := d.DecodeElement(&beam, &tt); err != nil {
					return err
				}
				l.Beams = append(l.Beams, beam)
				element = beam
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
			if element != nil {
				l.Elements = append(l.Elements, element)
			}
		case xml.EndElement:
			if tt == start.End() {
				return nil
			}
		}
	}
}

type MEINote struct {
	ID       string        `xml:"id,attr"`
	Duration string        `xml:"dur,attr"`
	DurPPQ   int           `xml:"dur.ppq,attr"`
	Dots     int           `xml:"dots,attr"`
	Octave   int           `xml:"oct,attr"`
	PName    string        `xml:"pname,attr"`
	PNum     int           `xml:"pnum,attr"`
	Accid    string        `xml:"accid,attr"`
	Reason   string        `xml:"reason,attr"`
	Color    string        `xml:"color,attr"`
	StemDir  string        `xml:"stem.dir,attr"`
	Tie      string        `xml:"tie,attr"`
	Verses   []MEIVerse    `xml:"verse"`
	Accids   []MEIAccid    `xml:"accid"`
	Supplied []MEISupplied `xml:"supplied"`
}

type MEIAccid struct {
	Accid    string `xml:"accid,attr"`
	AccidGes string `xml:"accid.ges,attr"`
	Reason   string `xml:"reason,attr"`
}

type MEISupplied struct {
	Reason string     `xml:"reason,attr"`
	Accids []MEIAccid `xml:"accid"`
}

type MEIRest struct {
	ID       string `xml:"id,attr"`
	Duration string `xml:"dur,attr"`
	DurPPQ   int    `xml:"dur.ppq,attr"`
	Dots     int    `xml:"dots,attr"`
}

type MEIChord struct {
	ID       string    `xml:"id,attr"`
	Duration string    `xml:"dur,attr"`
	DurPPQ   int       `xml:"dur.ppq,attr"`
	Dots     int       `xml:"dots,attr"`
	Notes    []MEINote `xml:"note"`
}

type MEIBeam struct {
	Notes  []MEINote  `xml:"note"`
	Rests  []MEIRest  `xml:"rest"`
	Chords []MEIChord `xml:"chord"`
}

type MEIVerse struct {
	N   string `xml:"n,attr"`
	Syl MEISyl `xml:"syl"`
}

type MEISyl struct {
	Con     string `xml:"con,attr"`
	WordPos string `xml:"wordpos,attr"`
	Value   string `xml:",chardata"`
}

func meiCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "iso-8859-1", "latin1":
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	case "windows-1252":
		return charmap.Windows1252.NewDecoder().Reader(input), nil
	case "utf-16", "utf-16le", "utf-16be":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Reader(input), nil
	default:
		return input, nil
	}
}

func LoadMEI(path string) (*MEI, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	r, err := makeUTF8Reader(f)
	if err != nil {
		return nil, fmt.Errorf("reader: %w", err)
	}

	dec := xml.NewDecoder(r)
	dec.CharsetReader = meiCharsetReader
	dec.Strict = false

	var mei MEI
	if err := dec.Decode(&mei); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return &mei, nil
}

func (m *MEI) ToScore() *Score {
	partwise := &ScorePartwise{
		Version:        m.Version,
		Work:           &Work{WorkTitle: m.Title()},
		Identification: &Identification{Creators: []TypedText{{Type: "composer", Value: m.Composer()}}},
	}

	staffDefs := m.StaffDefs()
	for _, staffDef := range staffDefs {
		partwise.PartList.ScoreParts = append(partwise.PartList.ScoreParts, ScorePart{
			ID:       staffDef.N,
			PartName: strings.TrimSpace(cmpNonEmpty(staffDef.Label, staffDef.LabelElement)),
		})
		partwise.Parts = append(partwise.Parts, Part{ID: staffDef.N})
	}

	if len(partwise.Parts) == 0 {
		for _, measure := range m.Measures() {
			for _, staff := range measure.Staffs {
				partwise.PartList.ScoreParts = append(partwise.PartList.ScoreParts, ScorePart{ID: staff.N})
				partwise.Parts = append(partwise.Parts, Part{ID: staff.N})
			}
			break
		}
	}

	partIndex := make(map[string]int, len(partwise.Parts))
	for i, part := range partwise.Parts {
		partIndex[part.ID] = i
	}

	currentScoreDef := m.scoreDef()
	currentStaffDefs := make(map[string]MEIStaffDef)
	for _, sd := range staffDefs {
		currentStaffDefs[sd.N] = sd
	}
	needsAttributes := make(map[string]bool)
	for n := range currentStaffDefs {
		needsAttributes[n] = true
	}

	var processElements func(elements []any)
	processElements = func(elements []any) {
		for _, el := range elements {
			switch e := el.(type) {
			case MEIScoreDef:
				currentScoreDef = e
				for _, sd := range collectStaffDefs(e.StaffGrps) {
					currentStaffDefs[sd.N] = sd
					needsAttributes[sd.N] = true
				}
				for n := range currentStaffDefs {
					needsAttributes[n] = true
				}
			case MEIStaffDef:
				currentStaffDefs[e.N] = e
				needsAttributes[e.N] = true
			case MEISection:
				processElements(e.Elements)
			case MEIMeasure:
				for _, staff := range e.Staffs {
					idx, ok := partIndex[staff.N]
					if !ok {
						continue
					}
					musicXMLMeasure := Measure{Number: e.N}
					if needsAttributes[staff.N] {
						musicXMLMeasure.Attributes = m.attributesFromState(staff.N, currentScoreDef, currentStaffDefs[staff.N])
						if musicXMLMeasure.Attributes != nil {
							musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, *musicXMLMeasure.Attributes)
						}
						needsAttributes[staff.N] = false
					}
					for _, layer := range staff.Layers {
						for _, element := range layer.Elements {
							switch el := element.(type) {
							case MEINote:
								musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, meiNoteToMusicXML(el, currentScoreDef.PPQ))
							case MEIRest:
								musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, meiRestToMusicXML(el, currentScoreDef.PPQ))
							case MEIChord:
								for noteIndex, note := range el.Notes {
									converted := meiNoteToMusicXML(note, currentScoreDef.PPQ)
									converted.Duration = meiDuration(el.DurPPQ, el.Duration, el.Dots, currentScoreDef.PPQ)
									if noteIndex > 0 {
										converted.Chord = &Empty{}
									}
									musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, converted)
								}
							case MEIBeam:
								for _, note := range el.Notes {
									musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, meiNoteToMusicXML(note, currentScoreDef.PPQ))
								}
								for _, rest := range el.Rests {
									converted := meiRestToMusicXML(rest, currentScoreDef.PPQ)
									converted.Duration = meiDuration(rest.DurPPQ, rest.Duration, rest.Dots, currentScoreDef.PPQ)
									musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, converted)
								}
								for _, chord := range el.Chords {
									for noteIndex, note := range chord.Notes {
										converted := meiNoteToMusicXML(note, currentScoreDef.PPQ)
										converted.Duration = meiDuration(chord.DurPPQ, chord.Duration, chord.Dots, currentScoreDef.PPQ)
										if noteIndex > 0 {
											converted.Chord = &Empty{}
										}
										musicXMLMeasure.Elements = append(musicXMLMeasure.Elements, converted)
									}
								}
							}
						}
					}
					partwise.Parts[idx].Measures = append(partwise.Parts[idx].Measures, musicXMLMeasure)
				}
			}
		}
	}

	for _, mdiv := range m.Music.Body.MDivs {
		processElements(mdiv.Score.Elements)
	}

	return &Score{Partwise: partwise}
}

func (m *MEI) attributesFromState(staffN string, sd MEIScoreDef, staffDef MEIStaffDef) *Attributes {
	attrs := &Attributes{Divisions: 24}

	meterCount := sd.MeterCount
	meterUnit := sd.MeterUnit
	if staffDef.MeterCount != "" {
		meterCount = staffDef.MeterCount
		meterUnit = staffDef.MeterUnit
	}
	if beats, err := strconv.Atoi(meterCount); err == nil {
		attrs.Time = []Time{{Beats: strconv.Itoa(beats), BeatType: meterUnit}}
	}

	keySig := sd.KeySig
	keyMode := sd.KeyMode
	if staffDef.KeySig != "" {
		keySig = staffDef.KeySig
		keyMode = staffDef.KeyMode
	}
	if keySig != "" {
		attrs.Key = []Key{{Fifths: meiKeySigToFifths(keySig), Mode: keyMode}}
	}

	clefShape := sd.ClefShape
	clefLine := sd.ClefLine
	clefDis := sd.ClefDis
	clefDisPlace := sd.ClefDisPlace
	if staffDef.ClefShape != "" {
		clefShape = staffDef.ClefShape
		clefLine = staffDef.ClefLine
		clefDis = staffDef.ClefDis
		clefDisPlace = staffDef.ClefDisPlace
	}
	if clefShape != "" {
		attrs.Clef = []Clef{{Sign: clefShape, Line: clefLine, ClefOctaveChange: meiClefOctaveChangeFromParts(clefDis, clefDisPlace)}}
	}

	return attrs
}

func (m *MEI) Title() string {
	if len(m.MEIHead.WorkList.Works) > 0 && strings.TrimSpace(m.MEIHead.WorkList.Works[0].Title) != "" {
		return strings.TrimSpace(m.MEIHead.WorkList.Works[0].Title)
	}
	return strings.TrimSpace(m.MEIHead.FileDesc.TitleStmt.Title)
}

func (m *MEI) Composer() string {
	if len(m.MEIHead.WorkList.Works) > 0 {
		for _, composer := range m.MEIHead.WorkList.Works[0].Composers {
			if composer.PersName != nil && strings.TrimSpace(composer.PersName.Value) != "" {
				return strings.TrimSpace(composer.PersName.Value)
			}
			if strings.TrimSpace(composer.Value) != "" {
				return strings.TrimSpace(composer.Value)
			}
		}
	}
	for _, respStmt := range m.MEIHead.FileDesc.TitleStmt.RespStmt {
		for _, persName := range respStmt.PersNames {
			if persName.Role == "composer" {
				return strings.TrimSpace(persName.Value)
			}
		}
	}
	return ""
}

func (m *MEI) Measures() []MEIMeasure {
	var measures []MEIMeasure
	for _, mdiv := range m.Music.Body.MDivs {
		for _, el := range mdiv.Score.Elements {
			if sec, ok := el.(MEISection); ok {
				measures = append(measures, collectMeasures(sec)...)
			}
		}
	}
	return measures
}

func collectMeasures(sec MEISection) []MEIMeasure {
	var measures []MEIMeasure
	for _, el := range sec.Elements {
		switch e := el.(type) {
		case MEIMeasure:
			measures = append(measures, e)
		case MEISection:
			measures = append(measures, collectMeasures(e)...)
		}
	}
	return measures
}

func (m *MEI) StaffDefs() []MEIStaffDef {
	return collectStaffDefs(m.scoreDef().StaffGrps)
}

func (m *MEI) scoreDef() MEIScoreDef {
	if len(m.Music.Body.MDivs) == 0 {
		return MEIScoreDef{}
	}
	for _, el := range m.Music.Body.MDivs[0].Score.Elements {
		if sd, ok := el.(MEIScoreDef); ok {
			return sd
		}
	}
	return MEIScoreDef{}
}

func collectStaffDefs(groups []MEIStaffGrp) []MEIStaffDef {
	var staffDefs []MEIStaffDef
	for _, group := range groups {
		staffDefs = append(staffDefs, group.StaffDefs...)
		staffDefs = append(staffDefs, collectStaffDefs(group.StaffGrps)...)
	}
	return staffDefs
}

func (m *MEI) AttributesForStaff(staffN string) *Attributes {
	attrs := &Attributes{Divisions: 24}
	sd := m.scoreDef()

	if beats, err := strconv.Atoi(sd.MeterCount); err == nil {
		attrs.Time = []Time{{Beats: strconv.Itoa(beats), BeatType: sd.MeterUnit}}
	}

	if sd.KeySig != "" {
		attrs.Key = []Key{{Fifths: meiKeySigToFifths(sd.KeySig), Mode: sd.KeyMode}}
	}
	if sd.ClefShape != "" {
		attrs.Clef = []Clef{{Sign: sd.ClefShape, Line: sd.ClefLine, ClefOctaveChange: meiClefOctaveChangeFromParts(sd.ClefDis, sd.ClefDisPlace)}}
	}

	for _, staffDef := range m.StaffDefs() {
		if staffDef.N != staffN {
			continue
		}
		if staffDef.MeterCount != "" {
			if beats, err := strconv.Atoi(staffDef.MeterCount); err == nil {
				attrs.Time = []Time{{Beats: strconv.Itoa(beats), BeatType: staffDef.MeterUnit}}
			}
		}
		if staffDef.KeySig != "" {
			attrs.Key = []Key{{Fifths: meiKeySigToFifths(staffDef.KeySig), Mode: staffDef.KeyMode}}
		}
		if staffDef.ClefShape != "" {
			attrs.Clef = []Clef{{Sign: staffDef.ClefShape, Line: staffDef.ClefLine, ClefOctaveChange: meiClefOctaveChangeFromParts(staffDef.ClefDis, staffDef.ClefDisPlace)}}
		}
		return attrs
	}

	return attrs
}

func meiNoteToMusicXML(note MEINote, ppq int) Note {
	step := strings.ToUpper(note.PName)
	alter := meiAccidentalToAlter(note.Accid)
	editorial := note.Reason == "edit"

	for _, accid := range note.Accids {
		alter = meiAccidentalToAlter(cmpNonEmpty(accid.Accid, accid.AccidGes))
		if accid.Reason == "edit" {
			editorial = true
		}
	}
	for _, supplied := range note.Supplied {
		for _, accid := range supplied.Accids {
			alter = meiAccidentalToAlter(cmpNonEmpty(accid.Accid, accid.AccidGes))
			editorial = true
		}
	}

	if note.Accid == "" && step == "B" {
		alter = -1
	}
	return Note{
		Pitch:               &Pitch{Step: step, Alter: alter, Octave: note.Octave},
		AccidentalEditorial: editorial,
		Duration:            meiDuration(note.DurPPQ, note.Duration, note.Dots, ppq),
		Type:                meiDurationType(note.Duration),
		Tie:                 meiTie(note.Tie),
	}
}

func meiRestToMusicXML(rest MEIRest, ppq int) Note {
	return Note{Rest: &Rest{}, Duration: meiDuration(rest.DurPPQ, rest.Duration, rest.Dots, ppq), Type: meiDurationType(rest.Duration)}
}

func meiDuration(durPPQ int, dur string, dots int, ppq int) int {
	if ppq == 0 {
		ppq = 256
	}
	var baseDur int
	if durPPQ > 0 {
		baseDur = durPPQ * 24 / ppq
		return baseDur
	}
	switch dur {
	case "maxima":
		baseDur = 24 * 32
	case "long":
		baseDur = 24 * 16
	case "breve":
		baseDur = 24 * 8
	case "1":
		baseDur = 24 * 4
	case "2":
		baseDur = 24 * 2
	case "4":
		baseDur = 24
	case "8":
		baseDur = 24 / 2
	case "16":
		baseDur = 24 / 4
	case "32":
		baseDur = 24 / 8
	default:
		baseDur = 24
	}

	res := baseDur
	for i := range dots {
		res += baseDur / (1 << (i + 1))
	}
	return res
}

func meiDurationType(dur string) string {
	switch dur {
	case "breve":
		return "breve"
	case "1":
		return "whole"
	case "2":
		return "half"
	case "4":
		return "quarter"
	case "8":
		return "eighth"
	case "16":
		return "16th"
	case "32":
		return "32nd"
	}
	return ""
}

func meiAccidentalToAlter(accid string) int {
	switch accid {
	case "s", "sharp":
		return 1
	case "f", "flat":
		return -1
	case "ss", "x":
		return 2
	case "ff":
		return -2
	}
	return 0
}

func meiTie(tie string) []Tie {
	switch tie {
	case "i":
		return []Tie{{Type: "start"}}
	case "m":
		return []Tie{{Type: "continue"}}
	case "t":
		return []Tie{{Type: "stop"}}
	}
	return nil
}

func meiKeySigToFifths(keySig string) int {
	if keySig == "" || keySig == "0" {
		return 0
	}
	last := keySig[len(keySig)-1]
	if last == 'f' || last == 's' {
		n, err := strconv.Atoi(keySig[:len(keySig)-1])
		if err != nil {
			return 0
		}
		if last == 'f' {
			return -n
		}
		return n
	}
	n, err := strconv.Atoi(keySig)
	if err == nil {
		return n
	}
	return 0
}

func meiClefOctaveChangeFromParts(dis int, place string) int {
	if dis == 8 && place == "below" {
		return -1
	}
	if dis == 8 && place == "above" {
		return 1
	}
	return 0
}

func cmpNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
