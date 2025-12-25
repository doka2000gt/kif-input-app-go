// kif_test.go
//
// このファイルは KIF 出力の「完全互換性」を保証するための
// ゴールデンテストを集約したもの。
// 
// 各テストケースは、KIF特有の表記（同・打・成・盤面図など）が
// 将棋的・仕様的に壊れやすいポイントを意図的に含む局面を作り、
// GenerateKIF の出力が「一字一句」変わらないことを保証する。
//
// domain 側の実装や TUI 実装を変更しても、
// このテストが通る限り KIF 互換性は保たれる。

package kif

import (
	"os"
	"path/filepath"
	"testing"

	"kif-tui/internal/domain"
)

func TestGenerateKIF_Golden(t *testing.T) {
	// 日時の揺れを固定
	oldNow := NowFunc
	NowFunc = func() string { return "2000/01/01 00:00:00" }
	t.Cleanup(func() { NowFunc = oldNow })

	tests := []struct {
		name string
		make func(t *testing.T) (domain.Snapshot, []domain.Move)
	}{
		{
			// [demo]
			// 最小構成のデモケース。
			//
			// - start snapshot 方式が正しく機能すること
			// - 通常の指し手・打ちが KIF に反映されること
			// - 時間表記・行フォーマットが崩れていないこと
			//
			// 他のテストが失敗した場合の「基準点」としても使う。
			name: "demo",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()

				// 開始局面（空）
				start := snapshotAndClearForPlay(st)

				// demo moves: “domain経由” で積む
				// 盤上に必要な駒を用意してから動かす（minimal適用なので合法性は後で強化）
				st.SetPieceAt(domain.Square{File: 2, Rank: 4}, &domain.Piece{Color: domain.Black, Kind: 'G'})
				st.SetPieceAt(domain.Square{File: 3, Rank: 2}, &domain.Piece{Color: domain.White, Kind: 'K'})
				st.Hands[domain.Black]['G'] = 1

				mustMove(t, st, domain.Black, 'G', domain.Square{File: 2, Rank: 4}, domain.Square{File: 3, Rank: 3}, false)
				mustMove(t, st, domain.White, 'K', domain.Square{File: 3, Rank: 2}, domain.Square{File: 2, Rank: 1}, false)
				mustDrop(t, st, domain.Black, 'G', domain.Square{File: 2, Rank: 2})

				return start, st.Moves
			},
		},
		{
			// [promoted-pieces-in-startpos]
			// 開始局面の「盤面図」における成駒表記を固定するテスト。
			//
			// - 杏 / 圭 / 全 / 馬 / 竜 が正しい文字で出力されること
			// - 指し手が無い場合でも、盤面図が正しく出ること
			//
			// KIFでは「開始局面の盤面図」が非常に重要なため、
			// domain や描画ロジック変更時に壊れやすい箇所を守る。
			name: "promoted-pieces-in-startpos",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()
				// 開始局面に成駒（盤面図の互換性確認）
				st.SetPieceAt(domain.Square{File: 9, Rank: 1}, &domain.Piece{Color: domain.Black, Kind: 'L', Prom: true}) // 杏
				st.SetPieceAt(domain.Square{File: 8, Rank: 1}, &domain.Piece{Color: domain.Black, Kind: 'N', Prom: true}) // 圭
				st.SetPieceAt(domain.Square{File: 7, Rank: 1}, &domain.Piece{Color: domain.Black, Kind: 'S', Prom: true}) // 全
				st.SetPieceAt(domain.Square{File: 6, Rank: 1}, &domain.Piece{Color: domain.Black, Kind: 'B', Prom: true}) // 馬
				st.SetPieceAt(domain.Square{File: 5, Rank: 1}, &domain.Piece{Color: domain.Black, Kind: 'R', Prom: true}) // 竜

				start := snapshotAndClearForPlay(st)
				return start, st.Moves // 手順なし
			},
		},
		{
			// [drop-and-promote]
			// 「打」と「成」が同時に関わるケースを合法局面で固定するテスト。
			//
			// - 持駒からの「打」が正しく KIF に出ること
			// - 成れる駒（飛）が成った場合に「竜」と表記されること
			// - 打 → 相手手 → 成 の流れでも文脈が壊れないこと
			//
			// 将来的に ApplyMoveMinimal を厳密化した際にも、
			// KIF 表記が変わっていないかを検出するための重要ケース。
			name: "drop-and-promote",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()

				// 開始局面：玉だけ置いて、後手が動けるようにする（合法手確保）
				st.SetPieceAt(domain.Square{File: 5, Rank: 9}, &domain.Piece{Color: domain.Black, Kind: 'K'})
				st.SetPieceAt(domain.Square{File: 5, Rank: 1}, &domain.Piece{Color: domain.White, Kind: 'K'})

				// 先手の持駒：飛1（打＋成(竜)）
				st.Hands[domain.Black]['R'] = 1

				start := snapshotAndClearForPlay(st)

				// 1) 先手：飛打 55
				mustDrop(t, st, domain.Black, 'R', domain.Square{File: 5, Rank: 5})

				// 2) 後手：玉 51→52（適当な合法手）
				mustMove(t, st, domain.White, 'K', domain.Square{File: 5, Rank: 1}, domain.Square{File: 5, Rank: 2}, false)

				// 3) 先手：飛 55→54 成（竜）
				mustMove(t, st, domain.Black, 'R', domain.Square{File: 5, Rank: 5}, domain.Square{File: 5, Rank: 4}, true)

				return start, st.Moves
			},
		},
		{
			// [same-only]
			// KIF特有の「同」表記を保証する専用テスト。
			//
			// - 直前の指し手と同じマスに着手した場合に「同」と出ること
			// - Move 自体には「同」の情報を持たせず、
			//   表示側（KIF生成）が文脈で判断していることを確認する
			//
			// 「同」は KIF の中でも特に文脈依存度が高く、
			// 実装変更で壊れやすいため、単独ケースとして固定する。
			name: "same-only",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()

				// 開始局面（必要最低限）
				// 玉を置いて、局面として“それっぽさ”を確保（必須ではないが将来の厳密化に強い）
				st.SetPieceAt(domain.Square{File: 5, Rank: 9}, &domain.Piece{Color: domain.Black, Kind: 'K'})
				st.SetPieceAt(domain.Square{File: 5, Rank: 1}, &domain.Piece{Color: domain.White, Kind: 'K'})

				// 「同」を発生させるための配置：
				// 先手の歩：77
				// 後手の歩：75（これが76に取ると「同歩」になる）
				st.SetPieceAt(domain.Square{File: 7, Rank: 7}, &domain.Piece{Color: domain.Black, Kind: 'P'})
				st.SetPieceAt(domain.Square{File: 7, Rank: 5}, &domain.Piece{Color: domain.White, Kind: 'P'})

				start := snapshotAndClearForPlay(st)

				// 1) 先手：77→76（７六歩）
				mustMove(t, st, domain.Black, 'P', domain.Square{File: 7, Rank: 7}, domain.Square{File: 7, Rank: 6}, false)

				// 2) 後手：75→76（同歩）
				mustMove(t, st, domain.White, 'P', domain.Square{File: 7, Rank: 5}, domain.Square{File: 7, Rank: 6}, false)

				return start, st.Moves
			},
		},
		{
			// [same-drop]
			// KIF特有の「同」表記と「打」表記が同時に成立するケースを固定するテスト。
			//
			// 目的：
			// - 直前の着地点と同じマスに「打」を行った場合に「同◯打」と出力されること
			// - 「同」は文脈（直前のTo）で決まり、「打」は手の種類（IsDrop）で決まる、という
			//   “責務分離” が崩れていないことを確認する
			//
			// 局面設計：
			// 1手目で先手が 76 に着手し、2手目で後手が同じ 76 に「歩打」をする（＝同歩打）。
			//
			// 注意：
			// - 将来的に drop の厳密化（空マス必須）を入れると、この局面は成立しません。
			//   その場合は「同打」を別の合法局面へ作り直します（テストの意図は維持）。
			name: "same-drop",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()

				// 開始局面：玉だけ置いておく（将来の厳密化で“局面らしさ”が必要になった時に強い）
				st.SetPieceAt(domain.Square{File: 5, Rank: 9}, &domain.Piece{Color: domain.Black, Kind: 'K'})
				st.SetPieceAt(domain.Square{File: 5, Rank: 1}, &domain.Piece{Color: domain.White, Kind: 'K'})

				// 先手の歩：77（76に着手させるため）
				st.SetPieceAt(domain.Square{File: 7, Rank: 7}, &domain.Piece{Color: domain.Black, Kind: 'P'})

				// 後手の持駒：歩1（「歩打」を作るため）
				if st.Hands[domain.White] == nil {
					st.Hands[domain.White] = map[domain.PieceKind]int{}
				}
				st.Hands[domain.White]['P'] = 1

				start := snapshotAndClearForPlay(st)

				// 1) 先手：77→76（７六歩）
				mustMove(t, st, domain.Black, 'P',
					domain.Square{File: 7, Rank: 7},
					domain.Square{File: 7, Rank: 6},
					false)

				// 2) 後手：同歩打（直前と同じ 76 に歩を打つ）
				mustDrop(t, st, domain.White, 'P',
					domain.Square{File: 7, Rank: 6})

				return start, st.Moves
			},
		},
		{
			// [same-promote]
			// KIF特有の「同」表記と「成」表記が同時に成立するケースを固定するテスト。
			//
			// 目的：
			// - 直前の着地点と同じマスに着手した場合に「同」と出力されること
			// - その着手が Promote=true のときに「同◯成(…)」の形になること
			// - 「同」は文脈（直前のTo）、「成」は手の属性（Promote）で決まるという
			//   責務分離が保たれていることを確認する
			//
			// 局面設計：
			// 1手目：先手が 76 に着手
			// 2手目：後手が同じ 76 に着手し、Promote=true（＝同成）
			//
			// 注意：
			// - 将来的に「成れる条件（敵陣・移動元/先）」を厳密化した場合、
			//   この局面は“成の条件”を満たさず不正になる可能性があります。
			//   その場合は敵陣内で同成が発生する合法局面へ作り直します（テスト意図は維持）。
			name: "same-promote",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()

				// 開始局面：玉だけ置いておく（将来の厳密化に備えて局面らしさを確保）
				st.SetPieceAt(domain.Square{File: 5, Rank: 9}, &domain.Piece{Color: domain.Black, Kind: 'K'})
				st.SetPieceAt(domain.Square{File: 5, Rank: 1}, &domain.Piece{Color: domain.White, Kind: 'K'})

				// 「同成」を発生させるための配置：
				// 先手の歩：77 → 76
				// 後手の歩：75 → 76（Promote=true）
				st.SetPieceAt(domain.Square{File: 7, Rank: 7}, &domain.Piece{Color: domain.Black, Kind: 'P'})
				st.SetPieceAt(domain.Square{File: 7, Rank: 5}, &domain.Piece{Color: domain.White, Kind: 'P'})

				start := snapshotAndClearForPlay(st)

				// 1) 先手：77→76（７六歩）
				mustMove(t, st, domain.Black, 'P',
					domain.Square{File: 7, Rank: 7},
					domain.Square{File: 7, Rank: 6},
					false)

				// 2) 後手：75→76 Promote=true（同歩成(75) ＝ “同成” を固定）
				mustMove(t, st, domain.White, 'P',
					domain.Square{File: 7, Rank: 5},
					domain.Square{File: 7, Rank: 6},
					true)

				return start, st.Moves
			},
		},
		{
			// [same-drop-promote]
			// 「同」「打」「成」という KIF 表記要素がすべて同時に関与する
			// “設計上の最複雑ケース”を固定するためのテスト。
			//
			// 目的：
			// - 「同」は直前の着地点という“文脈”から決定されること
			// - 「打」は IsDrop によって決定されること
			// - 「成」は Promote フラグによって決定されること
			// - これら3要素が互いに独立して合成されることを保証する
			//
			// 注意（重要）：
			// - 将棋ルール上「打った駒がその手で成る」ことは不可能。
			// - このテストは“将棋的合法性”ではなく
			//   “KIF 表記ロジックの合成耐性”を確認するためのもの。
			// - ApplyMoveMinimal を将来的に厳密化した場合は、
			//   本ケースを合法な局面に作り直すか、削除する可能性がある。
			name: "same-drop-promote",
			make: func(t *testing.T) (domain.Snapshot, []domain.Move) {
				st := domain.NewStateEmpty()

				// 開始局面：玉のみ配置（最低限の局面成立）
				st.SetPieceAt(domain.Square{File: 5, Rank: 9}, &domain.Piece{Color: domain.Black, Kind: 'K'})
				st.SetPieceAt(domain.Square{File: 5, Rank: 1}, &domain.Piece{Color: domain.White, Kind: 'K'})

				// 先手の歩：77 → 76（「同」を発生させるための基準手）
				st.SetPieceAt(domain.Square{File: 7, Rank: 7}, &domain.Piece{Color: domain.Black, Kind: 'P'})

				// 後手の持駒：歩1（打を行うため）
				if st.Hands[domain.White] == nil {
					st.Hands[domain.White] = map[domain.PieceKind]int{}
				}
				st.Hands[domain.White]['P'] = 1

				start := snapshotAndClearForPlay(st)

				// 1) 先手：77→76（７六歩）
				mustMove(t, st, domain.Black, 'P',
					domain.Square{File: 7, Rank: 7},
					domain.Square{File: 7, Rank: 6},
					false)

				// 2) 後手：同歩打＋Promote=true
				// ※ 将棋的には不正だが、「同」「打」「成」の合成テストとして実施
				mustDrop(t, st, domain.White, 'P',
					domain.Square{File: 7, Rank: 6})
				// Promote フラグを強制的に立てる（表記確認用）
				st.Moves[len(st.Moves)-1].Promote = true

				return start, st.Moves
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, moves := tc.make(t)
			got := GenerateKIF(start, moves, DefaultKIFOptions())

			wantPath := filepath.Join("testdata", tc.name+".golden.kif")

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.MkdirAll(filepath.Dir(wantPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(wantPath, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated golden: %s", wantPath)
				return
			}

			wantBytes, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read golden failed: %v (set UPDATE_GOLDEN=1 to create)", err)
			}
			if got != string(wantBytes) {
				// 失敗時に got をファイルに書き出して差分確認しやすくする
				// 例: testdata/same-only.got.kif
				gotPath := filepath.Join("testdata", tc.name+".got.kif")
				_ = os.WriteFile(gotPath, []byte(got), 0o644)

				t.Fatalf(
				    "golden mismatch.\n\nwrote: %s\ncompare (PowerShell): fc %s %s\n\n--- got ---\n%s\n--- want ---\n%s",
				    gotPath, gotPath, wantPath, got, string(wantBytes),
				)
			}
		})
	}
}

