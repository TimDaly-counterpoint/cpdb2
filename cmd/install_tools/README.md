# `install_tools`

`install_tools` is the build and deployment utility for the CPDB2 suite.

## Overview

`install_tools` automates the compilation and staging of all standalone command-line tools in the project into a dedicated `tools/` directory.

When run, the program:
1. Creates the `tools/` folder in the project root (if it does not already exist).
2. Discovers and compiles all standalone CLI tools in `cmd/` (excluding `install_tools` itself) into individual executable binaries placed in `tools/`.
3. Copies essential SQL template and seed files (`template.sql` and `voces.sql`) into `tools/` for local standalone database configuration.
4. Copies the updated project `README.md` into `tools/`.

## Usage

Execute the tool from the repository root:

```bash
go run ./cmd/install_tools
```

Once installed, all binaries in the `tools/` folder can be executed directly as standalone programs:

```bash
./tools/set_db dbname
./tools/import_scores dbname
./tools/cpdb2 dbname
```

## Installed Tools

The following binaries are built and placed in `tools/`:

- **`cmatrix`**: Melodic interval calculations, voice difference detection, and CSV generation.
- **`cpdb2`**: Master interactive CLI menu for the full suite.
- **`disstable`**: Dissonance table recreation from database views with bar mapping.
- **`export_sequences`**: Sequence results export to multi-voice MusicXML 4.0.
- **`export_smooth_xml`**: Smoothed dissonance table export to MusicXML 4.0.
- **`find_tritones`**: Identification of tritone occurrences in dissonant sonorities.
- **`import_and_search`**: Combined workflow for score ingestion and immediate sequence search.
- **`import_scores`**: Ingestion of MusicXML, MEI, and LilyPond score files.
- **`process_sonority`**: Recreates and populates `sonority`, `dissonance`, and `consonant` tables for all pieces in the database.
- **`search_cross_piece`**: Cross-piece melodic pattern and sequence search.
- **`search_sequences`**: Corpus-wide repeated melodic sequence search.
- **`set_db`**: Automated empty database setup, schema import (`template.sql`), and `voces` and `intervala` table population (`voces.sql`).
- **`smoothcp`**: Resolution and smoothing of dissonance passages.
