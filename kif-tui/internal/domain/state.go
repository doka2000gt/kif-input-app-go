package domain

type Color byte // 'B' or 'W'

const (
	Black Color = 'B'
	White Color = 'W'
)

type PieceKind byte // 'P','L','N','S','G','B','R','K'

type Piece struct {
	Color Color
	Kind  PieceKind
	Prom  bool
}

type Square struct {
	File int // 1..9
	Rank int // 1..9
}

type Move struct {
	IsDrop  bool
	Kind    PieceKind
	From    *Square // nil if drop
	To      Square
	Promote bool
}

// Hands[color][kind] = count
type Hands map[Color]map[PieceKind]int

type State struct {
	Board      [10][10]*Piece // [file][rank] 1..9 only
	Hands      Hands
	SideToMove Color
	Moves      []Move

	history []Snapshot
}

type Snapshot struct {
	Board      [10][10]*Piece
	Hands      Hands
	SideToMove Color
	Moves      []Move
}

func NewStateEmpty() *State {
	s := &State{
		Hands:      NewHands(),
		SideToMove: Black,
		Moves:      make([]Move, 0),
		history:    make([]Snapshot, 0),
	}
	return s
}

func NewHands() Hands {
	return Hands{
		Black: map[PieceKind]int{},
		White: map[PieceKind]int{},
	}
}

func (s *State) CloneSnapshot() Snapshot {
	var b [10][10]*Piece
	for f := 1; f <= 9; f++ {
		for r := 1; r <= 9; r++ {
			if s.Board[f][r] == nil {
				b[f][r] = nil
				continue
			}
			p := *s.Board[f][r]
			b[f][r] = &p
		}
	}

	h := NewHands()
	for c, m := range s.Hands {
		for k, n := range m {
			h[c][k] = n
		}
	}

	mv := make([]Move, len(s.Moves))
	copy(mv, s.Moves)

	return Snapshot{
		Board:      b,
		Hands:      h,
		SideToMove: s.SideToMove,
		Moves:      mv,
	}
}

func (s *State) RestoreSnapshot(ss Snapshot) {
	// board
	for f := 1; f <= 9; f++ {
		for r := 1; r <= 9; r++ {
			if ss.Board[f][r] == nil {
				s.Board[f][r] = nil
				continue
			}
			p := *ss.Board[f][r]
			s.Board[f][r] = &p
		}
	}
	// hands
	s.Hands = NewHands()
	for c, m := range ss.Hands {
		for k, n := range m {
			s.Hands[c][k] = n
		}
	}
	// moves
	s.Moves = make([]Move, len(ss.Moves))
	copy(s.Moves, ss.Moves)

	s.SideToMove = ss.SideToMove
}

func (s *State) PushHistory() {
	s.history = append(s.history, s.CloneSnapshot())
}

func (s *State) Undo() bool {
	if len(s.history) == 0 {
		return false
	}
	last := s.history[len(s.history)-1]
	s.history = s.history[:len(s.history)-1]
	s.RestoreSnapshot(last)
	return true
}

func (s *State) PieceAt(sq Square) *Piece {
	if sq.File < 1 || sq.File > 9 || sq.Rank < 1 || sq.Rank > 9 {
		return nil
	}
	return s.Board[sq.File][sq.Rank]
}

func (s *State) SetPieceAt(sq Square, p *Piece) {
	if sq.File < 1 || sq.File > 9 || sq.Rank < 1 || sq.Rank > 9 {
		return
	}
	if p == nil {
		s.Board[sq.File][sq.Rank] = nil
		return
	}
	cp := *p
	s.Board[sq.File][sq.Rank] = &cp
}

func (s *State) ClearAll() {
	for f := 1; f <= 9; f++ {
		for r := 1; r <= 9; r++ {
			s.Board[f][r] = nil
		}
	}
	s.Hands = NewHands()
	s.Moves = nil
	s.SideToMove = Black
	s.history = nil
}
