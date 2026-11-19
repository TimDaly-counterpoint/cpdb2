package score

import (
	"encoding/xml"
	"strconv"
)

// Score is a wrapper to unmarshal either score-partwise or score-timewise.
type Score struct {
	XMLName     xml.Name
	Partwise    *ScorePartwise `xml:"score-partwise"`
	Timewise    *ScoreTimewise `xml:"score-timewise"`
	RawInnerXML string         `xml:",innerxml"`
}

func (s *Score) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	s.XMLName = start.Name
	switch start.Name.Local {
	case "score-partwise":
		var pw ScorePartwise
		if err := d.DecodeElement(&pw, &start); err != nil {
			return err
		}
		s.Partwise = &pw
		s.Timewise = nil
		s.RawInnerXML = ""
		return nil
	case "score-timewise":
		var tw ScoreTimewise
		if err := d.DecodeElement(&tw, &start); err != nil {
			return err
		}
		s.Timewise = &tw
		s.Partwise = nil
		s.RawInnerXML = ""
		return nil
	default:
		var raw struct {
			RawInnerXML string `xml:",innerxml"`
		}
		if err := d.DecodeElement(&raw, &start); err != nil {
			return err
		}
		s.Partwise = nil
		s.Timewise = nil
		s.RawInnerXML = raw.RawInnerXML
		return nil
	}
}

type ScorePartwise struct {
	XMLName        xml.Name        `xml:"score-partwise"`
	Version        string          `xml:"version,attr"`
	Work           *Work           `xml:"work"`
	MovementNumber string          `xml:"movement-number,omitempty"`
	MovementTitle  string          `xml:"movement-title,omitempty"`
	Identification *Identification `xml:"identification"`
	Defaults       *Defaults       `xml:"defaults"`
	Credits        []Credit        `xml:"credit"`
	PartList       PartList        `xml:"part-list"`
	Parts          []Part          `xml:"part"`
}

type ScoreTimewise struct {
	XMLName        xml.Name          `xml:"score-timewise"`
	Version        string            `xml:"version,attr"`
	Work           *Work             `xml:"work"`
	MovementNumber string            `xml:"movement-number,omitempty"`
	MovementTitle  string            `xml:"movement-title,omitempty"`
	Identification *Identification   `xml:"identification"`
	Defaults       *Defaults         `xml:"defaults"`
	Credits        []Credit          `xml:"credit"`
	PartList       PartList          `xml:"part-list"`
	Measures       []TimewiseMeasure `xml:"measure"`
}

type TimewiseMeasure struct {
	XMLName        xml.Name       `xml:"measure"`
	Number         string         `xml:"number,attr,omitempty"`
	Width          float64        `xml:"width,attr,omitempty"`
	Implicit       string         `xml:"implicit,attr,omitempty"`
	NonControlling string         `xml:"non-controlling,attr,omitempty"`
	Parts          []TimewisePart `xml:"part"`
}

type TimewisePart struct {
	XMLName     xml.Name      `xml:"part"`
	ID          string        `xml:"id,attr,omitempty"`
	Attributes  *Attributes   `xml:"attributes"`
	Notes       []Note        `xml:"note"`
	Backup      []Backup      `xml:"backup"`
	Forward     []Forward     `xml:"forward"`
	Direction   []Direction   `xml:"direction"`
	Harmony     []Harmony     `xml:"harmony"`
	FiguredBass []FiguredBass `xml:"figured-bass"`
	Print       []Print       `xml:"print"`
	Barline     []Barline     `xml:"barline"`
	Sound       []Sound       `xml:"sound"`
	Elements    []interface{} `xml:"-"`
}

type Work struct {
	WorkNumber string `xml:"work-number,omitempty"`
	WorkTitle  string `xml:"work-title,omitempty"`
	Opus       *Opus  `xml:"opus,omitempty"`
}

type Opus struct {
	Href string `xml:"xlink:href,attr,omitempty"`
}

type Identification struct {
	Creators      []TypedText    `xml:"creator"`
	Rights        []TypedText    `xml:"rights"`
	Encoding      *Encoding      `xml:"encoding"`
	Source        string         `xml:"source,omitempty"`
	Relations     []TypedText    `xml:"relation"`
	Miscellaneous *Miscellaneous `xml:"miscellaneous"`
}

