package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/seed"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

func runSeed(cfg config.Config, args []string) error {
	if len(args) != 1 || args[0] != "demo" {
		return fmt.Errorf("usage: deepcuts seed demo")
	}
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	svc := service.New(d, cfg.Location)
	proofs, err := storage.NewLocal(filepath.Join(cfg.DataDir, "uploads"))
	if err != nil {
		return err
	}
	svc.Proofs = proofs
	info, err := seed.Demo(ctx, svc, auth.New(queries.New(d)))
	if err != nil {
		return err
	}
	fmt.Printf("seeded %d customers, %d products, %d orders into %s\n", info.Customers, info.Products, info.Orders, cfg.DBPath)
	fmt.Printf("office login:  %s / %s\n", info.OfficeEmail, info.OfficePassword)
	for _, dl := range info.Drivers {
		fmt.Printf("driver login:  %s PIN %s\n", dl.Name, dl.PIN)
	}
	return nil
}
