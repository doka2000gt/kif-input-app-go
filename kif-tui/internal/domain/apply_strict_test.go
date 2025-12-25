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

func TestApplyMoveStrict_DropPawnLastRankIsError(t *testing.T) {
	// [drop-pawn-last-rank]
	// 目的：歩を1段目に打てない（行き所のない駒）をStrictで保証する。
	st := NewStateEmpty()
	st.SideToMove = Black
	st.Hands[Black] = map[PieceKind]int{'P': 1}

	// 先手が 1段目（Rank=1）に歩打ちは禁止
	err := st.ApplyMoveStrict('P', nil, Square{File: 5, Rank: 1}, false, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyMoveStrict_DropLanceLastRankIsError(t *testing.T) {
	// [drop-lance-last-rank]
	// 目的：香を1段目に打てない（行き所のない駒）をStrictで保証する。
	st := NewStateEmpty()
	st.SideToMove = Black
	st.Hands[Black] = map[PieceKind]int{'L': 1}

	err := st.ApplyMoveStrict('L', nil, Square{File: 5, Rank: 1}, false, true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyMoveStrict_DropKnightLastTwoRanksIsError(t *testing.T) {
	// [drop-knight-last-two-ranks]
	// 目的：桂を1-2段目に打てない（行き所のない駒）をStrictで保証する。
	st := NewStateEmpty()
	st.SideToMove = Black
	st.Hands[Black] = map[PieceKind]int{'N': 1}

	// 先手：1段目
	err := st.ApplyMoveStrict('N', nil, Square{File: 5, Rank: 1}, false, true)
	if err == nil {
		t.Fatalf("expected error, got nil (rank=1)")
	}
	// 先手：2段目
	st.Hands[Black]['N'] = 1 // 消費されている可能性に備えて戻す
	err = st.ApplyMoveStrict('N', nil, Square{File: 5, Rank: 2}, false, true)
	if err == nil {
		t.Fatalf("expected error, got nil (rank=2)")
	}
}

func TestApplyMoveStrict_PromoteOutsideZoneIsError(t *testing.T) {
	// [promote-outside-zone]
	// 目的：敵陣に入っていない移動で Promote=true はエラーになることをStrictで保証する。
	st := NewStateEmpty()
	st.SideToMove = Black

	// 先手の歩：77 -> 76（敵陣ではない）
	st.SetPieceAt(Square{File: 7, Rank: 7}, &Piece{Color: Black, Kind: 'P'})
	from := Square{File: 7, Rank: 7}

	err := st.ApplyMoveStrict('P', &from, Square{File: 7, Rank: 6}, true, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyMoveStrict_PromoteWhenEnteringZoneIsOK(t *testing.T) {
	// [promote-entering-zone]
	// 目的：敵陣に「入る」移動（to が敵陣）では Promote=true が許可されることを確認する。
	st := NewStateEmpty()
	st.SideToMove = Black

	// 先手の歩：74 -> 73（to=3段目は先手の敵陣）
	st.SetPieceAt(Square{File: 7, Rank: 4}, &Piece{Color: Black, Kind: 'P'})
	from := Square{File: 7, Rank: 4}

	err := st.ApplyMoveStrict('P', &from, Square{File: 7, Rank: 3}, true, false)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestApplyMoveStrict_PromoteWhenLeavingZoneIsOK(t *testing.T) {
	// [promote-leaving-zone]
	// 目的：敵陣から「出る」移動（from が敵陣）でも Promote=true が許可されることを確認する。
	st := NewStateEmpty()
	st.SideToMove = Black

	// 先手の歩：73 -> 74（from=3段目は先手の敵陣）
	st.SetPieceAt(Square{File: 7, Rank: 3}, &Piece{Color: Black, Kind: 'P'})
	from := Square{File: 7, Rank: 3}

	err := st.ApplyMoveStrict('P', &from, Square{File: 7, Rank: 4}, true, false)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestApplyMoveStrict_PawnMustPromoteOnLastRank(t *testing.T) {
	// [pawn-must-promote]
	// 目的：歩が1段目に進む場合、不成は禁止（成が強制）されることを確認する。
	st := NewStateEmpty()
	st.SideToMove = Black

	// 先手の歩：72 -> 71
	st.SetPieceAt(Square{File: 7, Rank: 2}, &Piece{Color: Black, Kind: 'P'})
	from := Square{File: 7, Rank: 2}

	// 不成はエラー
	err := st.ApplyMoveStrict('P', &from, Square{File: 7, Rank: 1}, false, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// 成はOK
	err = st.ApplyMoveStrict('P', &from, Square{File: 7, Rank: 1}, true, false)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestApplyMoveStrict_LanceMustPromoteOnLastRank(t *testing.T) {
	// [lance-must-promote]
	// 目的：香が1段目に進む場合、不成は禁止（成が強制）されることを確認する。
	st := NewStateEmpty()
	st.SideToMove = Black

	// 先手の香：51 -> 51? ではなく、単純に 52 -> 51 の移動で作る
	st.SetPieceAt(Square{File: 5, Rank: 2}, &Piece{Color: Black, Kind: 'L'})
	from := Square{File: 5, Rank: 2}

	// 不成はエラー
	err := st.ApplyMoveStrict('L', &from, Square{File: 5, Rank: 1}, false, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// 成はOK
	err = st.ApplyMoveStrict('L', &from, Square{File: 5, Rank: 1}, true, false)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestApplyMoveStrict_KnightMustPromoteOnLastTwoRanks(t *testing.T) {
	// [knight-must-promote]
	// 目的：桂が1-2段目に進む場合、不成は禁止（成が強制）されることを確認する。
	st := NewStateEmpty()
	st.SideToMove = Black

	// 先手の桂：53 -> 41（=Rank1） / 53 -> 42（=Rank2）を作る
	st.SetPieceAt(Square{File: 5, Rank: 3}, &Piece{Color: Black, Kind: 'N'})
	from := Square{File: 5, Rank: 3}

	// --- to Rank=1 (41) ---
	err := st.ApplyMoveStrict('N', &from, Square{File: 4, Rank: 1}, false, false)
	if err == nil {
		t.Fatalf("expected error, got nil (to rank=1)")
	}
	err = st.ApplyMoveStrict('N', &from, Square{File: 4, Rank: 1}, true, false)
	if err != nil {
		t.Fatalf("expected nil, got %v (to rank=1)", err)
	}

	// 盤面が進んでいるので、再度初期化して Rank=2 のケース
	st = NewStateEmpty()
	st.SideToMove = Black
	st.SetPieceAt(Square{File: 5, Rank: 3}, &Piece{Color: Black, Kind: 'N'})
	from = Square{File: 5, Rank: 3}

	// --- to Rank=2 (42) ---
	err = st.ApplyMoveStrict('N', &from, Square{File: 4, Rank: 2}, false, false)
	if err == nil {
		t.Fatalf("expected error, got nil (to rank=2)")
	}
	err = st.ApplyMoveStrict('N', &from, Square{File: 4, Rank: 2}, true, false)
	if err != nil {
		t.Fatalf("expected nil, got %v (to rank=2)", err)
	}
}