type TypedText struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type Encoding struct {
	EncodingDate     []string          `xml:"encoding-date"`
	Encoder          []TypedText       `xml:"encoder"`
	Software         []string          `xml:"software"`
	EncodingDesc     []string          `xml:"encoding-description"`
	Supports         []Supports        `xml:"supports"`
	ScoreInstruments []ScoreInstrument `xml:"score-instrument"`
}

type Supports struct {
	Type      string `xml:"type,attr"`
	Element   string `xml:"element,attr"`
	Attribute string `xml:"attribute,attr,omitempty"`
	Value     string `xml:"value,attr,omitempty"`
}

type Miscellaneous struct {
	Fields []MiscField `xml:"miscellaneous-field"`
}

type MiscField struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type Defaults struct {
	Scaling       *Scaling       `xml:"scaling"`
	PageLayout    *PageLayout    `xml:"page-layout"`
	SystemLayout  *SystemLayout  `xml:"system-layout"`
	StaffLayout   []StaffLayout  `xml:"staff-layout"`
	Appearance    *Appearance    `xml:"appearance"`
	MusicFont     *Font          `xml:"music-font"`
	WordFont      *Font          `xml:"word-font"`
	LyricFont     []Font         `xml:"lyric-font"`
	LyricLanguage []Lang         `xml:"lyric-language"`
}

type Scaling struct {
	Millimeters float64 `xml:"millimeters"`
	Tenths      float64 `xml:"tenths"`
}

type PageLayout struct {
	PageHeight  float64       `xml:"page-height"`
	PageWidth   float64       `xml:"page-width"`
	PageMargins []PageMargins `xml:"page-margins"`
}

type PageMargins struct {
	Type         string  `xml:"type,attr,omitempty"`
	LeftMargin   float64 `xml:"left-margin"`
	RightMargin  float64 `xml:"right-margin"`
	TopMargin    float64 `xml:"top-margin"`
	BottomMargin float64 `xml:"bottom-margin"`
}

type SystemLayout struct {
	SystemMargins     *SystemMargins   `xml:"system-margins"`
	SystemDistance    float64          `xml:"system-distance,omitempty"`
	TopSystemDistance float64          `xml:"top-system-distance,omitempty"`
	SystemDividers    *SystemDividers  `xml:"system-dividers"`
}

type SystemMargins struct {
	LeftMargin  float64 `xml:"left-margin"`
	RightMargin float64 `xml:"right-margin"`
}

type SystemDividers struct {
	LeftDivider  *EmptyPrint `xml:"left-divider"`
	RightDivider *EmptyPrint `xml:"right-divider"`
}

type StaffLayout struct {
	Number        int           `xml:"number,attr,omitempty"`
	StaffDistance float64       `xml:"staff-distance,omitempty"`
	StaffSize     float64       `xml:"staff-size,omitempty"`
	StaffTuning   []StaffTuning `xml:"staff-tuning"`
	StaffLines    int           `xml:"staff-lines,omitempty"`
}

type StaffTuning struct {
	Line         int     `xml:"line,attr,omitempty"`
	TuningStep   string  `xml:"tuning-step"`
	TuningAlter  float64 `xml:"tuning-alter,omitempty"`
	TuningOctave int     `xml:"tuning-octave"`
}

type Appearance struct {
	LineWidths      []LineWidth       `xml:"line-width"`
	NoteSizes       []NoteSize        `xml:"note-size"`
	Distance        []Distance        `xml:"distance"`
	OtherAppearance []OtherAppearance `xml:"other-appearance"`
}

type LineWidth struct {
	Type  string  `xml:"type,attr"`
	Value float64 `xml:",chardata"`
}

type NoteSize struct {
	Type  string  `xml:"type,attr,omitempty"`
	Value float64 `xml:",chardata"`
}

type Distance struct {
	Type  string  `xml:"type,attr"`
	Value float64 `xml:",chardata"`
}

type OtherAppearance struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type Font struct {
	FontFamily string `xml:"font-family,attr,omitempty"`
	FontStyle  string `xml:"font-style,attr,omitempty"`
	FontSize   string `xml:"font-size,attr,omitempty"`
	FontWeight string `xml:"font-weight,attr,omitempty"`
}

type Lang struct {
	Name string `xml:"name,attr,omitempty"`
	Lang string `xml:"xml:lang,attr,omitempty"`
}

