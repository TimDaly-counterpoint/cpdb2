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

	sourcePiece, err := prompt.PromptInt(reader, "Enter source piece ID to compare across corpus", 1)
	if err != nil {
		log.Fatalf("Invalid source piece ID: %v", err)
	}

	targetPieces, err := prompt.PromptIntList(reader, "Enter target piece IDs (comma-separated, empty for all other pieces)")
	if err != nil {
		log.Fatalf("Invalid target pieces: %v", err)
	}

	voices, err := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated, empty for all)")
	if err != nil {
		log.Fatalf("Invalid voice numbers: %v", err)
	}

	x, err := prompt.PromptInt(reader, "Enter minimum sequence length (x)", 10)
	if err != nil {
		log.Fatalf("Invalid sequence length: %v", err)
	}

	defaultFile := fmt.Sprintf("CrossPiece_%d_%s.txt", sourcePiece, time.Now().Format("20060102_150405"))
	outFile, err := prompt.PromptString(reader, "Enter output filename (optional, press Enter for default)", defaultFile)
	if err != nil {
		log.Fatalf("Invalid filename: %v", err)
	}

	var allPieces []int
	if len(targetPieces) > 0 {
		allPieces = append([]int{sourcePiece}, targetPieces...)
	}

	qb, err := patterns.NewQueryBlock(ctx, database, allPieces, false)
	if err != nil {
		log.Fatalf("Failed to build query block: %v", err)
	}

	finder := patterns.NewSequenceFinder(qb, voices)
	results := finder.FindCrossPieceSubstrings(sourcePiece, x)
	results = patterns.FilterRepeatedResults(results, sourcePiece)

	fmt.Printf("\nFound %d maximal pattern(s) from piece %d of length >= %d\n", len(results), sourcePiece, x)
	if err := finder.PrintResults(results, outFile); err != nil {
		log.Fatalf("Error printing results: %v", err)
	}
	if outFile != "" {
		fmt.Printf("Results written to %s\n", outFile)
	}
}
