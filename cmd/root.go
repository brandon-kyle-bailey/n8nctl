// Package cmd provides the main command-line interface for n8nctl.
package cmd

import (
	"fmt"
	"os"

	"github.com/brandon-kyle-bailey/n8nctl/config"
	"github.com/brandon-kyle-bailey/n8nctl/entities"
)

func Execute() {
	if len(os.Args) < 2 {
		entities.PrintHelp()
		os.Exit(1)
	}

	entity := os.Args[1]

	if entity == "login" {
		entities.HandleLogin(os.Args[2:])
		return
	}

	if entity == "--help" || entity == "-h" || entity == "help" {
		entities.PrintHelp()
		return
	}

	actions, ok := entities.Entities[entity]
	if !ok {
		fmt.Printf("Unknown entity: %s\n\n", entity)
		entities.PrintHelp()
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\nPlease run `n8nctl login` first.\n", err)
		os.Exit(1)
	}

	entities.HandleEntityCommand(entity, os.Args[2:], actions, cfg)
}
