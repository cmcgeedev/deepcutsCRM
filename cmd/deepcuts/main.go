// Command deepcuts is the single binary: API server, migrations, imports, users, demo seed.
package main

import (
	"fmt"
	"os"

	"github.com/cmcgeedev/deepcutsCRM/internal/config"
)

const usage = `usage: deepcuts <command> [args]

commands:
  serve                       run the API server and web app
  migrate                     apply database migrations and exit
  import customers <file.csv> import customers from CSV
  import products <file.csv>  import products from CSV
  import prices <file.csv>    import per-customer prices from CSV
  user add-office <email> <display name>   create an office user (prompts for password)
  user add-driver <display name>           create a driver (prompts for 6 digit PIN)
  user set-pin <display name>              reset a driver PIN
  user deactivate <email or display name>  deactivate a user
  seed demo                   load demo data into an empty database
`

type command func(cfg config.Config, args []string) error

var commands = map[string]command{
	"serve":   runServe,
	"migrate": runMigrate,
	"import":  runImport,
	"user":    runUser,
	"seed":    runSeed,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	cfg, err := config.FromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if err := cmd(cfg, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
