package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"kif-tui/internal/domain"
	"kif-tui/internal/kif"
)

type mode int

const (
	modeNormal mode = iota
	modeInput
)

type Model struct {
	st            *domain.State
	startSnapshot *domain.Snapshot // nil=EDIT, non-nil=PLAY

	m        mode
	input    textinput.Model
	logLines []string

	width  int
	height int
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.Prompt = "> "
	ti.CharLimit = 200
	ti.Width = 60

	st := domain.NewStateEmpty()

	return Model{
		st:    st,
		m:     modeNormal,
		input: ti,
		logLines: []string{
			"ready (press i to input command)",
		},
	}
}

func (m Model) inPlay() bool { return m.startSnapshot != nil }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = min(80, max(30, m.width-4))
		return m, nil

	case tea.KeyMsg:
		switch m.m {
		case modeNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "i":
				m.m = modeInput
				m.input.SetValue("")
				m.input.Focus()
				m.appendLog("INPUT mode")
				return m, nil
			default:
				return m, nil
			}

		case modeInput:
			switch msg.String() {
			case "esc":
				m.m = modeNormal
				m.input.Blur()
				m.appendLog("NORMAL mode")
				return m, nil
			case "enter":
				cmdline := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				m.m = modeNormal
				m.input.Blur()

				if cmdline != "" {
					m.execCommand(cmdline)
				} else {
					m.appendLog("NORMAL mode")
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) execCommand(line string) {
	m.appendLog("> " + line)
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "start":
		snap := m.st.CloneSnapshot()
		m.startSnapshot = &snap
		m.st.Moves = nil
		m.appendLog("game started (PLAY)")

	case "clear", "new", "reset":
		m.st = domain.NewStateEmpty()
		m.startSnapshot = nil
		m.appendLog("cleared (EDIT)")

	case "kif":
		// startSnapshot が無ければ現在局面を開始局面として扱う（最小デモ）
		start := m.startSnapshot
		if start == nil {
			s := m.st.CloneSnapshot()
			start = &s
		}

		out := kif.GenerateKIF(*start, m.st.Moves, kif.DefaultKIFOptions())

		m.appendLog("KIF preview:")
		for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			m.appendLog("  " + ln)
		}

	default:
		m.appendLog(fmt.Sprintf("unknown command: %s", parts[0]))
	}
}

func (m *Model) appendLog(s string) {
	m.logLines = append(m.logLines, s)
	if len(m.logLines) > 200 {
		m.logLines = m.logLines[len(m.logLines)-200:]
	}
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	status := "EDIT"
	if m.inPlay() {
		status = "PLAY"
	}
	modeStr := "NORMAL"
	if m.m == modeInput {
		modeStr = "INPUT"
	}

	header := titleStyle.Render(fmt.Sprintf("kif-tui  [%s]  mode:%s", status, modeStr))

	// ログ領域
	logHeight := max(5, m.height-6)
	logStart := max(0, len(m.logLines)-logHeight)
	logBody := strings.Join(m.logLines[logStart:], "\n")
	logBox := boxStyle.Width(max(20, m.width-2)).Height(logHeight).Render(logBody)

	// 入力領域
	var inputLine string
	if m.m == modeInput {
		inputLine = m.input.View()
	} else {
		inputLine = "press i to enter command"
	}
	inputBox := boxStyle.Width(max(20, m.width-2)).Render(inputLine)

	return header + "\n" + logBox + "\n" + inputBox + "\n"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
