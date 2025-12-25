package domain

import "testing"

func TestApplyMoveStrict_DropToOccupiedIsError(t *testing.T) {
	st := NewStateEmpty()
	st.SideToMove = Black
	st.Hands[Black] = map[PieceKind]int{'P': 1}

	// 先に 76 を埋める
	st.SetPieceAt(Square{File: 7, Rank: 6}, &Piece{Color: Black, Kind: 'G'})

	err := st.ApplyMoveStrict('P', nil, Square{File: 7, Rank: 6}, false, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyMoveStrict_CannotPromoteOnDrop(t *testing.T) {
	st := NewStateEmpty()
	st.SideToMove = Black
	st.Hands[Black] = map[PieceKind]int{'R': 1}

	err := st.ApplyMoveStrict('R', nil, Square{File: 5, Rank: 5}, true, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyMoveStrict_CannotCaptureOwnPiece(t *testing.T) {
	st := NewStateEmpty()
	st.SideToMove = Black

	st.SetPieceAt(Square{File: 7, Rank: 7}, &Piece{Color: Black, Kind: 'P'})
	st.SetPieceAt(Square{File: 7, Rank: 6}, &Piece{Color: Black, Kind: 'G'})

	from := Square{File: 7, Rank: 7}
	err := st.ApplyMoveStrict('P', &from, Square{File: 7, Rank: 6}, false, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyMoveStrict_DropPawnSameFileIsError(t *testing.T) {
	st := NewStateEmpty()
	st.SideToMove = Black
	st.Hands[Black] = map[PieceKind]int{'P': 1}

	// 先手の歩がすでに7筋に存在する
	st.SetPieceAt(Square{File: 7, Rank: 7}, &Piece{Color: Black, Kind: 'P'})

	// 7筋に歩打ちは二歩なのでエラー
	err := st.ApplyMoveStrict('P', nil, Square{File: 7, Rank: 5}, false, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
