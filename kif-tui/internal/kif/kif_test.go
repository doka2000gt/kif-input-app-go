package kif

import (
	"os"
	"path/filepath"
	"testing"

	"kif-tui/internal/domain"
)

func TestGenerateKIF_Golden(t *testing.T) {
	// 開始局面（空）+ デモ手順（main.go と同じ）
	st := domain.NewStateEmpty()
	start := st.CloneSnapshot()

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
			To:      domain.Square{File: 2, Rank: 2},
			Promote: false,
		},
	}

	// ★テストが日時で揺れないように、ここだけ固定値を注入できるようにする
	// 今は NowYYYYMMDDHHMMSS() が time.Now() なので、テスト側で差し替える仕組みを作るのがベスト。
	// そのために、kif/format.go に “NowFunc” を追加する（後述）。

	NowFunc = func() string { return "2000/01/01 00:00:00" }
	t.Cleanup(func() { NowFunc = func() string { return "2000/01/0100:00:00" } })

	wantPath := filepath.Join("testdata", "demo.golden.kif")
	got := GenerateKIF(start, moves, DefaultKIFOptions())

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
	want := string(wantBytes)

	if got != want {
		t.Fatalf("golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
