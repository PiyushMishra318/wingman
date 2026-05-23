package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/piyushmishra318/wingman/internal/discover"
	"github.com/piyushmishra318/wingman/internal/model"
	"github.com/piyushmishra318/wingman/internal/redact"
	"github.com/piyushmishra318/wingman/internal/tui"
	"github.com/piyushmishra318/wingman/internal/upgrade"
)

func main() {
	yesAll := flag.Bool("y", false, "upgrade all auto-updatable packages and exit")
	installSc := flag.Bool("install-shortcut", false, "create Start Menu shortcut")
	shortcutName := flag.String("shortcut-name", "Wingman", "shortcut display name")
	flag.Parse()

	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	bat := filepath.Join(dir, "wingman.bat")
	icon := filepath.Join(dir, "assets", "wingman.ico")

	if *installSc {
		name := *shortcutName
		_, err := upgrade.InstallShortcut(bat, name, icon)
		if err != nil {
			fmt.Fprintf(os.Stderr, "shortcut failed: %s\n", redact.String(err.Error()))
			os.Exit(1)
		}
		fmt.Println("Created Start Menu shortcut:", name)
		return
	}

	if *yesAll {
		res := discover.All(false)
		var targets []model.Package
		for _, p := range res.Packages {
			if p.CanAutoUpdate() {
				targets = append(targets, p)
			}
		}
		upgrade.SortPackages(targets)
		ok, fail := 0, 0
		for _, p := range targets {
			fmt.Printf("[%s] %s…\n", p.Source, p.Name)
			if upgrade.Package(p, func(s string) { fmt.Println(" ", s) }) {
				ok++
			} else {
				fail++
			}
		}
		fmt.Printf("Done: %d ok, %d failed\n", ok, fail)
		if fail > 0 {
			os.Exit(1)
		}
		return
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", redact.String(err.Error()))
		os.Exit(1)
	}
}
