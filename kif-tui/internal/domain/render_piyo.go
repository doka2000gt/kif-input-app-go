package domain

// Python版 manual_kif.py:board_map_to_piyo を移植した盤面描画（KIF出力の開始局面用）

var pieceJP = map[PieceKind]string{
	'P': "歩", 'L': "香", 'N': "桂", 'S': "銀", 'G': "金", 'B': "角", 'R': "飛", 'K': "玉",
}

var kindToPyo = map[[2]interface{}]string{
	{'P', false}: "歩", {'L', false}: "香", {'N', false}: "桂", {'S', false}: "銀", {'G', false}: "金",
	{'B', false}: "角", {'R', false}: "飛", {'K', false}: "玉",

	// promoted (Python版 KIND_TO_PYO に合わせる)
	{'P', true}: "と",
	{'L', true}: "杏",
	{'N', true}: "圭",
	{'S', true}: "全",
	{'B', true}: "馬",
	{'R', true}: "竜",
}

var rankKanji = map[int]string{
	1: "一", 2: "二", 3: "三", 4: "四", 5: "五", 6: "六", 7: "七", 8: "八", 9: "九",
}

func BoardToPiyo(board *[10][10]*Piece) string {
	lines := make([]string, 0, 12)
	lines = append(lines, "  ９ ８ ７ ６ ５ ４ ３ ２ １")
	lines = append(lines, "+---------------------------+")
	for r := 1; r <= 9; r++ {
		row := ""
		for f := 9; f >= 1; f-- {
			p := board[f][r]
			if p == nil {
				row += " ・"
				continue
			}
			name := pieceJP[p.Kind]
			key := [2]interface{}{p.Kind, p.Prom}
			if v, ok := kindToPyo[key]; ok {
				name = v
			}
			cell := " " + name
			if p.Color == White {
				cell = "v" + name
			}
			row += cell
		}
		lines = append(lines, "|"+row+"|"+rankKanji[r])
	}
	lines = append(lines, "+---------------------------+")
	return joinLines(lines)
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}
