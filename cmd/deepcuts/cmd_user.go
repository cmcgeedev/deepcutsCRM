package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
)

// readSecret prompts on stderr and reads without echo when stdin is a terminal; otherwise reads a line.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func runUser(cfg config.Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: deepcuts user add-office <email> <display name> | add-driver <display name> | set-pin <display name> | deactivate <email or name>")
	}
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	a := auth.New(queries.New(d))
	switch args[0] {
	case "add-office":
		if len(args) < 3 {
			return fmt.Errorf("usage: deepcuts user add-office <email> <display name>")
		}
		pw, err := readSecret("password: ")
		if err != nil {
			return err
		}
		u, err := a.CreateOfficeUser(ctx, args[1], strings.Join(args[2:], " "), pw)
		if err != nil {
			return err
		}
		fmt.Println("created office user", u.ID, u.Email.String)
	case "add-driver":
		pin, err := readSecret("6 digit PIN: ")
		if err != nil {
			return err
		}
		u, err := a.CreateDriver(ctx, strings.Join(args[1:], " "), pin)
		if err != nil {
			return err
		}
		fmt.Println("created driver", u.ID, u.DisplayName)
	case "set-pin":
		pin, err := readSecret("new 6 digit PIN: ")
		if err != nil {
			return err
		}
		if err := a.SetDriverPIN(ctx, strings.Join(args[1:], " "), pin); err != nil {
			return err
		}
		fmt.Println("pin updated")
	case "deactivate":
		if err := a.DeactivateUser(ctx, strings.Join(args[1:], " ")); err != nil {
			return err
		}
		fmt.Println("deactivated")
	default:
		return fmt.Errorf("unknown user subcommand %q", args[0])
	}
	return nil
}
