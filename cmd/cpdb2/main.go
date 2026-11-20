package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cpdb2/pkg/cmatrix"
	"cpdb2/pkg/db"
	"cpdb2/pkg/disstable"
	"cpdb2/pkg/exportxml"
	"cpdb2/pkg/patterns"
	"cpdb2/pkg/prompt"
	"cpdb2/pkg/score"
	"cpdb2/pkg/setdb"
	"cpdb2/pkg/smooth"
	"cpdb2/pkg/sonority"
	"cpdb2/pkg/tritones"
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

	for {
		fmt.Println("\n================ CPDB2 Main Menu ================")
		fmt.Println("1.  Set up database (set_db)")
		fmt.Println("2.  Import scores (MusicXML, MEI, LilyPond)")
		fmt.Println("3.  Search repeated sequences across corpus")
		fmt.Println("4.  Search cross-piece sequence occurrences for a piece")
		fmt.Println("5.  Import scores and search sequences")
		fmt.Println("6.  Compare voices and generate CMatrix CSV")
		fmt.Println("7.  Recreate dissonance table (disstable)")
		fmt.Println("8.  Find tritones in dissonant sonorities")
		fmt.Println("9.  Smooth dissonance passages (smoothcp)")
		fmt.Println("10. Export smoothed table to MusicXML")
		fmt.Println("11. Export sequence search results to MusicXML")
		fmt.Println("12. Recreate sonority, dissonance, and consonant tables (process_sonority)")
		fmt.Println("0.  Exit")
		fmt.Println("=================================================")

		choice, err := prompt.PromptInt(reader, "Choose an option", 1)
		if err != nil {
			fmt.Printf("Invalid input: %v\n", err)
			continue
		}

		switch choice {
		case 0:
			fmt.Println("Exiting cpdb2. Goodbye!")
			return

		case 1:
			fmt.Println("\n--- Set Up Database ---")
			if err := setdb.RunInteractive(ctx, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 2:
			fmt.Println("\n--- Import Scores ---")
			patternsStr, _ := prompt.PromptString(reader, "Enter file path or glob pattern", "input/*.musicxml, input/*.mei, input/*.ly")
			var pats []string
			for _, p := range strings.Split(patternsStr, ",") {
				if t := strings.TrimSpace(p); t != "" {
					pats = append(pats, t)
				}
			}
			n, err := score.ImportScores(ctx, database, pats...)
			if err != nil {
				fmt.Printf("Import notice: %v\n", err)
			} else {
				fmt.Printf("Successfully imported %d score(s).\n", n)
			}

		case 3:
			fmt.Println("\n--- Search Sequences ---")
			pieces, _ := prompt.PromptIntList(reader, "Enter piece IDs (comma-separated or ranges e.g. 1-3, empty for all)")
			voices, _ := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated or ranges e.g. 1-3, empty for all)")
			x, _ := prompt.PromptInt(reader, "Enter minimum sequence length (x)", 10)
			y, _ := prompt.PromptInt(reader, "Enter minimum occurrences (y)", 2)
			excludeSame, _ := prompt.PromptBool(reader, "Exclude sequences from same piece only?", false)
			defFile := fmt.Sprintf("Mente_%s.txt", time.Now().Format("20060102_150405"))
			outFile, _ := prompt.PromptString(reader, "Enter output filename (press Enter for default)", defFile)

			qb, err := patterns.NewQueryBlock(ctx, database, pieces, false)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				break
			}
			finder := patterns.NewSequenceFinder(qb, voices)
			res := finder.FindRepeatedSubstrings(x, y, excludeSame)
			res = patterns.FilterRepeatedResults(res, 0)
			fmt.Printf("Found %d pattern(s).\n", len(res))
			_ = finder.PrintResults(res, outFile)

		case 4:
			fmt.Println("\n--- Search Cross-Piece Sequences ---")
			src, _ := prompt.PromptInt(reader, "Enter source piece ID", 1)
			targets, _ := prompt.PromptIntList(reader, "Enter target piece IDs (comma-separated or ranges e.g. 1-3, empty for all others)")
			voices, _ := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated or ranges e.g. 1-3, empty for all)")
			x, _ := prompt.PromptInt(reader, "Enter minimum sequence length (x)", 10)
			defFile := fmt.Sprintf("CrossPiece_%d_%s.txt", src, time.Now().Format("20060102_150405"))
			outFile, _ := prompt.PromptString(reader, "Enter output filename", defFile)

			var allP []int
			if len(targets) > 0 {
				allP = append([]int{src}, targets...)
			}
			qb, err := patterns.NewQueryBlock(ctx, database, allP, false)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				break
			}
			finder := patterns.NewSequenceFinder(qb, voices)
			res := finder.FindCrossPieceSubstrings(src, x)
			res = patterns.FilterRepeatedResults(res, src)
			fmt.Printf("Found %d cross-piece pattern(s).\n", len(res))
			_ = finder.PrintResults(res, outFile)

		case 5:
			fmt.Println("\n--- Import & Search Sequences ---")
			patternsStr, _ := prompt.PromptString(reader, "Enter file path or glob pattern", "input/*.musicxml, input/*.mei, input/*.ly")
			var pats []string
			for _, p := range strings.Split(patternsStr, ",") {
				if t := strings.TrimSpace(p); t != "" {
					pats = append(pats, t)
				}
			}
			_, _ = score.ImportScores(ctx, database, pats...)

			pieces, _ := prompt.PromptIntList(reader, "Enter piece IDs (comma-separated or ranges e.g. 1-3, empty for all)")
			voices, _ := prompt.PromptIntList(reader, "Enter voice numbers (comma-separated or ranges e.g. 1-3, empty for all)")
			x, _ := prompt.PromptInt(reader, "Enter minimum sequence length (x)", 10)
			y, _ := prompt.PromptInt(reader, "Enter minimum occurrences (y)", 2)
			excludeSame, _ := prompt.PromptBool(reader, "Exclude sequences from same piece only?", false)
			defFile := fmt.Sprintf("Mente_%s.txt", time.Now().Format("20060102_150405"))
			outFile, _ := prompt.PromptString(reader, "Enter output filename", defFile)

			qb, err := patterns.NewQueryBlock(ctx, database, pieces, false)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				break
			}
			finder := patterns.NewSequenceFinder(qb, voices)
			res := finder.FindRepeatedSubstrings(x, y, excludeSame)
			res = patterns.FilterRepeatedResults(res, 0)
			_ = finder.PrintResults(res, outFile)

		case 6:
			fmt.Println("\n--- CMatrix Voice Analysis ---")
			if err := cmatrix.RunInteractive(ctx, database, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 7:
			fmt.Println("\n--- Recreate Dissonance Table ---")
			if err := disstable.RunInteractive(ctx, database, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 8:
			fmt.Println("\n--- Find Tritones in Dissonant Sonorities ---")
			if err := tritones.RunInteractive(ctx, database, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 9:
			fmt.Println("\n--- Smooth Dissonance Table ---")
			if err := smooth.RunInteractive(ctx, database, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 10:
			fmt.Println("\n--- Export Smoothed Table to MusicXML ---")
			if err := exportxml.RunInteractiveSmooth(ctx, database, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 11:
			fmt.Println("\n--- Export Sequences to MusicXML ---")
			if err := exportxml.RunInteractive(ctx, database, reader, "patterns.txt"); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case 12:
			fmt.Println("\n--- Recreate Sonority, Dissonance, and Consonant Tables ---")
			if err := sonority.RunInteractive(ctx, database, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		default:
			fmt.Println("Unknown option. Please choose a number from 0 to 12.")
		}
	}
}
