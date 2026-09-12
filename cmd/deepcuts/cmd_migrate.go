package main

import (
	"context"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
)

func runMigrate(cfg config.Config, args []string) error {
	d, err := db.OpenAndMigrate(context.Background(), cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	fmt.Println("migrated", cfg.DBPath)
	return nil
}
