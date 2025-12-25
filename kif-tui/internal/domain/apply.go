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
		p := s.Board[file][r]
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

func (s *State) toggleSide() {
	if s.SideToMove == Black {
		s.SideToMove = White
	} else {
		s.SideToMove = Black
	}
}
