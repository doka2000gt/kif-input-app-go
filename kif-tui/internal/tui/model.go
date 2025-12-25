package tui

import (
	"fmt"
	"regexp"
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

var reNumericInput = regexp.MustCompile(`^\d{3,5}$`)

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

	// 数字入力（7776 / 77761 / 076）はコマンドより先に処理
	if reNumericInput.MatchString(line) {
		m.execNumeric(line)
		return
	}

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

	case "setup":
		// 平手初期局面をセット（EDITに戻す）
		m.st = domain.NewStateHirate()
		m.startSnapshot = nil
		m.appendLog("setup hirate (EDIT)")

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

func (m *Model) execNumeric(s string) {
	// 対局モードでのみ有効（EDIT で数字入力したい仕様なら、ここを変える）
	if !m.inPlay() {
		m.appendLog("not in PLAY. use start first.")
		return
	}

	tag, from, to, promote, err := domain.ParseNumeric(s)
	if err != nil {
		m.appendLog(fmt.Sprintf("invalid numeric: %v", err))
		return
	}

	switch tag {
	case "drop_pick":
		// 076: 駒種が入力にないので候補から決める
		cands := m.st.DropCandidates(to)
		if len(cands) == 0 {
			m.appendLog(fmt.Sprintf("drop: no candidates to=%v", to))
			return
		}
		if len(cands) > 1 {
			// 次ステップで picker UI にする（今はログ表示で止める）
			m.appendLog(fmt.Sprintf("drop ambiguous at %v: candidates=%v", to, cands))
			return
		}

		kind := cands[0]
		if err := m.st.ApplyMoveStrict(kind, nil, to, false, true); err != nil {
			m.appendLog(fmt.Sprintf("drop failed: %v", err))
			return
		}
		m.appendLog(fmt.Sprintf("drop %c to %v", kind, to))
		return

	case "move":
		if from == nil {
			m.appendLog("internal error: from is nil")
			return
		}
		p := m.st.PieceAt(*from)
		if p == nil {
			m.appendLog(fmt.Sprintf("no piece at from: %v", *from))
			return
		}
		kind := p.Kind

		if err := m.st.ApplyMoveStrict(kind, from, to, promote, false); err != nil {
			m.appendLog(fmt.Sprintf("move failed: %v", err))
			return
		}
		m.appendLog(fmt.Sprintf("move %v->%v promote=%v", *from, to, promote))
		return

	default:
		m.appendLog(fmt.Sprintf("unknown numeric tag: %s", tag))
		return
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
