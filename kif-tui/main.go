package main

import (
	"fmt"

	"kif-tui/internal/domain"
	"kif-tui/internal/kif"
	"kif-tui/internal/tui"
)

func main() {
	// ひとまずコンパイル確認 & KIF生成確認用のデモ
	_ = demoKIF()

	// TUI（仮）
	if err := tui.Run(); err != nil {
		fmt.Println(err)
	}
}

func demoKIF() string {
	// start snapshot: empty
	st := domain.NewStateEmpty()
	start := st.CloneSnapshot()

	// demo moves: 7776, 3332 みたいに “Move” を積んで確認できる
	// （本番は domain.ApplyMoveMinimal で moves を作る）
	moves := []domain.Move{
		{
			IsDrop:  false,
			Kind:    'G',
			From:    &domain.Square{File: 2, Rank: 4},
			To:      domain.Square{File: 3, Rank: 3},
			Promote: false,
		},
		{
			IsDrop:  false,
			Kind:    'K',
			From:    &domain.Square{File: 3, Rank: 2},
			To:      domain.Square{File: 2, Rank: 1},
			Promote: false,
		},
		{
			IsDrop:  true,
			Kind:    'G',
			From:    nil,
			To:      domain.Square{File: 2, Rank: 2},
			Promote: false,
		},
	}

	txt := kif.GenerateKIF(start, moves, kif.DefaultKIFOptions())
	fmt.Println("---- KIF DEMO ----")
	fmt.Print(txt)
	return txt
}
