package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NoteEntry represents a single note or rest row in the database.
type NoteEntry struct {
	Seq         int
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
	CreatedAt   string
}

// SequenceNote holds note data retrieved for sequence export or analysis.
type SequenceNote struct {
	Midi     int
	Bar      int
	Index    int
	Pitch    string
	Toffset  float64
	Duration float64
	Voice    int
}

// PieceInfo represents metadata for a piece in the database.
type PieceInfo struct {
	Seq       int
	Composer  string
	Title     string
	Voices    int
	CreatedAt string
}

// DB wraps a PostgreSQL connection pool.
type DB struct {
	Pool *pgxpool.Pool
	DSN  string
}

// ResolveConnString turns a db argument (name or URL) into a full PostgreSQL connection string.
func ResolveConnString(dbArg string) string {
	trimmed := strings.TrimSpace(dbArg)

	if trimmed == "" {
		if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
			return envURL
		}
		if envDB := os.Getenv("PGDATABASE"); envDB != "" {
			trimmed = envDB
		} else {
			trimmed = "bcz"
		}
	}

	if strings.HasPrefix(trimmed, "postgres://") || strings.HasPrefix(trimmed, "postgresql://") || strings.Contains(trimmed, "host=") {
		return trimmed
	}

	user := os.Getenv("DATABASE_USER")
	if user == "" {
		user = os.Getenv("PGUSER")
	}
	if user == "" {
		user = "goclient"
	}

	pass := os.Getenv("DATABASE_PASSWORD")
	if pass == "" {
		pass = os.Getenv("PGPASSWORD")
	}
	if pass == "" {
		pass = "gopherz"
	}

	host := os.Getenv("DATABASE_HOST")
	if host == "" {
		host = os.Getenv("PGHOST")
	}
	if host == "" {
		host = "127.0.0.1"
	}

	port := os.Getenv("DATABASE_PORT")
	if port == "" {
		port = os.Getenv("PGPORT")
	}
	if port == "" {
		port = "5432"
	}

	// Escape user and password if necessary
	u := url.UserPassword(user, pass)
	return fmt.Sprintf("postgres://%s@%s:%s/%s", u.String(), host, port, trimmed)
}

// NewDB creates and connects a PostgreSQL connection pool using dbArg.
func NewDB(ctx context.Context, dbArg string) (*DB, error) {
	connStr := ResolveConnString(dbArg)
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{
		Pool: pool,
		DSN:  connStr,
	}, nil
}

// Close closes the database pool.
func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}

// Ping checks the database connection.
func (d *DB) Ping(ctx context.Context) error {
	if d == nil || d.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}
	return d.Pool.Ping(ctx)
}

// GetPieceVoices returns the number of voices for a piece.
func (d *DB) GetPieceVoices(ctx context.Context, pieceSeq int) (int, error) {
	var voices int
	err := d.Pool.QueryRow(ctx, "SELECT voices FROM pieces WHERE seq = $1", pieceSeq).Scan(&voices)
	if err != nil {
		return 0, fmt.Errorf("query piece %d voices: %w", pieceSeq, err)
	}
	return voices, nil
}