type Credit struct {
	Page        int           `xml:"page,attr,omitempty"`
	CreditType  []string      `xml:"credit-type"`
	CreditWords []CreditWords `xml:"credit-words"`
	CreditImage *CreditImage  `xml:"credit-image"`
	Link        []Link        `xml:"link"`
	Bookmark    []Bookmark    `xml:"bookmark"`
}

type CreditWords struct {
	Value       string  `xml:",chardata"`
	DefaultX    float64 `xml:"default-x,attr,omitempty"`
	DefaultY    float64 `xml:"default-y,attr,omitempty"`
	Justify     string  `xml:"justify,attr,omitempty"`
	Halign      string  `xml:"halign,attr,omitempty"`
	Valign      string  `xml:"valign,attr,omitempty"`
	FontSize    string  `xml:"font-size,attr,omitempty"`
	FontWeight  string  `xml:"font-weight,attr,omitempty"`
	Underline   int     `xml:"underline,attr,omitempty"`
	Overline    int     `xml:"overline,attr,omitempty"`
	LineThrough int     `xml:"line-through,attr,omitempty"`
}

type CreditImage struct {
	Source string `xml:"source,attr"`
	Type   string `xml:"type,attr,omitempty"`
}

type Link struct {
	Href string `xml:"xlink:href,attr"`
}

type Bookmark struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr,omitempty"`
}

type PartList struct {
	PartGroups []PartGroup `xml:"part-group"`
	ScoreParts []ScorePart `xml:"score-part"`
}

type PartGroup struct {
	Type          string         `xml:"type,attr"`
	Number        string         `xml:"number,attr,omitempty"`
	GroupName     *GroupName     `xml:"group-name"`
	GroupSymbol   *GroupSymbol   `xml:"group-symbol"`
	GroupBarlines *GroupBarlines `xml:"group-barline"`
}

type GroupName struct {
	Value string `xml:",chardata"`
}

type GroupSymbol struct {
	Value string `xml:",chardata"`
}

type GroupBarlines struct {
	Value string `xml:",chardata"`
}

type ScorePart struct {
	ID               string            `xml:"id,attr"`
	PartName         string            `xml:"part-name"`
	PartNameDisplay  *NameDisplay      `xml:"part-name-display"`
	PartAbbreviation *string           `xml:"part-abbreviation"`
	PartAbbrDisplay  *NameDisplay      `xml:"part-abbreviation-display"`
	ScoreInstruments []ScoreInstrument `xml:"score-instrument"`
	MidiInstruments  []MidiInstrument  `xml:"midi-instrument"`
}

type NameDisplay struct {
	DisplayText    []DisplayText    `xml:"display-text"`
	AccidentalText []AccidentalText `xml:"accidental-text"`
}

type DisplayText struct {
	Value string `xml:",chardata"`
}

type AccidentalText struct {
	Value string `xml:",chardata"`
}

type DisplayStepOctave struct {
	Step   string `xml:"step"`
	Octave int    `xml:"octave"`
}

type ScoreInstrument struct {
	ID             string `xml:"id,attr"`
	InstrumentName string `xml:"instrument-name"`
	InstrumentAbbr string `xml:"instrument-abbreviation,omitempty"`
	InstrumentSound string `xml:"instrument-sound,omitempty"`
}

type MidiInstrument struct {
	ID          string  `xml:"id,attr"`
	MidiChannel int     `xml:"midi-channel,omitempty"`
	MidiName    string  `xml:"midi-name,omitempty"`
	MidiBank    int     `xml:"midi-bank,omitempty"`
	MidiProgram int     `xml:"midi-program,omitempty"`
	Unpitched   int     `xml:"unpitched,omitempty"`
	Volume      float64 `xml:"volume,omitempty"`
	Pan         float64 `xml:"pan,omitempty"`
	Elevation   float64 `xml:"elevation,omitempty"`
}

type Part struct {
	ID       string    `xml:"id,attr"`
	Measures []Measure `xml:"measure"`
}

