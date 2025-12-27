package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kif-tui/internal/tui"
)

func main() {
	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
