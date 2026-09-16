package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/puppe1990/grok-accounts/internal/cli"
)

func main() {
	home, err := cli.ResolveGrokHome(os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(1)
	}

	env := cli.Env{
		GrokHome: home,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Now:      time.Now,
		RunLogin: func(deviceAuth bool) error {
			args := []string{"login"}
			if deviceAuth {
				args = append(args, "--device-auth")
			}
			cmd := exec.Command("grok", args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("%w (o binário grok está no PATH?)", err)
			}
			return nil
		},
	}

	os.Exit(cli.Run(os.Args[1:], env))
}
