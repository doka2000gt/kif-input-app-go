package tui

import (
	"fmt"
	"regexp"
	"strconv"
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
	modePicker
	modeHandEdit
)

// PlaceState represents continuous placement mode (EDIT only).
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

	// picker (generic)
	pickerOn    bool
	pickerIdx   int
	pickerTitle string
	pickerItems []string
	pickerMode  string // "place" / "drop" / "hand"

	// drop picker payload
	pickerDropTo domain.Square

	// hand edit
	handEditKind domain.PieceKind
}

var (
	// numeric input (7776 / 77761 / 076)
	reNumericInput = regexp.MustCompile(`^\d{3,5}$`)

	// hand edit accepts: "B=2 W=0" or "2 0"
	reHandEdit = regexp.MustCompile(`(?i)^\s*(?:B\s*=\s*(\d+)\s*)?(?:\s*[,; ]\s*)?(?:W\s*=\s*(\d+)\s*)?\s*$`)
	reTwoNums  = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s*$`)
)

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.Prompt = "> "
	ti.CharLimit = 200
	ti.Width = 60

	st := domain.NewStateEmpty()
	// EDIT中は先手番想定（混乱防止）
	st.SideToMove = domain.Black

	return Model{
		st:     st,
		cursor: domain.Square{File: 5, Rank: 5},

		place: PlaceState{
			On:      false,
			Color:   domain.Black,
			Kind:    'P',
			Promote: false,
		},

		m:     modeNormal,
		input: ti,
		logLines: []string{
			"ready (press i or : to input command)",
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
		// NORMAL mode
		// ----------------------------
		case modeNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "i", ":":
				m.m = modeInput
				m.input.SetValue("")
				m.input.Focus()
				m.appendLog("INPUT mode")
				return m, nil

			// cursor move (hjkl + arrows)
			// board renders files 9..1 left->right, so "left" means file+1.
			case "h", "left":
				m.moveCursor(+1, 0)
			case "l", "right":
				m.moveCursor(-1, 0)
			case "k", "up":
				m.moveCursor(0, -1)
			case "j", "down":
				m.moveCursor(0, +1)

			// placement toggle (EDIT only)
			case "P":
				if m.inPlay() {
					m.appendLog("cannot edit in PLAY (use clear/reset)")
					return m, nil
				}
				m.place.On = !m.place.On
				if m.place.On {
					m.appendLog("placement ON (Tab picker, L/N/S/G/B/R/K, p=Pawn, v toggle, + promote, space/enter place, x delete)")
				} else {
					m.appendLog("placement OFF")
				}
				return m, nil

			// placement controls (only when placement ON)
			case "v":
				if m.place.On && !m.inPlay() {
					if m.place.Color == domain.Black {
						m.place.Color = domain.White
					} else {
						m.place.Color = domain.Black
					}
				}
				return m, nil

			case "+":
				if m.place.On && !m.inPlay() {
					m.place.Promote = !m.place.Promote
				}
				return m, nil

			case "L", "N", "S", "G", "B", "R", "K":
				if m.place.On && !m.inPlay() {
					m.place.Kind = domain.PieceKind(msg.String()[0])
					m.placeAtCursor()
				}
				return m, nil

			case "p":
				if m.place.On && !m.inPlay() {
					m.place.Kind = 'P'
					m.placeAtCursor()
				}
				return m, nil

			case " ", "enter":
				if m.place.On && !m.inPlay() {
					m.placeAtCursor()
				}
				return m, nil

			case "x":
				if m.place.On && !m.inPlay() {
					m.st.SetPieceAt(m.cursor, nil)
					m.st.SideToMove = domain.Black
				}
				return m, nil

			case "tab":
				if m.place.On && !m.inPlay() {
					m.openPickerPlace()
					return m, nil
				}
				return m, nil

			// hands picker (EDIT only)
			case "H":
				if m.inPlay() {
					m.appendLog("hands picker is EDIT-only (use clear/reset to edit)")
					return m, nil
				}
				m.openPickerHand()
				return m, nil
			}

			return m, nil

		// ----------------------------
		// INPUT mode
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

		// ----------------------------
		// Picker (place/drop/hand)
		// ----------------------------
		case modePicker:
			switch msg.String() {
			case "esc", "tab":
				m.closePicker("picker OFF")
				return m, nil

			case "k", "up":
				m.pickerIdx = clamp(m.pickerIdx-1, 0, max(0, len(m.pickerItems)-1))
				return m, nil

			case "j", "down":
				m.pickerIdx = clamp(m.pickerIdx+1, 0, max(0, len(m.pickerItems)-1))
				return m, nil

			case "enter":
				switch m.pickerMode {
				case "place":
					// pickerItems are strings like "P", "L", ...
					if len(m.pickerItems) > 0 {
						m.place.Kind = domain.PieceKind(m.pickerItems[m.pickerIdx][0])
						m.appendLog(fmt.Sprintf("picker select: %c", m.place.Kind))
					}
					m.closePicker("")
					return m, nil

				case "hand":
					if len(m.pickerItems) == 0 {
						m.appendLog("hand picker empty")
						m.closePicker("")
						return m, nil
					}
					m.handEditKind = domain.PieceKind(m.pickerItems[m.pickerIdx][0])

					m.m = modeHandEdit
					b := m.st.Hands[domain.Black][m.handEditKind]
					w := m.st.Hands[domain.White][m.handEditKind]
					m.input.SetValue(fmt.Sprintf("B=%d W=%d", b, w))
					m.input.Focus()
					m.appendLog(fmt.Sprintf("hand edit: %c (enter to apply / esc to cancel)", m.handEditKind))
					return m, nil

				case "drop":
					// pickerItems = ["G", "P", ...]
					if len(m.pickerItems) == 0 {
						m.appendLog("drop picker empty")
						m.closePicker("")
						return m, nil
					}
					kind := domain.PieceKind(m.pickerItems[m.pickerIdx][0])
					to := m.pickerDropTo
					if err := m.st.ApplyMoveStrict(kind, nil, to, false, true); err != nil {
						m.appendLog(fmt.Sprintf("drop failed: %v", err))
						m.closePicker("")
						return m, nil
					}
					m.appendLog(fmt.Sprintf("drop %c to %v", kind, to))
					m.closePicker("")
					return m, nil

				default:
					m.appendLog("picker: unhandled mode: " + m.pickerMode)
					m.closePicker("")
					return m, nil
				}
			}
			return m, nil

		// ----------------------------
		// Hand edit
		// ----------------------------
		case modeHandEdit:
			switch msg.String() {
			case "esc":
				m.input.Blur()
				// hand edit をやめて picker に戻す
				m.m = modePicker
				m.appendLog("hand edit canceled")
				return m, nil

			case "enter":
				line := strings.TrimSpace(m.input.Value())
				b, w, err := parseHandEditLine(line)
				if err != nil {
					m.appendLog("hand edit invalid: " + err.Error())
					return m, nil
				}
				if err := validateHandCount(m.handEditKind, b, w); err != nil {
					m.appendLog("hand edit invalid: " + err.Error())
					return m, nil
				}

				m.st.Hands[domain.Black][m.handEditKind] = b
				m.st.Hands[domain.White][m.handEditKind] = w

				// EDIT中は常に先手番固定
				if !m.inPlay() {
					m.st.SideToMove = domain.Black
				}

				// picker表示更新して戻る
				m.pickerItems = handItems(m.st)

				m.input.Blur()
				m.m = modePicker
				m.appendLog(fmt.Sprintf("hand set: %c (B=%d W=%d)", m.handEditKind, b, w))
				return m, nil
			}

			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		return m, nil
	}

	return m, nil
}

func (m *Model) moveCursor(df, dr int) {
	f := m.cursor.File + df
	r := m.cursor.Rank + dr
	if f < 1 || f > 9 || r < 1 || r > 9 {
		return
	}
	m.cursor = domain.Square{File: f, Rank: r}
}

func (m *Model) placeAtCursor() {
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

	// EDIT中は先手番固定
	m.st.SideToMove = domain.Black

	// 配置後に1マス下へ
	m.moveCursor(0, +1)
}

func (m *Model) execCommand(line string) {
	m.appendLog("> " + line)

	// numeric input first
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

		// PLAY 開始時は必ず先手番から
		m.st.SideToMove = domain.Black

		m.appendLog("game started (PLAY)")

	case "setup":
		m.st = domain.NewStateHirate()
		m.startSnapshot = nil
		m.st.SideToMove = domain.Black
		m.appendLog("setup hirate (EDIT)")

	case "clear", "new", "reset":
		m.st = domain.NewStateEmpty()
		m.startSnapshot = nil
		m.st.SideToMove = domain.Black
		m.appendLog("cleared (EDIT)")

	case "kif":
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
		cands := m.st.DropCandidates(to)
		if len(cands) == 0 {
			m.appendLog(fmt.Sprintf("drop: no candidates to=%v", to))
			return
		}
		if len(cands) > 1 {
			m.openPickerDrop(to, cands)
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

func (m *Model) openPickerPlace() {
	m.m = modePicker
	m.pickerOn = true
	m.pickerMode = "place"
	m.pickerTitle = "Piece Picker"
	m.pickerItems = pieceOptionItems()
	m.pickerIdx = 0
	m.appendLog("picker ON (j/k or up/down, enter select, esc/tab close)")
}

func (m *Model) openPickerHand() {
	m.m = modePicker
	m.pickerOn = true
	m.pickerMode = "hand"
	m.pickerTitle = "Hands"
	m.pickerItems = handItems(m.st)
	m.pickerIdx = 0
	m.appendLog("hands picker ON (enter to edit counts, esc/tab close)")
}

func (m *Model) openPickerDrop(to domain.Square, cands []domain.PieceKind) {
	m.m = modePicker
	m.pickerOn = true
	m.pickerMode = "drop"
	m.pickerTitle = fmt.Sprintf("Drop Candidates to %d%d", to.File, to.Rank)
	m.pickerItems = dropCandidateItems(cands)
	m.pickerIdx = 0
	m.pickerDropTo = to
	m.appendLog("drop ambiguous: select piece to drop")
}

// closePicker: pickerを閉じたら必ずNORMALへ（操作不能防止）
func (m *Model) closePicker(logLine string) {
	m.m = modeNormal
	m.pickerOn = false
	m.pickerIdx = 0
	m.pickerTitle = ""
	m.pickerItems = nil
	m.pickerMode = ""
	m.pickerDropTo = domain.Square{File: 0, Rank: 0}
	if logLine != "" {
		m.appendLog(logLine)
	}
}

func (m *Model) appendLog(s string) {
	m.logLines = append(m.logLines, s)
	if len(m.logLines) > 200 {
		m.logLines = m.logLines[len(m.logLines)-200:]
	}
}

func (m Model) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	status := "EDIT"
	if m.inPlay() {
		status = "PLAY"
	}
	modeStr := "NORMAL"
	switch m.m {
	case modeInput:
		modeStr = "INPUT"
	case modePicker:
		modeStr = "PICKER"
	case modeHandEdit:
		modeStr = "HAND-EDIT"
	}

	turnMark := "▲"
	if m.st.SideToMove == domain.White {
		turnMark = "▽"
	}
	turnLabel := turnMark
	if m.inPlay() {
		turnLabel = fmt.Sprintf("%s %d", turnMark, len(m.st.Moves)+1)
	} else {
		turnLabel = fmt.Sprintf("%s EDIT", turnMark)
	}

	// ヘッダは背景付きバーにして見えなくなる問題を回避
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("7")).
		Padding(0, 1).
		Width(max(20, m.width))

	header := headerStyle.Render(
		fmt.Sprintf("kif-tui  [%s]  TURN:%s  mode:%s", status, turnLabel, modeStr),
	)

	// ---- left: board ----
	next := domain.Piece{Color: m.place.Color, Kind: m.place.Kind, Prom: m.place.Promote}
	boardBody := RenderBoard(m.st, m.cursor, m.place.On && !m.inPlay(), next)

	boardW := 38
	boardBox := boxStyle.Width(boardW).Render(boardBody)

	// ---- right: logs + picker + input ----
	rightWidth := max(20, m.width-2-boardW-1)

	logHeight := max(5, m.height-7)
	logStart := max(0, len(m.logLines)-logHeight)
	statusLine := fmt.Sprintf(
		"[%s] TURN:%s mode:%s",
		status, turnLabel, modeStr,
	)

	logBody := statusLine + "\n" +
		strings.Repeat("-", len(statusLine)) + "\n" +
		strings.Join(m.logLines[logStart:], "\n")
	innerLog := lipgloss.NewStyle().Width(max(10, rightWidth-2)).Render(logBody)
	logBox := boxStyle.Width(rightWidth).Height(logHeight).Render(innerLog)

	var inputLine string
	if m.m == modeInput || m.m == modeHandEdit {
		inputLine = m.input.View()
	} else {
		inputLine = "press i or : to enter command"
	}
	inputBox := boxStyle.Width(rightWidth).Render(inputLine)

	rightPane := lipgloss.JoinVertical(lipgloss.Top, logBox)

	if m.pickerOn {
		pickerBox := boxStyle.Width(rightWidth).Render(renderPicker(m.pickerTitle, m.pickerItems, m.pickerIdx))
		rightPane = lipgloss.JoinVertical(lipgloss.Top, rightPane, pickerBox)
	}

	rightPane = lipgloss.JoinVertical(lipgloss.Top, rightPane, inputBox)

	body := lipgloss.JoinHorizontal(lipgloss.Top, boardBox, rightPane)

	return header + "\n" + body + "\n"
}

// piece order used by picker
var pieceOptions = []domain.PieceKind{'P', 'L', 'N', 'S', 'G', 'B', 'R', 'K'}

func pieceOptionItems() []string {
	items := make([]string, 0, len(pieceOptions))
	for _, k := range pieceOptions {
		items = append(items, string(k))
	}
	return items
}

func handItems(st *domain.State) []string {
	items := make([]string, 0, len(pieceOptions))
	for _, k := range pieceOptions {
		b := st.Hands[domain.Black][k]
		w := st.Hands[domain.White][k]
		items = append(items, fmt.Sprintf("%c  (B:%d  W:%d)", k, b, w))
	}
	return items
}

func dropCandidateItems(cands []domain.PieceKind) []string {
	items := make([]string, 0, len(cands))
	for _, k := range cands {
		items = append(items, string(k))
	}
	return items
}

func renderPicker(title string, items []string, idx int) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	b.WriteString(strings.Repeat("-", len(title)) + "\n")
	for i, it := range items {
		prefix := "  "
		if i == idx {
			prefix = "> "
		}
		b.WriteString(prefix + it + "\n")
	}
	return b.String()
}

func parseHandEditLine(s string) (b int, w int, err error) {
	if m := reTwoNums.FindStringSubmatch(s); m != nil {
		bi, _ := strconv.Atoi(m[1])
		wi, _ := strconv.Atoi(m[2])
		return bi, wi, nil
	}

	m := reHandEdit.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, fmt.Errorf(`format: "B=2 W=0" or "2 0"`)
	}

	b = 0
	w = 0
	if m[1] != "" {
		b, _ = strconv.Atoi(m[1])
	}
	if m[2] != "" {
		w, _ = strconv.Atoi(m[2])
	}
	return b, w, nil
}

func validateHandCount(kind domain.PieceKind, b, w int) error {
	if b < 0 || w < 0 {
		return fmt.Errorf("counts must be >= 0")
	}
	const maxAny = 18
	if b > maxAny || w > maxAny {
		return fmt.Errorf("too many pieces: max=%d", maxAny)
	}
	_ = kind
	return nil
}

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
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
