package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cpdb2/pkg/prompt"
	"cpdb2/pkg/setdb"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	dbArg := ""
	if len(os.Args) > 1 {
		dbArg = strings.TrimSpace(os.Args[1])
	}
	if dbArg == "" {
		var err error
		dbArg, err = prompt.PromptString(reader, "Enter name of empty database", "")
		if err != nil {
			log.Fatalf("Error reading database argument: %v", err)
		}
	}

	dbArg = strings.TrimSpace(dbArg)
	if dbArg == "" {
		log.Fatalf("Error: database name cannot be empty")
	}

	ctx := context.Background()
	if err := setdb.SetDB(ctx, dbArg); err != nil {
		log.Fatalf("set_db error: %v", err)
	}

	fmt.Printf("Successfully set up database %q.\n", dbArg)
}
