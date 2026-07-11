package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: go run ./cmd/migrate <up|down> [steps]")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	dsn := cfg.Database.MigrationDSN()

	switch args[0] {
	case "up":
		return database.RunMigrations(dsn)
	case "down":
		steps := 1
		if len(args) > 1 {
			steps, err = strconv.Atoi(args[1])
			if err != nil || steps < 1 {
				return fmt.Errorf("down steps must be a positive integer")
			}
		}
		return database.RollbackMigrations(dsn, steps)
	default:
		return fmt.Errorf("unknown migration action %q; use up or down", args[0])
	}
}
