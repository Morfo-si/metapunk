package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Morfo-si/metapunk/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via -ldflags "-X main.version=x.y.z".
// It falls back to "dev" for local builds without the flag.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("metapunk", version)
		return
	}

	p := tea.NewProgram(tui.NewAppModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