// FetchNotesForVoices fetches note records for a piece and specified voices.
func (d *DB) FetchNotesForVoices(ctx context.Context, pieceSeq int, voices []int) (map[int][]SequenceNote, error) {
	voiceNotes := make(map[int][]SequenceNote)
	for _, v := range voices {
		rows, err := d.Pool.Query(ctx,
			"SELECT midi, bar, index, pitch, toffset, duration, voice FROM notes WHERE piece = $1 AND voice = $2 ORDER BY toffset, index",
			pieceSeq, v)
		if err != nil {
			return nil, fmt.Errorf("query notes for piece %d voice %d: %w", pieceSeq, v, err)
		}
		defer rows.Close()

		var notes []SequenceNote
		for rows.Next() {
			var n SequenceNote
			if err := rows.Scan(&n.Midi, &n.Bar, &n.Index, &n.Pitch, &n.Toffset, &n.Duration, &n.Voice); err != nil {
				return nil, fmt.Errorf("scan note: %w", err)
			}
			notes = append(notes, n)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		voiceNotes[v] = notes
	}
	return voiceNotes, nil
}

// FetchAllPieceNotes fetches all notes for a piece, ordered by index.
func (d *DB) FetchAllPieceNotes(ctx context.Context, pieceSeq int) ([]SequenceNote, error) {
	rows, err := d.Pool.Query(ctx,
		"SELECT midi, bar, index, pitch, toffset, duration, voice FROM notes WHERE piece = $1 ORDER BY voice, index",
		pieceSeq)
	if err != nil {
		return nil, fmt.Errorf("query notes for piece %d: %w", pieceSeq, err)
	}
	defer rows.Close()

	var notes []SequenceNote
	for rows.Next() {
		var n SequenceNote
		if err := rows.Scan(&n.Midi, &n.Bar, &n.Index, &n.Pitch, &n.Toffset, &n.Duration, &n.Voice); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// FetchNotesForPieceVoice fetches notes for a single piece and voice.
func (d *DB) FetchNotesForPieceVoice(ctx context.Context, pieceSeq, voice int) ([]SequenceNote, error) {
	rows, err := d.Pool.Query(ctx,
		"SELECT midi, bar, index, pitch, toffset, duration, voice FROM notes WHERE piece = $1 AND voice = $2 ORDER BY index",
		pieceSeq, voice)
	if err != nil {
		return nil, fmt.Errorf("query notes for piece %d voice %d: %w", pieceSeq, voice, err)
	}
	defer rows.Close()

	var notes []SequenceNote
	for rows.Next() {
		var n SequenceNote
		if err := rows.Scan(&n.Midi, &n.Bar, &n.Index, &n.Pitch, &n.Toffset, &n.Duration, &n.Voice); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// InsertPiece inserts a piece and returns its seq id.
func (d *DB) InsertPiece(ctx context.Context, composer, title string, voices int) (int, error) {
	var pieceSeq int
	err := d.Pool.QueryRow(ctx,
		"INSERT INTO pieces (composer, title, voices) VALUES ($1, $2, $3) RETURNING seq",
		composer, title, voices).Scan(&pieceSeq)
	if err != nil {
		return 0, fmt.Errorf("insert piece: %w", err)
	}
	return pieceSeq, nil
}

// InsertStaff inserts a staff configuration and returns its seq id.
func (d *DB) InsertStaff(ctx context.Context, key, num, denom *int, clef *string) (int, error) {
	var staffSeq int
	err := d.Pool.QueryRow(ctx,
		"INSERT INTO staves (key, num, denom, clef) VALUES ($1, $2, $3, $4) RETURNING seq",
		key, num, denom, clef).Scan(&staffSeq)
	if err != nil {
		return 0, fmt.Errorf("insert staff: %w", err)
	}
	return staffSeq, nil
}

// CopyNotes performs batch insertion of notes using COPY protocol.
func (d *DB) CopyNotes(ctx context.Context, notes [][]any) (int64, error) {
	if len(notes) == 0 {
		return 0, nil
	}
	return d.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"notes"},
		[]string{"piece", "mvt", "staffchange", "voice", "index", "pitch", "duration", "midi", "inflec", "ficta", "colour", "bar", "islast", "toffset", "dimlevel"},
		pgx.CopyFromRows(notes),
	)
}

// ListPieceIDs returns all piece sequence IDs currently in the database.
func (d *DB) ListPieceIDs(ctx context.Context) ([]int, error) {
	rows, err := d.Pool.Query(ctx, "SELECT seq FROM pieces ORDER BY seq")
	if err != nil {
		return nil, fmt.Errorf("query piece ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
