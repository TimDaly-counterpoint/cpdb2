package main

import (
	"context"
	"fmt"
	"log"

	"cpdb2/pkg/installtools"
)

func main() {
	ctx := context.Background()
	if err := installtools.InstallTools(ctx, ""); err != nil {
		log.Fatalf("install_tools error: %v", err)
	}

	fmt.Println("install_tools finished successfully.")
}