type Measure struct {
	Number         string        `xml:"number,attr"`
	Implicit       string        `xml:"implicit,attr,omitempty"`
	NonControlling string        `xml:"non-controlling,attr,omitempty"`
	Width          float64       `xml:"width,attr,omitempty"`
	Attributes     *Attributes   `xml:"attributes"`
	Notes          []Note        `xml:"note"`
	Backup         []Backup      `xml:"backup"`
	Forward        []Forward     `xml:"forward"`
	Direction      []Direction   `xml:"direction"`
	Harmony        []Harmony     `xml:"harmony"`
	FiguredBass    []FiguredBass `xml:"figured-bass"`
	Print          []Print       `xml:"print"`
	Barline        []Barline     `xml:"barline"`
	Sound          []Sound       `xml:"sound"`
	Elements       []interface{} `xml:"-"`
}

func (m *Measure) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "number":
			m.Number = attr.Value
		case "implicit":
			m.Implicit = attr.Value
		case "non-controlling":
			m.NonControlling = attr.Value
		case "width":
			if val, err := strconv.ParseFloat(attr.Value, 64); err == nil {
				m.Width = val
			}
		}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch elem := tok.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "attributes":
				var a Attributes
				if err := d.DecodeElement(&a, &elem); err != nil {
					return err
				}
				m.Attributes = &a
				m.Elements = append(m.Elements, a)
			case "note":
				var n Note
				if err := d.DecodeElement(&n, &elem); err != nil {
					return err
				}
				m.Notes = append(m.Notes, n)
				m.Elements = append(m.Elements, n)
			case "backup":
				var b Backup
				if err := d.DecodeElement(&b, &elem); err != nil {
					return err
				}
				m.Backup = append(m.Backup, b)
				m.Elements = append(m.Elements, b)
			case "forward":
				var f Forward
				if err := d.DecodeElement(&f, &elem); err != nil {
					return err
				}
				m.Forward = append(m.Forward, f)
				m.Elements = append(m.Elements, f)
			case "direction":
				var dir Direction
				if err := d.DecodeElement(&dir, &elem); err != nil {
					return err
				}
				m.Direction = append(m.Direction, dir)
				m.Elements = append(m.Elements, dir)
			case "harmony":
				var h Harmony
				if err := d.DecodeElement(&h, &elem); err != nil {
					return err
				}
				m.Harmony = append(m.Harmony, h)
				m.Elements = append(m.Elements, h)
			case "barline":
				var bl Barline
				if err := d.DecodeElement(&bl, &elem); err != nil {
					return err
				}
				m.Barline = append(m.Barline, bl)
				m.Elements = append(m.Elements, bl)
			case "print":
				var p Print
				if err := d.DecodeElement(&p, &elem); err != nil {
					return err
				}
				m.Print = append(m.Print, p)
				m.Elements = append(m.Elements, p)
			case "sound":
				var s Sound
				if err := d.DecodeElement(&s, &elem); err != nil {
					return err
				}
				m.Sound = append(m.Sound, s)
				m.Elements = append(m.Elements, s)
			default:
				var dummy struct{}
				_ = d.DecodeElement(&dummy, &elem)
			}
		case xml.EndElement:
			if elem.Name == start.Name {
				return nil
			}
		}
	}
}

type Attributes struct {
	Divisions     int            `xml:"divisions,omitempty"`
	Key           []Key          `xml:"key"`
	Time          []Time         `xml:"time"`
	Staves        int            `xml:"staves,omitempty"`
	PartSymbol    *PartSym       `xml:"part-symbol"`
	Instruments   int            `xml:"instruments,omitempty"`
	Clef          []Clef         `xml:"clef"`
	StaffDetails  []StaffDetails `xml:"staff-details"`
	Transpose     []Transpose    `xml:"transpose"`
	Directive     []Directive    `xml:"directive"`
	MeasureStyles []MeasureStyle `xml:"measure-style"`
}

type Key struct {
	Number        int      `xml:"number,attr,omitempty"`
	Fifths        int      `xml:"fifths"`
	Mode          string   `xml:"mode,omitempty"`
	Cancel        *Cancel  `xml:"cancel"`
	KeyStep       []string `xml:"key-step"`
	KeyAlter      []int    `xml:"key-alter"`
	KeyAccidental []string `xml:"key-accidental"`
}

type Cancel struct {
	Fifths   int    `xml:",chardata"`
	Location string `xml:"location,attr,omitempty"`
}

type Time struct {
	Number      int    `xml:"number,attr,omitempty"`
	Symbol      string `xml:"symbol,attr,omitempty"`
	Beats       string `xml:"beats"`
	BeatType    string `xml:"beat-type"`
	SenzaMisura string `xml:"senza-misura,omitempty"`
}

