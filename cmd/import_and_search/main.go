package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cpdb2/pkg/db"
	"cpdb2/pkg/patterns"
	"cpdb2/pkg/prompt"
	"cpdb2/pkg/score"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	dbArg := ""
	if len(os.Args) > 1 {
		dbArg = strings.TrimSpace(os.Args[1])
	}
	if dbArg == "" {
		var err error
		dbArg, err = prompt.PromptString(reader, "Enter database name or connection string", "bcz")
		if err != nil {
			log.Fatalf("Error reading database argument: %v", err)
		}
	}

	ctx := context.Background()
	database, err := db.NewDB(ctx, dbArg)
	if err != nil {
		log.Fatalf("Failed to connect to database %q: %v", dbArg, err)
	}
	defer database.Close()

	fmt.Printf("Connected to database: %s\n", dbArg)

	// Step 1: Import
	fmt.Println("\n--- Step 1: Import Fresh Documents ---")
	patternsStr, err := prompt.PromptString(reader,
		"Enter file path, glob pattern, or directory of input files",
		"input/*.musicxml, input/*.mei, input/*.ly")
	if err != nil {
		log.Fatalf("Error reading input files: %v", err)
	}

	rawPatterns := strings.Split(patternsStr, ",")
	var importPatterns []string
	for _, p := range rawPatterns {
		p = strings.TrimSpace(p)
		if p != "" {
			importPatterns = append(importPatterns, p)
		}
	}

	importedCount, err := score.ImportScores(ctx, database, importPatterns...)
	if err != nil {
		log.Printf("Import completed with notices: %v", err)
	} else {
		fmt.Printf("Successfully imported %d document(s).\n", importedCount)
	}

	// Step 2: Search Sequences
	fmt.Println("\n--- Step 2: Search for Sequences ---")
	pieces, err := prompt.PromptIntList(reader, "Enter piece IDs (comma-separated or ranges e.g. 1-3, empty for all)")
	if err != nil {
		log.Fatalf("Invalid piece IDs: %v", err)
	}

	voices, err := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated or ranges e.g. 1-3, empty for all)")
	if err != nil {
		log.Fatalf("Invalid voice numbers: %v", err)
	}

	x, err := prompt.PromptInt(reader, "Enter minimum sequence length (x)", 10)
	if err != nil {
		log.Fatalf("Invalid sequence length: %v", err)
	}

	y, err := prompt.PromptInt(reader, "Enter minimum occurrences (y)", 2)
	if err != nil {
		log.Fatalf("Invalid occurrences: %v", err)
	}

	excludeSamePiece, err := prompt.PromptBool(reader, "Exclude sequences from same piece only?", false)
	if err != nil {
		log.Fatalf("Invalid boolean: %v", err)
	}

	defaultFile := fmt.Sprintf("Mente_%s.txt", time.Now().Format("20060102_150405"))
	outFile, err := prompt.PromptString(reader, "Enter output filename (optional, press Enter for default)", defaultFile)
	if err != nil {
		log.Fatalf("Invalid filename: %v", err)
	}

	qb, err := patterns.NewQueryBlock(ctx, database, pieces, false)
	if err != nil {
		log.Fatalf("Failed to build query block: %v", err)
	}

	finder := patterns.NewSequenceFinder(qb, voices)
	results := finder.FindRepeatedSubstrings(x, y, excludeSamePiece)
	results = patterns.FilterRepeatedResults(results, 0)

	fmt.Printf("\nFound %d maximal pattern(s) of length >= %d occurring >= %d time(s)\n", len(results), x, y)
	if err := finder.PrintResults(results, outFile); err != nil {
		log.Fatalf("Error printing results: %v", err)
	}
	if outFile != "" {
		fmt.Printf("Results written to %s\n", outFile)
	}
}
