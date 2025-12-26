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

// PlaceState represents the "continuous placement" mode state (EDIT only).
// - On: placement mode enabled/disabled
// - Color: side of the piece to place (Black/White)
// - Kind: piece kind to place ('P','L','N','S','G','B','R','K')
// - Promote: promote flag for the placed piece
type PlaceState struct {
	On      bool
	Color   domain.Color
	Kind    domain.PieceKind
	Promote bool
}

type Model struct {
	st            *domain.State
	startSnapshot *domain.Snapshot // nil=EDIT, non-nil=PLAY

	cursor domain.Square
	place  PlaceState

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
		st:     st,
		cursor: domain.Square{File: 5, Rank: 5}, // 中央
		place: PlaceState{
			On:      false,
			Color:   domain.Black,
			Kind:    'P',
			Promote: false,
		},
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

		// ----------------------------
		// NORMAL mode (vim-like)
		// ----------------------------
		case modeNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "i":
				// enter INPUT (command) mode
				m.m = modeInput
				m.input.SetValue("")
				m.input.Focus()
				m.appendLog("INPUT mode")
				return m, nil

			// --- cursor move (hjkl + arrows) ---
			// NOTE: board is rendered with files 9..1 left->right,
			// so "left" means file+1, "right" means file-1.
			case "h", "left":
				m.moveCursor(+1, 0)
			case "l", "right":
				m.moveCursor(-1, 0)
			case "k", "up":
				m.moveCursor(0, -1)
			case "j", "down":
				m.moveCursor(0, +1)

			// --- placement mode toggle (EDIT only) ---
			case "P":
				if m.inPlay() {
					m.appendLog("cannot edit in PLAY (use clear/reset)")
					return m, nil
				}
				m.place.On = !m.place.On
				if m.place.On {
					m.appendLog("placement ON (P/L/N/S/G/B/R/K, v toggle, + promote, space/enter place, x delete)")
				} else {
					m.appendLog("placement OFF")
				}

			// --- placement controls (only when placement ON) ---
			case "v":
				if m.place.On && !m.inPlay() {
					if m.place.Color == domain.Black {
						m.place.Color = domain.White
					} else {
						m.place.Color = domain.Black
					}
				}

			case "+":
				if m.place.On && !m.inPlay() {
					m.place.Promote = !m.place.Promote
				}

			// choose piece kind (uppercase)
			case "L", "N", "S", "G", "B", "R", "K":
				if m.place.On && !m.inPlay() {
					m.place.Kind = domain.PieceKind(msg.String()[0])
					m.placeAtCursor()
				}
			// 歩は小文字p
			case "p":
				if m.place.On && !m.inPlay() {
					m.place.Kind = 'p'
					m.placeAtCursor()
				}

			// place piece at cursor (EDIT only)
			case " ", "enter":
				if m.place.On && !m.inPlay() {
					m.placeAtCursor()
				}

			// delete piece at cursor (EDIT only)
			case "x":
				if m.place.On && !m.inPlay() {
					m.st.SetPieceAt(m.cursor, nil)
				}
			}

			return m, nil

		// ----------------------------
		// INPUT mode (command line)
		// ----------------------------
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

func (m *Model) moveCursor(df, dr int) {
	f := m.cursor.File + df
	r := m.cursor.Rank + dr

	if f < 1 || f > 9 || r < 1 || r > 9 {
		return // 盤外は無視
	}

	m.cursor = domain.Square{File: f, Rank: r}
}

// placeAtCursor places the current "next piece" to the cursor square.
// After placement, it resets the "next piece" state (Python版挙動に寄せる).
func (m *Model) placeAtCursor() {
	// PLAY なら何もしない
	if m.inPlay() {
		return
	}

	p := &domain.Piece{
		Color: m.place.Color,
		Kind:  m.place.Kind,
		Prom:  m.place.Promote,
	}
	m.st.SetPieceAt(m.cursor, p)

	// 配置後だけリセット（空マスでは next を保持）
	m.place.Color = domain.Black
	m.place.Kind = 'P'
	m.place.Promote = false

	// 配置後に自動で1マス下へ（連続配置を快適に）
	m.moveCursor(0, +1)
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

	// placement status line (helps UX)
	placeStatus := "PLAY MODE"
	if !m.inPlay() {
		if m.place.On {
			side := "▲"
			if m.place.Color == domain.White {
				side = "▽"
			}
			prom := ""
			if m.place.Promote {
				prom = "+"
			}
			placeStatus = fmt.Sprintf("PLACEMENT: ON  next=%s%c%s  (space/enter place, x delete, v,+)", side, m.place.Kind, prom)
		} else {
			placeStatus = "PLACEMENT: OFF  (press P to toggle)"
		}
	}

	// ---- Board (left pane) ----
	next := domain.Piece{Color: m.place.Color, Kind: m.place.Kind, Prom: m.place.Promote}
	boardBody := RenderBoard(m.st, m.cursor, m.place.On && !m.inPlay(), next)

	// 盤面は折り返しが致命的なので、十分に幅を確保する（Logパネルは狭くてもOK)
	boardW := 38
	boardBox := boxStyle.Width(boardW).Render(boardBody)

	// 右は残り（最低幅だけ保証）
	rightWidth := max(20, m.width-2-boardW-1)

	logHeight := max(5, m.height-7) // header+status分を少し引く
	logStart := max(0, len(m.logLines)-logHeight)
	logBody := strings.Join(m.logLines[logStart:], "\n")

	// 長い行で横に崩れないように、右ペイン幅に収める
	inner := lipgloss.NewStyle().Width(max(10, rightWidth-2)).Render(logBody)
	logBox := boxStyle.Width(rightWidth).Height(logHeight).Render(inner)

	// ---- Input (right-bottom) ----
	var inputLine string
	if m.m == modeInput {
		inputLine = m.input.View()
	} else {
		inputLine = "press i to enter command"
	}
	inputBox := boxStyle.Width(rightWidth).Render(inputLine)

	rightPane := lipgloss.JoinVertical(lipgloss.Top, logBox, inputBox)

	// Join 2 columns
	body := lipgloss.JoinHorizontal(lipgloss.Top, boardBox, rightPane)

	return header + "\n" + placeStatus + "\n" + body + "\n"
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
