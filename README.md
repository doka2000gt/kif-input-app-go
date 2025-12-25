# kit 入力ツール Go言語版

## 概要
Python版をベースに、Go版として開発

## 構成
.
├── README.md
└── kif-tui
    ├── internal
    │   ├── domain
    │   │   ├── apply.go          // ApplyMoveMinimal/Undo/DropCandidates
    │   │   ├── parse.go          // ParseNumeric
    │   │   ├── render_piyo.go    // board→piyo（開始局面用も含む）
    │   │   └── state.go          // State/Snapshot/Move/Piece
    │   ├── kif
    │   │   ├── format.go         // sqToKif, sqToParen, finalizeSpacing
    │   │   └── kif.go            // GenerateKIF(snapshot, moves)
    │   └── tui
    │       ├── board_view.go
    │       ├── commands.go       // start/reset/undo/kif/s etc.
    │       ├── modals.go         // hand/drop/piece picker
    │       └── model.go          // bubbletea model / modes / panes
    └── main.go
