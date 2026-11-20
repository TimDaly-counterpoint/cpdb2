# CPDB2: Counterpoint Database & Score Analysis Suite

`cpdb2` is a modular recreation of the CPDB (Counterpoint Database) score ingestion, interval pattern analysis, and dissonance processing suite in Go.



## Architecture & Modular Packages

The project separates the monolith into focused, reusable packages in `pkg/` and individual CLI tools in `cmd/`:

- **`pkg/prompt`**: Standardized interactive prompt helpers (strings, integers, comma-separated lists, booleans) with default value support.
- **`pkg/db`**: Database connection pool management, dynamic DSN resolution (`postgres://...` URL or database name `db`), note extraction queries, and batch insertions.
- **`pkg/music`**: Counterpoint matrices, Tinctoris tables, interval mappings, MIDI, Voces, and Sonority tables.
- **`pkg/score`**: MusicXML, MEI (Music Encoding Initiative), and LilyPond (`.ly`) parsers with automatic UTF-8 transcoding, score normalization, and note extraction.
- **`pkg/patterns`**: Suffix/substring pattern searching, interval sequences (direct, inverted, retrograde, retrograde-inverted), maximal sequence extraction, and cross-piece sequence search.
- **`pkg/cmatrix`**: Melodic interval calculations, voice difference detection, aligned interval matrices, and CSV export.
- **`pkg/disstable`**: Dissonance table recreation from database views with bar mapping.
- **`pkg/tritones`**: Identification of tritone occurrences in dissonant sonorities.
- **`pkg/smooth`**: Resolution and smoothing of dissonance passages.
- **`pkg/exportxml`**: Sequence results parser and MusicXML 4.0 score generator.
- **`pkg/sonority`**: Recreation and population of `sonority`, `dissonance`, and smoothed `consonant` tables across all pieces.
- **`pkg/setdb`**: Database setup tool that validates empty databases, imports database schema from `template.sql`, and populates the `voces` table from `voces.sql`.
- **`pkg/installtools`**: Tool compiler and asset staging utility that builds all CLI tools into the `tools/` folder alongside necessary SQL and documentation assets.

## Standalone CLI Programs

Each category from `main.go` is an individual program in `cmd/`. Each program can be invoked in the format:

```bash
program_name db
```

Where `db` is the database name (e.g. `bcz`) or PostgreSQL connection URL. If omitted, the program interactively prompts for the database name with fallback to environment variables (`DATABASE_URL`, `PGDATABASE`, `DATABASE_USER`, `DATABASE_PASSWORD`, `DATABASE_HOST`, `DATABASE_PORT`).

Each program then interactively prompts for any additional required parameters:

| Program | Command Example | Prompts / Description |
| :--- | :--- | :--- |
| **`import_scores`** | `go run ./cmd/import_scores bcz` | Prompts for input score files / glob patterns (`.musicxml`, `.xml`, `.mei`, `.ly`) and imports them into the database. |
| **`search_sequences`** | `go run ./cmd/search_sequences bcz` | Prompts for piece IDs, voice numbers, min sequence length, min occurrences, exclusion options, and output filename; finds all repeated patterns across the entire corpus. |
| **`search_cross_piece`** | `go run ./cmd/search_cross_piece bcz` | Prompts for source piece ID, target piece IDs, voice numbers, min length, and output filename; finds occurrences of source sequences in other pieces. Looks for sequences from souce piece in one or more other pieces from the database|
| **`import_and_search`** | `go run ./cmd/import_and_search bcz` | Combined workflow: prompts for score files to import and immediately launches sequence search. Check - needs updating or removal|
| **`cmatrix`** | `go run ./cmd/cmatrix bcz` | Prompts for piece ID, voice numbers, and CSV filename; detects voice differences and generates interval matrix CSV.  |
| **`disstable`** | `go run ./cmd/disstable bcz` | Prompts for source view and target table names; recreates `disstable` with bar calculation. This is away of avoiding the very slow sonority_view and sonority_dissonance_view. To be replaced by a utility function to recreate these as tables|
| **`find_tritones`** | `go run ./cmd/find_tritones bcz` | Prompts for piece identity; lists offsets of tritones within dissonant sonorities. |
| **`smoothcp`** | `go run ./cmd/smoothcp bcz` | Prompts for piece identity; smooths dissonance passages and saves to `smoothcp_<piece>`. Amend to create table, dump table to file and drop table |
| **`export_smooth_xml`** | `go run ./cmd/export_smooth_xml bcz` | Prompts for piece ID and output filename; exports smoothed consonant sonorities to MusicXML 4.0. |
| **`export_sequences`** | `go run ./cmd/export_sequences bcz` | Prompts for sequence result text file and output directory; generates multi-voice MusicXML 4.0 scores. |
| **`set_db`** | `go run ./cmd/set_db dbname` | Validates that `dbname` exists and is empty, imports `template.sql`, and populates `voces` from `voces.sql`.  Expects a running postgresql database with the named database created but empty, along with the database user 'goclient' and password specified in /pkg/db.go. Parameters can be edited there|
| **`install_tools`** | `go run ./cmd/install_tools` | Compiles all tools into the `tools/` directory and copies `template.sql`, `voces.sql`, and `README.md`. Tools can then be used directly from the command line. |
| **`process_sonority`** | `go run ./cmd/process_sonority bcz` | Recreates and populates `sonority`, `dissonance`, and `consonant` tables for all pieces in the database. |
| **`cpdb2`** | `go run ./cmd/cpdb2 bcz` | Master interactive CLI menu providing access to all 12 functions. |

## Running Tests

To run the full unit test suite:

```bash
cd cpdb2
go test -v ./...
```