type PartSym struct {
	Value       string `xml:",chardata"`
	TopStaff    int    `xml:"top-staff,attr,omitempty"`
	BottomStaff int    `xml:"bottom-staff,attr,omitempty"`
}

type Clef struct {
	Number           int    `xml:"number,attr,omitempty"`
	Sign             string `xml:"sign"`
	Line             int    `xml:"line,omitempty"`
	ClefOctaveChange int    `xml:"clef-octave-change,omitempty"`
}

type StaffDetails struct {
	Number      int           `xml:"number,attr,omitempty"`
	StaffType   string        `xml:"staff-type,omitempty"`
	StaffLines  int           `xml:"staff-lines,omitempty"`
	StaffTuning []StaffTuning `xml:"staff-tuning"`
	Capo        int           `xml:"capo,omitempty"`
	StaffSize   float64       `xml:"staff-size,omitempty"`
}

type Transpose struct {
	Number       int         `xml:"number,attr,omitempty"`
	Diatonic     int         `xml:"diatonic,omitempty"`
	Chromatic    int         `xml:"chromatic"`
	OctaveChange int         `xml:"octave-change,omitempty"`
	Double       *EmptyPrint `xml:"double"`
}

type Directive struct {
	Value string `xml:",chardata"`
}

type MeasureStyle struct {
	Number       int           `xml:"number,attr,omitempty"`
	MultipleRest *MultipleRest `xml:"multiple-rest"`
}

type MultipleRest struct {
	Value int `xml:",chardata"`
}

type Note struct {
	Pitch               *Pitch            `xml:"pitch"`
	Rest                *Rest             `xml:"rest"`
	Unpitched           *Unpitched        `xml:"unpitched"`
	Chord               *Empty            `xml:"chord"`
	Cue                 *Empty            `xml:"cue"`
	Grace               *Grace            `xml:"grace"`
	Duration            int               `xml:"duration,omitempty"`
	Tie                 []Tie             `xml:"tie"`
	Voice               string            `xml:"voice,omitempty"`
	Type                string            `xml:"type,omitempty"`
	Dot                 []Empty           `xml:"dot"`
	Accidental          string            `xml:"accidental,omitempty"`
	AccidentalEditorial bool              `xml:"-"`
	TimeModification    *TimeModification `xml:"time-modification"`
	Stem                string            `xml:"stem,omitempty"`
	NoteHead            string            `xml:"notehead,omitempty"`
	NoteHeadText        *NoteHeadText     `xml:"notehead-text"`
	Beam                []Beam            `xml:"beam"`
	Notations           []Notations       `xml:"notations"`
	Articulations       *Articulations    `xml:"articulations"`
	DimLevel            int               `xml:"dimlevel,omitempty"`
	Technical           *Technical        `xml:"technical"`
	Ornaments           *Ornaments        `xml:"ornaments"`
	Lyric               []Lyric           `xml:"lyric"`
	Staff               int               `xml:"staff,omitempty"`
	Instrument          *Instrument       `xml:"instrument"`
	DefaultX            float64           `xml:"default-x,attr,omitempty"`
	DefaultY            float64           `xml:"default-y,attr,omitempty"`
	RelativeX           float64           `xml:"relative-x,attr,omitempty"`
	RelativeY           float64           `xml:"relative-y,attr,omitempty"`
	PrintObject         string            `xml:"print-object,attr,omitempty"`
	InnerXML            string            `xml:",innerxml"`
}

type Pitch struct {
	Step   string `xml:"step"`
	Alter  int    `xml:"alter,omitempty"`
	Octave int    `xml:"octave"`
}

type Rest struct {
	Measure       string `xml:"measure,attr,omitempty"`
	DisplayStep   string `xml:"display-step,omitempty"`
	DisplayOctave int    `xml:"display-octave,omitempty"`
}

type Unpitched struct {
	DisplayStep   string `xml:"display-step,omitempty"`
	DisplayOctave int    `xml:"display-octave,omitempty"`
}

type Grace struct {
	StealTimePrevious  float64 `xml:"steal-time-previous,attr,omitempty"`
	StealTimeFollowing float64 `xml:"steal-time-following,attr,omitempty"`
	MakeTime           float64 `xml:"make-time,attr,omitempty"`
	Slash              string  `xml:"slash,attr,omitempty"`
}

