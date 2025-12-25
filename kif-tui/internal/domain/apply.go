package domain

import (
	"fmt"
)

var promotable = map[PieceKind]bool{
	'P': true, 'L': true, 'N': true, 'S': true, 'B': true, 'R': true,
}

func (s *State) ensureHands() {
	if s.Hands == nil {
		s.Hands = NewHands()
	}
	if s.Hands[Black] == nil {
		s.Hands[Black] = map[PieceKind]int{}
	}
	if s.Hands[White] == nil {
		s.Hands[White] = map[PieceKind]int{}
	}
}

// DropCandidates: to が空で、かつ手番側が持っている駒種を返す（歩は二歩を軽チェック）
func (s *State) DropCandidates(to Square) []PieceKind {
	if s.PieceAt(to) != nil {
		return nil
	}
	s.ensureHands()
	side := s.SideToMove
	h := s.Hands[side]
	out := make([]PieceKind, 0, len(h))
	for k, n := range h {
		if n <= 0 {
			continue
		}
		if k == 'P' && s.hasPawnOnFile(side, to.File) {
			// 二歩チェック（簡易）
			continue
		}
		out = append(out, k)
	}
	return out
}

func (s *State) hasPawnOnFile(side Color, file int) bool {
	for r := 1; r <= 9; r++ {
		p := s.PieceAt(Square{File: file, Rank: r})
		if p == nil {
			continue
		}
		if p.Color == side && p.Kind == 'P' && !p.Prom {
			return true
		}
	}
	return false
}

// ApplyMoveMinimal: Python版の minimal move 適用をGoで再現する（厳密ルールは後で追加可能）
func (s *State) ApplyMoveMinimal(kind PieceKind, from *Square, to Square, promote bool, isDrop bool) error {
	s.ensureHands()

	// undo
	s.PushHistory()

	side := s.SideToMove

	if isDrop {
		// hand consume
		if s.Hands[side][kind] <= 0 {
			return fmt.Errorf("no such piece in hand: %c", kind)
		}
		s.Hands[side][kind]--
		if s.Hands[side][kind] <= 0 {
			delete(s.Hands[side], kind)
		}

		s.SetPieceAt(to, &Piece{Color: side, Kind: kind, Prom: false})
		s.Moves = append(s.Moves, Move{
			IsDrop:  true,
			Kind:    kind,
			From:    nil,
			To:      to,
			Promote: false,
		})
		s.toggleSide()
		return nil
	}

	if from == nil {
		return fmt.Errorf("from is nil for non-drop move")
	}

	p := s.PieceAt(*from)
	if p == nil {
		return fmt.Errorf("no piece at from")
	}
	if p.Color != side {
		return fmt.Errorf("piece color mismatch")
	}

	// capture -> add to hand (unpromoted kind)
	dest := s.PieceAt(to)
	if dest != nil {
		s.Hands[side][dest.Kind] = s.Hands[side][dest.Kind] + 1
	}

	// move
	s.SetPieceAt(*from, nil)
	np := *p
	if promote {
		if !promotable[np.Kind] {
			return fmt.Errorf("not promotable: %c", np.Kind)
		}
		np.Prom = true
	}
	s.SetPieceAt(to, &np)

	s.Moves = append(s.Moves, Move{
		IsDrop:  false,
		Kind:    kind,
		From:    from,
		To:      to,
		Promote: promote,
	})
	s.toggleSide()
	return nil
}

