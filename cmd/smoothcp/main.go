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
	"cpdb2/pkg/smooth"
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

	if err := smooth.RunInteractive(ctx, database, reader); err != nil {
		log.Fatalf("Smooth table error: %v", err)
	}
}
