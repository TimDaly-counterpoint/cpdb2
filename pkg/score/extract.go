package score

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"cpdb2/pkg/db"
	"cpdb2/pkg/music"
	"github.com/jackc/pgx/v5"
)

var ErrDuplicatePiece = errors.New("duplicate piece")

// NoteValue holds extracted note data ready for database insertion.
type NoteValue struct {
	Piece       int
	Mvt         int
	StaffChange int
	Voice       int
	Pitch       string
	Duration    float64
	Midi        int
	Toffset     float64
	Index       int
	Bar         int
	DimLevel    int
	Inflec      int
	Ficta       int
	Colour      bool
	IsLast      int
}

// LoadScore loads and parses a score from file based on its extension.
func LoadScore(path string) (*Score, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".musicxml", ".xml":
		return LoadMusicXML(path)
	case ".mei":
		mei, err := LoadMEI(path)
		if err != nil {
			return nil, err
		}
		return mei.ToScore(), nil
	case ".ly":
		return LoadLilyPond(path)
	default:
		return nil, fmt.Errorf("unsupported score format %q", ext)
	}
}

// NoteDurationFromType converts a note type string (e.g., "quarter", "half") to duration units.
func NoteDurationFromType(noteType string, dots int) float64 {
	var base float64
	switch strings.ToLower(noteType) {
	case "maxima":
		base = 768.0
	case "long", "longa":
		base = 384.0
	case "breve":
		base = 192.0
	case "whole":
		base = 96.0
	case "half":
		base = 48.0
	case "quarter":
		base = 24.0
	case "eighth":
		base = 12.0
	case "16th":
		base = 6.0
	case "32nd":
		base = 3.0
	case "64th":
		base = 1.5
	case "128th":
		base = 0.75
	case "256th":
		base = 0.375
	default:
		return 0.0
	}
	res := base
	for i := 0; i < dots; i++ {
		base /= 2.0
		res += base
	}
	return res
}

// FindFilesByPatterns searches for files matching glob patterns or directories.
func FindFilesByPatterns(patterns ...string) ([]string, error) {
	var files []string
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			// Check if p itself is a file or directory
			matches = append(matches, p)
		}
		for _, m := range matches {
			ext := strings.ToLower(filepath.Ext(m))
			if ext == ".musicxml" || ext == ".xml" || ext == ".mei" || ext == ".ly" {
				files = append(files, m)
			}
		}
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}