// ApplyMoveStrict は、対局モード（指し手入力）向けの厳密適用。
// - 打ちは空マス必須
// - 自駒への着手禁止
// - 移動元に駒が存在し、手番と一致すること
// - Promote は「成れる駒」だけ（※成れる条件の厳密チェックは後で拡張）
func (st *State) ApplyMoveStrict(kind PieceKind, from *Square, to Square, promote bool, isDrop bool) error {
	st.ensureHands()

	// 呼び出し側の指定ミスを吸収：from == nil なら必ず drop とみなす
	isDrop = (from == nil)

	// 盤外チェック
	if to.File < 1 || to.File > 9 || to.Rank < 1 || to.Rank > 9 {
		return fmt.Errorf("out of board: to=%v", to)
	}
	if from != nil {
		if from.File < 1 || from.File > 9 || from.Rank < 1 || from.Rank > 9 {
			return fmt.Errorf("out of board: from=%v", *from)
		}
	}

	if isDrop {
		// 打ちは空マス必須
		if st.PieceAt(to) != nil {
			return fmt.Errorf("drop to occupied square: to=%v", to)
		}
		// 持駒が必要
		if st.Hands[st.SideToMove][kind] <= 0 {
			return fmt.Errorf("no piece in hand: %c", kind)
		}
		// 二歩チェックはここ（歩打ちの場合のみ）
		if kind == 'P' && st.hasPawnOnFile(st.SideToMove, to.File) {
			return fmt.Errorf("double pawn on file: file=%d", to.File)
		}
		// 行き所のない駒の禁止（打ち）
		// 先手視点：歩・香は1段目、桂は1-2段目に打てない。
		// 後手視点：歩・香は9段目、桂は8-9段目に打てない。
		if isDrop {
			lastRank := 1
			secondLastRank := 2
			if st.SideToMove == White {
				lastRank = 9
				secondLastRank = 8
			}

			switch kind {
			case 'P', 'L':
				if to.Rank == lastRank {
					return fmt.Errorf("illegal drop (no legal moves): %c to rank=%d", kind, to.Rank)
				}
			case 'N':
				if to.Rank == lastRank || to.Rank == secondLastRank {
					return fmt.Errorf("illegal drop (no legal moves): %c to rank=%d", kind, to.Rank)
				}
			}
		}
		// 「打ち」で成はできない
		if promote {
			return fmt.Errorf("cannot promote on drop: %c", kind)
		}
	} else {
		// 移動は from 必須
		if from == nil {
			return fmt.Errorf("missing from for move")
		}
		p := st.PieceAt(*from)
		if p == nil {
			return fmt.Errorf("no piece at from: %v", *from)
		}
		if p.Color != st.SideToMove {
			return fmt.Errorf("piece color mismatch")
		}
		// 自駒を取れない
		dst := st.PieceAt(to)
		if dst != nil && dst.Color == st.SideToMove {
			return fmt.Errorf("cannot capture own piece: to=%v", to)
		}
		// 成れる駒だけ成れる
		if promote && !isPromotable(kind) {
			return fmt.Errorf("not promotable: %c", kind)
		}
		// 成は敵陣に入る／出るときだけ許可（from または to が敵陣）
		if promote {
			if !inPromotionZone(st.SideToMove, *from) && !inPromotionZone(st.SideToMove, to) {
				return fmt.Errorf("promotion not allowed outside zone: from=%v to=%v", *from, to)
			}
		}
		// 不成だと行き所がなくなる駒は、成が強制
		if !promote {
			lastRank := 1
			secondLastRank := 2
			if st.SideToMove == White {
				lastRank = 9
				secondLastRank = 8
			}

			switch kind {
			case 'P', 'L':
				if to.Rank == lastRank {
					return fmt.Errorf("must promote on last rank: %c", kind)
				}
			case 'N':
				if to.Rank == lastRank || to.Rank == secondLastRank {
					return fmt.Errorf("must promote on last ranks: %c", kind)
				}
			}
		}
	}
	// 実際の更新は minimal に委譲
	return st.ApplyMoveMinimal(kind, from, to, promote, isDrop)
}

// 成れる駒（将棋の基本）
// 歩香桂銀角飛のみ。金・玉は成れない。
func isPromotable(kind PieceKind) bool {
	switch kind {
	case 'P', 'L', 'N', 'S', 'B', 'R':
		return true
	default:
		return false
	}
}

func inPromotionZone(side Color, sq Square) bool {
	// 先手：敵陣は 1〜3段目
	// 後手：敵陣は 7〜9段目
	if side == Black {
		return sq.Rank >= 1 && sq.Rank <= 3
	}
	return sq.Rank >= 7 && sq.Rank <= 9
}

func (s *State) toggleSide() {
	if s.SideToMove == Black {
		s.SideToMove = White
	} else {
		s.SideToMove = Black
	}
}