// startスナップショット方式をテストでも再現：
// 「開始局面を確定」→「手順をクリアして対局モード開始」相当
func snapshotAndClearForPlay(st *domain.State) domain.Snapshot {
	start := st.CloneSnapshot()
	st.Moves = nil
	// undo履歴もクリア（start後にundoできない方針にするならこれが自然）
	// 今は history が非公開なので Undo() できなくするには State に ClearHistory を追加してもOK。
	return start
}

func mustMove(t *testing.T, st *domain.State, side domain.Color, kind domain.PieceKind, from domain.Square, to domain.Square, promote bool) {
	t.Helper()
	st.SideToMove = side
	if err := st.ApplyMoveMinimal(kind, &from, to, promote, false); err != nil {
		t.Fatalf("mustMove failed: side=%c kind=%c from=%v to=%v prom=%v err=%v", side, kind, from, to, promote, err)
	}
}

func mustDrop(t *testing.T, st *domain.State, side domain.Color, kind domain.PieceKind, to domain.Square) {
	t.Helper()
	st.SideToMove = side
	// dropのために持駒が必要
	if st.Hands[side] == nil {
		st.Hands[side] = map[domain.PieceKind]int{}
	}
	if st.Hands[side][kind] <= 0 {
		st.Hands[side][kind] = 1
	}
	if err := st.ApplyMoveMinimal(kind, nil, to, false, true); err != nil {
		t.Fatalf("mustDrop failed: side=%c kind=%c to=%v err=%v", side, kind, to, err)
	}
}
