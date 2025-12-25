package tui

import tea "github.com/charmbracelet/bubbletea"

func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
