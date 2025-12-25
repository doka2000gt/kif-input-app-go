package tui

import (
	"strings"

	"kif-tui/internal/domain"
)

// RenderBoard renders current position (m.st) in a fixed-width grid.
// Coordinate: [File 9..1] x [Rank 1..9] (KIF-style).
// We intentionally keep it plain and stable for UX/readability.
func RenderBoard(st *domain.State, cursor domain.Square) string {
	// Files header: ９..１
	// Use ASCII digits for now to avoid width issues; we keep columns aligned.
	// You can switch to full-width digits later if you prefer.
	var b strings.Builder
	b.WriteString("    9 8 7 6 5 4 3 2 1\n")
	b.WriteString("  +-------------------+\n")

	for r := 1; r <= 9; r++ {
		// Rank label on the left
		b.WriteString(" ")
		b.WriteByte(byte('0' + r))
		b.WriteString("|")

		for f := 9; f >= 1; f-- {
			sq := domain.Square{File: f, Rank: r}
			p := st.PieceAt(sq)
			isCursor := (sq == cursor)
			b.WriteString(cell(p, isCursor))
		}
		b.WriteString("|\n")
	}

	b.WriteString("  +-------------------+\n")
	return b.String()
}

// cell returns a fixed-width 2-char cell.
// We use "▲" for Black, "▽" for White, and a 1-letter piece kind.
// Promoted pieces are shown with '+' prefix-like marker by using lowercase mapping,
// but we keep it simple: same kind letter with a leading marker.
// Later you can switch to full Japanese piece glyphs safely.
func cell(p *domain.Piece, isCursor bool) string {
	if p == nil {
		if isCursor {
			return "[.]"
		}
		return " . "
	}

	tri := "▲"
	if p.Color == domain.White {
		tri = "▽"
	}
	s := tri + string(p.Kind)

	if isCursor {
		return "[" + s + "]"
	}
	return " " + s + " "
}