// ExtractNotes parses the score and creates note values, optionally inserting piece and staves into the database.
func ExtractNotes(ctx context.Context, score *Score, database *db.DB) ([]NoteValue, error) {
	var entries []NoteValue
	var piece int

	title := ""
	if score.Partwise != nil {
		if score.Partwise.Work != nil {
			title = score.Partwise.Work.WorkTitle
		}
		if title == "" {
			title = score.Partwise.MovementTitle
		}
	} else if score.Timewise != nil {
		if score.Timewise.Work != nil {
			title = score.Timewise.Work.WorkTitle
		}
		if title == "" {
			title = score.Timewise.MovementTitle
		}
	}

	composer := ""
	var ident *Identification
	if score.Partwise != nil {
		ident = score.Partwise.Identification
	} else if score.Timewise != nil {
		ident = score.Timewise.Identification
	}

	if ident != nil && len(ident.Creators) > 0 {
		composer = ident.Creators[0].Value
	}

	voices := 0
	var parts []Part
	if score.Partwise != nil {
		voices = len(score.Partwise.Parts)
		parts = score.Partwise.Parts
	} else if score.Timewise != nil {
		if len(score.Timewise.Measures) > 0 {
			voices = len(score.Timewise.Measures[0].Parts)
		}
	}

	if database != nil && database.Pool != nil {
		err := database.Pool.QueryRow(ctx, "SELECT seq FROM pieces WHERE title = $1 AND composer = $2", title, composer).Scan(&piece)
		if err == nil {
			return nil, ErrDuplicatePiece
		} else if errors.Is(err, pgx.ErrNoRows) {
			err = database.Pool.QueryRow(ctx, "INSERT INTO pieces (title, composer, voices) VALUES ($1, $2, $3) RETURNING seq", title, composer, voices).Scan(&piece)
			if err != nil {
				return nil, fmt.Errorf("error inserting piece: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error querying piece: %w", err)
		}
	}

	for voice, part := range parts {
		noteIndex := 1
		tDur := 0.0
		prevOffset := 0.0
		durCarryMap := make(map[string]int)
		barCarryMap := make(map[string]int)
		durMult := 1.0
		var (
			currentKey   *int
			currentNum   *int
			currentDenom *int
			currentClef  *string
			staffType    int
		)

		for meas, measure := range part.Measures {
			for _, element := range measure.Elements {
				var noteValues NoteValue
				noteValues.Piece = piece
				noteValues.Mvt = 1
				noteValues.StaffChange = staffType
				noteValues.Voice = voice + 1

				switch el := element.(type) {
				case Attributes:
					if el.Divisions != 0 {
						durMult = 24.0 / float64(el.Divisions)
					}

					if len(el.Key) > 0 {
						k := el.Key[0].Fifths
						currentKey = &k
					}
					if len(el.Time) > 0 {
						if n, err := strconv.Atoi(el.Time[0].Beats); err == nil {
							currentNum = &n
						}
						if d, err := strconv.Atoi(el.Time[0].BeatType); err == nil {
							currentDenom = &d
						}
					}
					if len(el.Clef) > 0 {
						c := el.Clef[0].Sign + strconv.Itoa(el.Clef[0].Line)
						currentClef = &c
					}

					if database != nil && database.Pool != nil {
						query := "SELECT seq FROM staves WHERE true"
						var args []interface{}
						argIdx := 1

						if currentKey != nil {
							query += fmt.Sprintf(" AND key = $%d", argIdx)
							args = append(args, *currentKey)
							argIdx++
						} else {
							query += " AND key IS NULL"
						}

						if currentNum != nil {
							query += fmt.Sprintf(" AND num = $%d", argIdx)
							args = append(args, *currentNum)
							argIdx++
						} else {
							query += " AND num IS NULL"
						}

						if currentDenom != nil {
							query += fmt.Sprintf(" AND denom = $%d", argIdx)
							args = append(args, *currentDenom)
							argIdx++
						} else {
							query += " AND denom IS NULL"
						}

						if currentClef != nil {
							query += fmt.Sprintf(" AND clef = $%d", argIdx)
							args = append(args, *currentClef)
							argIdx++
						} else {
							query += " AND clef IS NULL"
						}

						err := database.Pool.QueryRow(ctx, query, args...).Scan(&staffType)
						if err != nil {
							if errors.Is(err, pgx.ErrNoRows) {
								var insertCols []string
								var insertValues []string
								var insertArgs []interface{}
								insIdx := 1

								if currentKey != nil {
									insertCols = append(insertCols, "key")
									insertValues = append(insertValues, fmt.Sprintf("$%d", insIdx))
									insertArgs = append(insertArgs, *currentKey)
									insIdx++
								}
								if currentNum != nil {
									insertCols = append(insertCols, "num")
									insertValues = append(insertValues, fmt.Sprintf("$%d", insIdx))
									insertArgs = append(insertArgs, *currentNum)
									insIdx++
								}
								if currentDenom != nil {
									insertCols = append(insertCols, "denom")
									insertValues = append(insertValues, fmt.Sprintf("$%d", insIdx))
									insertArgs = append(insertArgs, *currentDenom)
									insIdx++
								}
								if currentClef != nil {
									insertCols = append(insertCols, "clef")
									insertValues = append(insertValues, fmt.Sprintf("$%d", insIdx))
									insertArgs = append(insertArgs, *currentClef)
									insIdx++
								}

								if len(insertCols) > 0 {
									insertQuery := fmt.Sprintf("INSERT INTO staves (%s) VALUES (%s) RETURNING seq",
										strings.Join(insertCols, ", "),
										strings.Join(insertValues, ", "))
									err = database.Pool.QueryRow(ctx, insertQuery, insertArgs...).Scan(&staffType)
									if err != nil {
										log.Printf("Error inserting into staves: %v", err)
									}
								}
							}
						}
					}

				case Note:
					note := el
					pitchKey := "0"
					if note.Rest == nil && note.Pitch != nil {
						pitchKey = fmt.Sprintf("%s%d_%d", note.Pitch.Step, note.Pitch.Octave, note.Pitch.Alter)
						if _, exists := durCarryMap[pitchKey]; !exists {
							prefix := fmt.Sprintf("%s%d_", note.Pitch.Step, note.Pitch.Octave)
							for k := range durCarryMap {
								if strings.HasPrefix(k, prefix) {
									pitchKey = k
									break
								}
							}
						}
					}

					if barCarryMap[pitchKey] == 0 {
						barCarryMap[pitchKey] = meas + 1
					}
					if note.Tie != nil {
						q := len(note.Tie) - 1
						if note.Tie[q].Type != "stop" {
							if note.Duration > 0 {
								durCarryMap[pitchKey] += note.Duration
							} else if note.Type != "" {
								equiv := int(NoteDurationFromType(note.Type, len(note.Dot)) / durMult)
								durCarryMap[pitchKey] += equiv
							}
							continue
						}
					}

					durVal := note.Duration + durCarryMap[pitchKey]
					var durFloat float64
					if durVal > 0 {
						durFloat = float64(durVal) * durMult
					} else if note.Type != "" {
						durFloat = NoteDurationFromType(note.Type, len(note.Dot))
					}
					noteValues.Duration = durFloat

					if note.Chord != nil {
						noteValues.Toffset = prevOffset
					} else {
						prevOffset = tDur
						noteValues.Toffset = tDur
						tDur += durFloat
					}

					if note.Rest != nil {
						noteValues.Pitch = "0"
					} else if note.Pitch != nil {
						noteValues.Pitch = note.Pitch.Step + strconv.Itoa(note.Pitch.Octave)
					}

					noteValues.Bar = barCarryMap[pitchKey]
					delete(durCarryMap, pitchKey)
					delete(barCarryMap, pitchKey)
					noteValues.Midi = music.Midi[noteValues.Pitch]

					if note.Rest == nil && note.Pitch != nil && note.Pitch.Alter != 0 {
						if !note.AccidentalEditorial {
							noteValues.Inflec = note.Pitch.Alter
							noteValues.Midi += note.Pitch.Alter
						} else {
							noteValues.Ficta = note.Pitch.Alter
						}
					}

					noteValues.Index = noteIndex
					noteValues.Colour = false
					noteValues.IsLast = 0
					noteValues.StaffChange = staffType

					entries = append(entries, noteValues)
					noteIndex++

				case Barline:
					if len(entries) > 0 {
						entries[len(entries)-1].IsLast = 1
					}
				}
			}
		}
	}

	return entries, nil
}

// ImportScores reads matching score files and inserts them into the database.
func ImportScores(ctx context.Context, database *db.DB, patterns ...string) (int, error) {
	files, err := FindFilesByPatterns(patterns...)
	if err != nil {
		return 0, fmt.Errorf("find files: %w", err)
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no score files found matching patterns: %v", patterns)
	}

	importedCount := 0
	for _, f := range files {
		score, err := LoadScore(f)
		if err != nil {
			log.Printf("Failed to load score %s: %v", f, err)
			continue
		}

		notes, err := ExtractNotes(ctx, score, database)
		if err != nil {
			if errors.Is(err, ErrDuplicatePiece) {
				log.Printf("Skipping duplicate piece in %s", f)
				continue
			}
			log.Printf("Failed to extract notes for %s: %v", f, err)
			continue
		}

		if len(notes) > 0 && database != nil {
			var rows [][]any
			for _, n := range notes {
				var staffChange any
				if n.StaffChange > 0 {
					staffChange = n.StaffChange
				}
				mvt := n.Mvt
				if mvt == 0 {
					mvt = 1
				}
				rows = append(rows, []any{
					n.Piece,
					mvt,
					staffChange,
					n.Voice,
					n.Index,
					n.Pitch,
					int(math.Round(n.Duration)),
					n.Midi,
					n.Inflec,
					n.Ficta,
					n.Colour,
					n.Bar,
					n.IsLast,
					int(math.Round(n.Toffset)),
					n.DimLevel,
				})
			}

			_, err = database.CopyNotes(ctx, rows)
			if err != nil {
				log.Printf("Failed to insert notes for %s: %v", f, err)
				continue
			}
		}

		importedCount++
		fmt.Printf("Successfully imported %s (%d notes)\n", f, len(notes))
	}

	return importedCount, nil
}
