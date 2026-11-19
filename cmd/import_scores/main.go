package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cpdb2/pkg/db"
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

	patternsStr, err := prompt.PromptString(reader,
		"Enter file path, glob pattern, or directory of input files",
		"input/*.musicxml, input/*.mei, input/*.ly")
	if err != nil {
		log.Fatalf("Error reading input files: %v", err)
	}

	rawPatterns := strings.Split(patternsStr, ",")
	var patterns []string
	for _, p := range rawPatterns {
		p = strings.TrimSpace(p)
		if p != "" {
			patterns = append(patterns, p)
		}
	}

	count, err := score.ImportScores(ctx, database, patterns...)
	if err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	fmt.Printf("\nFinished importing %d score document(s) into database %q.\n", count, dbArg)
}
