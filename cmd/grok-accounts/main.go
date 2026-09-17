package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/puppe1990/grok-accounts/internal/browser"
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
		RunLogin: runLogin,
	}

	os.Exit(cli.Run(os.Args[1:], env))
}

func runLogin(opts cli.LoginOptions) error {
	args := []string{"login"}
	if opts.DeviceAuth {
		args = append(args, "--device-auth")
	}

	if !opts.Incognito {
		cmd := exec.Command("grok", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return grokError(cmd.Run())
	}

	b, err := resolveBrowser(opts.Browser)
	if err != nil {
		return fmt.Errorf("janela anônima: %w", err)
	}

	cmd := exec.Command("grok", args...)
	cmd.Stdin = os.Stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return grokError(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return grokError(err)
	}
	if err := cmd.Start(); err != nil {
		return grokError(err)
	}

	var once sync.Once
	open := func(url string) {
		once.Do(func() {
			if err := openPrivateWindow(b, url); err != nil {
				fmt.Fprintf(os.Stderr, "aviso: não foi possível abrir a janela anônima: %v\n", err)
			}
		})
	}

	var wg sync.WaitGroup
	for _, stream := range []struct {
		src io.Reader
		dst io.Writer
	}{{stdout, os.Stdout}, {stderr, os.Stderr}} {
		wg.Add(1)
		go func(src io.Reader, dst io.Writer) {
			defer wg.Done()
			if err := browser.WatchURLs(src, dst, open); err != nil {
				fmt.Fprintf(os.Stderr, "aviso: leitura do login interrompida: %v\n", err)
			}
		}(stream.src, stream.dst)
	}

	wg.Wait()
	return grokError(cmd.Wait())
}

func openPrivateWindow(b browser.Browser, url string) error {
	argv := browser.OpenCommand(b, url)
	return exec.Command(argv[0], argv[1:]...).Run()
}

func resolveBrowser(name string) (browser.Browser, error) {
	if runtime.GOOS != "darwin" {
		return browser.Browser{}, errors.New("abrir em janela anônima é suportado apenas no macOS")
	}
	if name != "" {
		return browser.ByName(name)
	}

	plist, err := launchServicesPlist()
	if err != nil {
		return browser.Browser{}, err
	}
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", plist).Output()
	if err != nil {
		return browser.Browser{}, fmt.Errorf("não foi possível ler a LaunchServices: %w", err)
	}
	return browser.DefaultBrowser(out)
}

func launchServicesPlist() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Preferences", "com.apple.LaunchServices", "com.apple.launchservices.secure.plist"), nil
}

func grokError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (o binário grok está no PATH?)", err)
}
