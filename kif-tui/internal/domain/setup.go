package domain

// NewStateHirate returns a standard initial position (平手).
// Coordinate system matches KIF-style squares:
// - File: 1..9
// - Rank: 1..9
// Example: Black pawn starts at 7,7 (７七歩), moves to 7,6 (７六歩).
func NewStateHirate() *State {
	s := NewStateEmpty()
	s.ClearAll() // clears board/hands/moves/side/history
	s.SideToMove = Black

	set := func(c Color, k PieceKind, f, r int) {
		s.SetPieceAt(Square{File: f, Rank: r}, &Piece{Color: c, Kind: k, Prom: false})
	}

	// --- Black (先手) ---
	// Back rank (rank 9): 9..1 = L N S G K G S N L
	set(Black, 'L', 9, 9)
	set(Black, 'N', 8, 9)
	set(Black, 'S', 7, 9)
	set(Black, 'G', 6, 9)
	set(Black, 'K', 5, 9)
	set(Black, 'G', 4, 9)
	set(Black, 'S', 3, 9)
	set(Black, 'N', 2, 9)
	set(Black, 'L', 1, 9)

	// Rook/Bishop
	set(Black, 'R', 2, 8) // 2八飛
	set(Black, 'B', 8, 8) // 8八角

	// Pawns (rank 7)
	for f := 1; f <= 9; f++ {
		set(Black, 'P', f, 7)
	}

	// --- White (後手) ---
	// Back rank (rank 1): 9..1 = L N S G K G S N L
	set(White, 'L', 9, 1)
	set(White, 'N', 8, 1)
	set(White, 'S', 7, 1)
	set(White, 'G', 6, 1)
	set(White, 'K', 5, 1)
	set(White, 'G', 4, 1)
	set(White, 'S', 3, 1)
	set(White, 'N', 2, 1)
	set(White, 'L', 1, 1)

	// Rook/Bishop
	set(White, 'R', 8, 2) // 8二飛
	set(White, 'B', 2, 2) // 2二角

	// Pawns (rank 3)
	for f := 1; f <= 9; f++ {
		set(White, 'P', f, 3)
	}

	return s
}