type Empty struct{}

type Tie struct {
	Type string `xml:"type,attr"`
}

type TimeModification struct {
	ActualNotes int     `xml:"actual-notes"`
	NormalNotes int     `xml:"normal-notes"`
	NormalType  string  `xml:"normal-type,omitempty"`
	NormalDot   []Empty `xml:"normal-dot"`
}

type NoteHeadText struct {
	DisplayText       []DisplayText      `xml:"display-text"`
	DisplayStepOctave *DisplayStepOctave `xml:"display-step-octave"`
}

type Beam struct {
	Number int    `xml:"number,attr,omitempty"`
	Value  string `xml:",chardata"`
}

type Notations struct {
	XMLName        xml.Name         `xml:"notations"`
	Tied           []Tied           `xml:"tied"`
	Slur           []Slur           `xml:"slur"`
	Arpeggiate     []Arpeggiate     `xml:"arpeggiate"`
	Glissando      []Glissando      `xml:"glissando"`
	Slide          []Slide          `xml:"slide"`
	Technical      *Technical       `xml:"technical"`
	Articulations  *Articulations   `xml:"articulations"`
	Ornaments      *Ornaments       `xml:"ornaments"`
	Dynamics       *Dynamics        `xml:"dynamics"`
	Fermata        []Fermata        `xml:"fermata"`
	AccidentalMark []AccidentalMark `xml:"accidental-mark"`
	Tuplet         []Tuplet         `xml:"tuplet"`
	InnerXML       string           `xml:",innerxml"`
}

type Tuplet struct {
	Type        string `xml:"type,attr"`
	Number      int    `xml:"number,attr,omitempty"`
	Bracket     string `xml:"bracket,attr,omitempty"`
	ShowNumber  string `xml:"show-number,attr,omitempty"`
	LineShape   string `xml:"line-shape,attr,omitempty"`
	Placement   string `xml:"placement,attr,omitempty"`
	ActualNotes int    `xml:"actual-notes,omitempty"`
	NormalNotes int    `xml:"normal-notes,omitempty"`
	NormalType  string `xml:"normal-type,omitempty"`
}

type Tied struct {
	Type   string `xml:"type,attr"`
	Number int    `xml:"number,attr,omitempty"`
}

type Slur struct {
	Type      string `xml:"type,attr"`
	Number    int    `xml:"number,attr,omitempty"`
	Placement string `xml:"placement,attr,omitempty"`
}

type Arpeggiate struct {
	Number    int    `xml:"number,attr,omitempty"`
	Direction string `xml:"direction,attr,omitempty"`
}

type Glissando struct {
	Type     string `xml:"type,attr"`
	Number   int    `xml:"number,attr,omitempty"`
	LineType string `xml:"line-type,attr,omitempty"`
}

type Slide struct {
	Type   string `xml:"type,attr"`
	Number int    `xml:"number,attr,omitempty"`
}

type Technical struct {
	UpBow         *EmptyPrint `xml:"up-bow"`
	DownBow       *EmptyPrint `xml:"down-bow"`
	Harmonic      *Harmonic   `xml:"harmonic"`
	OpenString    *EmptyPrint `xml:"open-string"`
	ThumbPosition *EmptyPrint `xml:"thumb-position"`
	Fingering     *Fingering  `xml:"fingering"`
	Pluck         *Pluck      `xml:"pluck"`
	Fret          *Fret       `xml:"fret"`
	String        *StringNum  `xml:"string"`
}

type Harmonic struct {
	Natural    *EmptyPrint `xml:"natural"`
	Artificial *EmptyPrint `xml:"artificial"`
}

type Fingering struct {
	Value string `xml:",chardata"`
}

type Pluck struct {
	Value string `xml:",chardata"`
}

type Fret struct {
	Value int `xml:",chardata"`
}

type StringNum struct {
	Value int `xml:",chardata"`
}

