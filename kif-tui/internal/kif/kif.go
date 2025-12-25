package kif

import (
	"fmt"

	"kif-tui/internal/domain"
)

var totalCounts = map[domain.PieceKind]int{
	'R': 2, 'B': 2, 'G': 4, 'S': 4, 'N': 4, 'L': 4, 'P': 18, 'K': 2,
}

// compute_gote_remaining(board0, hands0_b) の移植
func ComputeGoteRemaining(board0 *[10][10]*domain.Piece, senteHand map[domain.PieceKind]int) map[domain.PieceKind]int {
	used := map[domain.PieceKind]int{}
	for k := range totalCounts {
		used[k] = 0
	}

	for f := 1; f <= 9; f++ {
		for r := 1; r <= 9; r++ {
			p := board0[f][r]
			if p == nil {
				continue
			}
			if _, ok := used[p.Kind]; ok {
				used[p.Kind]++
			}
		}
	}

	for kind, n := range senteHand {
		if _, ok := used[kind]; ok {
			used[kind] += n
		}
	}

	rem := map[domain.PieceKind]int{}
	for kind, total := range totalCounts {
		if kind == 'K' {
			continue
		}
		left := total - used[kind]
		if left > 0 {
			rem[kind] = left
		}
	}
	return rem
}

func HandsDictToPiyo(d map[domain.PieceKind]int) string {
	order := []domain.PieceKind{'R', 'B', 'G', 'S', 'N', 'L', 'P'}
	parts := make([]string, 0, 8)
	for _, k := range order {
		n := d[k]
		if n <= 0 {
			continue
		}
		parts = append(parts, pieceJP[k]+InvCountKanji(n))
	}
	if len(parts) == 0 {
		return ""
	}
	// Python版: "　".join(parts) + "　"
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "　" + parts[i]
	}
	out += "　"
	return out
}

func KifLineForMinimalMove(idx int, mv domain.Move, prevTo *domain.Square, sec int, totalSec int) (string, *domain.Square) {
	dst := ""
	if prevTo != nil && prevTo.File == mv.To.File && prevTo.Rank == mv.To.Rank {
		dst = "同"
	} else {
		dst = SqToKIF(mv.To.File, mv.To.Rank)
	}

	var body string
	if mv.IsDrop {
		body = dst + pieceJP[mv.Kind] + "打"
	} else {
		name := pieceJP[mv.Kind]
		if mv.Promote {
			name += "成"
		}
		body = dst + name + SqToParen(mv.From.File, mv.From.Rank)
	}

	// Python版は一旦 "( 0:..)" を作ってから最終整形で空白を削る
	timePart := fmt.Sprintf("( 0:%02d/00:00:%02d)", sec, totalSec)
	line := fmt.Sprintf("%4d %s %s", idx, fmt.Sprintf("%-12s", body), timePart)
	return line, &domain.Square{File: mv.To.File, Rank: mv.To.Rank}
}

type KIFOptions struct {
	HeaderComment string // 互換ヘッダ先頭行
}

func DefaultKIFOptions() KIFOptions {
	return KIFOptions{
		HeaderComment: "# ----  ANKIF向け / 自作詰将棋メーカー by TUI  ----",
	}
}

// GenerateKIF: Python版 _generate_kif_text 互換
func GenerateKIF(start domain.Snapshot, moves []domain.Move, opt KIFOptions) string {
	out := make([]string, 0, 64)

	// --- header ---
	out = append(out, opt.HeaderComment)
	out = append(out, "手合割：詰将棋")
	out = append(out, "先手：先手")
	out = append(out, "後手：後手")

	// --- start snapshot ---
	hands0b := start.Hands[domain.Black]
	if hands0b == nil {
		hands0b = map[domain.PieceKind]int{}
	}

	goteRem := ComputeGoteRemaining(&start.Board, hands0b)

	out = append(out, "後手の持駒："+HandsDictToPiyo(goteRem))
	out = append(out, domain.BoardToPiyo(&start.Board))
	out = append(out, "先手の持駒："+HandsDictToPiyo(hands0b))

	out = append(out, "終了日時："+NowYYYYMMDDHHMMSS())
	out = append(out, "手数----指手---------消費時間--")

	prevTo := (*domain.Square)(nil)
	totalSec := 0
	secPerMove := 1

	for i, mv := range moves {
		idx := i + 1
		totalSec += secPerMove
		line, newPrev := KifLineForMinimalMove(idx, mv, prevTo, secPerMove, totalSec)

		// 最終整形（Python版と同じ正規化）
		line = FinalizeLineSpacing(line)

		out = append(out, line)
		prevTo = newPrev
	}

	if len(moves) > 0 {
		out = append(out, fmt.Sprintf("まで%d手で詰み", len(moves)))
	}

	return joinLines(out) + "\n"
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	s := lines[0]
	for i := 1; i < len(lines); i++ {
		s += "\n" + lines[i]
	}
	return s
}
