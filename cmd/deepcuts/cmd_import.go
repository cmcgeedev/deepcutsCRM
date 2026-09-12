package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/importer"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func runImport(cfg config.Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: deepcuts import customers|products|prices <file.csv> [--effective YYYY-MM-DD]")
	}
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	svc := service.New(d, cfg.Location)
	f, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	var sum importer.Summary
	switch args[0] {
	case "customers":
		sum, err = importer.Customers(ctx, svc, f)
	case "products":
		sum, err = importer.Products(ctx, svc, f)
	case "prices":
		eff := svc.Today()
		for i := 2; i+1 < len(args); i++ {
			if args[i] == "--effective" {
				eff = args[i+1]
			}
		}
		sum, err = importer.Prices(ctx, svc, f, eff)
	default:
		return fmt.Errorf("unknown import kind %q", args[0])
	}
	if err != nil {
		return err
	}
	fmt.Println(sum.String())
	return nil
}