type Articulations struct {
	Accent         *EmptyPrint   `xml:"accent"`
	StrongAccent   *StrongAccent `xml:"strong-accent"`
	Staccato       *EmptyPrint   `xml:"staccato"`
	Tenuto         *EmptyPrint   `xml:"tenuto"`
	DetachedLegato *EmptyPrint   `xml:"detached-legato"`
	Staccatissimo  *EmptyPrint   `xml:"staccatissimo"`
	Spiccato       *EmptyPrint   `xml:"spiccato"`
	Scoop          *EmptyPrint   `xml:"scoop"`
	Plop           *EmptyPrint   `xml:"plop"`
	Doit           *EmptyPrint   `xml:"doit"`
	Falloff        *EmptyPrint   `xml:"falloff"`
	BreathMark     *BreathMark   `xml:"breath-mark"`
	Caesura        *EmptyPrint   `xml:"caesura"`
	Stress         *EmptyPrint   `xml:"stress"`
	Unstress       *EmptyPrint   `xml:"unstress"`
}

type StrongAccent struct {
	Type string `xml:"type,attr,omitempty"`
}

type BreathMark struct {
	Value string `xml:",chardata"`
}

type Ornaments struct {
	TrillMark       *EmptyPrint `xml:"trill-mark"`
	Turn            *EmptyPrint `xml:"turn"`
	DelayedTurn     *EmptyPrint `xml:"delayed-turn"`
	InvertedTurn    *EmptyPrint `xml:"inverted-turn"`
	Mordent         *EmptyPrint `xml:"mordent"`
	InvertedMordent *EmptyPrint `xml:"inverted-mordent"`
	Tremolo         *Tremolo    `xml:"tremolo"`
}

type Tremolo struct {
	Type  string `xml:"type,attr,omitempty"`
	Value int    `xml:",chardata"`
}

type Dynamics struct {
	P             *Empty `xml:"p"`
	PP            *Empty `xml:"pp"`
	PPP           *Empty `xml:"ppp"`
	F             *Empty `xml:"f"`
	FF            *Empty `xml:"ff"`
	FFF           *Empty `xml:"fff"`
	MP            *Empty `xml:"mp"`
	MF            *Empty `xml:"mf"`
	FP            *Empty `xml:"fp"`
	FZ            *Empty `xml:"fz"`
	SF            *Empty `xml:"sf"`
	SFP           *Empty `xml:"sfp"`
	SFPP          *Empty `xml:"sfpp"`
	SFZ           *Empty `xml:"sfz"`
	OtherDynamics string `xml:"other-dynamics"`
}

type Fermata struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type AccidentalMark struct {
	Value string `xml:",chardata"`
}

type Lyric struct {
	Number       string      `xml:"number,attr,omitempty"`
	Name         string      `xml:"name,attr,omitempty"`
	Syllabic     string      `xml:"syllabic,omitempty"`
	Text         string      `xml:"text"`
	Elision      []string    `xml:"elision"`
	Extend       *EmptyPrint `xml:"extend"`
	EndLine      *Empty      `xml:"end-line"`
	EndParagraph *Empty      `xml:"end-paragraph"`
}

type Instrument struct {
	ID string `xml:"id,attr"`
}

type Backup struct {
	Duration int `xml:"duration"`
}

type Forward struct {
	Duration int    `xml:"duration"`
	Voice    string `xml:"voice,omitempty"`
	Staff    int    `xml:"staff,omitempty"`
}

type Direction struct {
	Placement     string          `xml:"placement,attr,omitempty"`
	DirectionType []DirectionType `xml:"direction-type"`
	Offset        *Offset         `xml:"offset"`
	Voice         string          `xml:"voice,omitempty"`
	Staff         int             `xml:"staff,omitempty"`
	Sound         *Sound          `xml:"sound"`
}

type DirectionType struct {
	Rehearsal      []Rehearsal      `xml:"rehearsal"`
	Segno          []EmptyPrint     `xml:"segno"`
	Coda           []EmptyPrint     `xml:"coda"`
	Words          []Words          `xml:"words"`
	Crescendo      *Wedge           `xml:"crescendo"`
	Decrescendo    *Wedge           `xml:"decrescendo"`
	Wedge          *Wedge           `xml:"wedge"`
	Dynamics       []Dynamics       `xml:"dynamics"`
	Dashes         *Dashes          `xml:"dashes"`
	Bracket        *Bracket         `xml:"bracket"`
	Pedal          *Pedal           `xml:"pedal"`
	Metronome      *Metronome       `xml:"metronome"`
	OctaveShift    *OctaveShift     `xml:"octave-shift"`
	OtherDirection []OtherDirection `xml:"other-direction"`
}

type Rehearsal struct {
	Value string `xml:",chardata"`
}

type Words struct {
	Value string `xml:",chardata"`
}

type Wedge struct {
	Type   string `xml:"type,attr"`
	Number int    `xml:"number,attr,omitempty"`
}

type Dashes struct {
	Type   string `xml:"type,attr"`
	Number int    `xml:"number,attr,omitempty"`
}

type Bracket struct {
	Type     string `xml:"type,attr"`
	Number   int    `xml:"number,attr,omitempty"`
	LineType string `xml:"line-type,attr,omitempty"`
}

type Pedal struct {
	Type string `xml:"type,attr"`
	Line string `xml:"line,attr,omitempty"`
}

type Metronome struct {
	BeatUnit    string  `xml:"beat-unit"`
	BeatUnitDot []Empty `xml:"beat-unit-dot"`
	PerMinute   string  `xml:"per-minute"`
}

type OctaveShift struct {
	Type   string `xml:"type,attr"`
	Number int    `xml:"number,attr,omitempty"`
	Size   int    `xml:"size,attr,omitempty"`
}

type OtherDirection struct {
	Value string `xml:",chardata"`
}

type Offset struct {
	Value float64 `xml:",chardata"`
	Sound string  `xml:"sound,attr,omitempty"`
}

type Harmony struct {
	Type      string    `xml:"type,attr,omitempty"`
	Root      Root      `xml:"root"`
	Kind      Kind      `xml:"kind"`
	Inversion *Inversion `xml:"inversion"`
	Bass      *Bass     `xml:"bass"`
	Degree    []Degree  `xml:"degree"`
	Offset    *Offset   `xml:"offset"`
	Staff     int       `xml:"staff,omitempty"`
}

type Root struct {
	RootStep  string  `xml:"root-step"`
	RootAlter float64 `xml:"root-alter,omitempty"`
}

type Kind struct {
	Value string `xml:",chardata"`
	Text  string `xml:"text,attr,omitempty"`
}

type Inversion struct {
	Value int `xml:",chardata"`
}

type Bass struct {
	BassStep  string  `xml:"bass-step"`
	BassAlter float64 `xml:"bass-alter,omitempty"`
}

type Degree struct {
	DegreeValue int     `xml:"degree-value"`
	DegreeAlter float64 `xml:"degree-alter"`
	DegreeType  string  `xml:"degree-type"`
}

type FiguredBass struct {
	Figure []Figure `xml:"figure"`
}

type Figure struct {
	Prefix       *Prefix       `xml:"prefix"`
	FigureNumber *FigureNumber `xml:"figure-number"`
	Suffix       *Suffix       `xml:"suffix"`
}

type Prefix struct {
	Value string `xml:",chardata"`
}

type FigureNumber struct {
	Value string `xml:",chardata"`
}

type Suffix struct {
	Value string `xml:",chardata"`
}

type Print struct {
	StaffSpacing float64 `xml:"staff-spacing,attr,omitempty"`
	NewSystem    string  `xml:"new-system,attr,omitempty"`
	NewPage      string  `xml:"new-page,attr,omitempty"`
	PageNumber   int     `xml:"page-number,attr,omitempty"`
}

type Barline struct {
	Location string    `xml:"location,attr,omitempty"`
	BarStyle string    `xml:"bar-style,omitempty"`
	Repeat   *Repeat   `xml:"repeat"`
	Ending   *Ending   `xml:"ending"`
	Fermata  []Fermata `xml:"fermata"`
}

type Repeat struct {
	Direction string `xml:"direction,attr"`
	Times     int    `xml:"times,attr,omitempty"`
}

type Ending struct {
	Number string `xml:"number,attr"`
	Type   string `xml:"type,attr"`
}

type Sound struct {
	Tempo     float64 `xml:"tempo,attr,omitempty"`
	Dynamics  float64 `xml:"dynamics,attr,omitempty"`
	Dacapo    string  `xml:"dacapo,attr,omitempty"`
	Segno     string  `xml:"segno,attr,omitempty"`
	Dalsegno  string  `xml:"dalsegno,attr,omitempty"`
	Coda      string  `xml:"coda,attr,omitempty"`
	Tocoda    string  `xml:"tocoda,attr,omitempty"`
	Divisions int     `xml:"divisions,attr,omitempty"`
}

type EmptyPrint struct{}
