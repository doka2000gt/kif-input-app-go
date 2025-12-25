package kif

import (
	"fmt"
	"regexp"
	"time"

	"kif-tui/internal/domain"
)

var fwDigits = map[int]string{
	0: "０", 1: "１", 2: "２", 3: "３", 4: "４", 5: "５", 6: "６", 7: "７", 8: "８", 9: "９",
}

var rankKanji = map[int]string{
	1: "一", 2: "二", 3: "三", 4: "四", 5: "五", 6: "六", 7: "七", 8: "八", 9: "九",
}

var pieceJP = map[domain.PieceKind]string{
	'P': "歩", 'L': "香", 'N': "桂", 'S': "銀", 'G': "金", 'B': "角", 'R': "飛", 'K': "玉",
}

var NowFunc = func() string {
	// Python版: YYYY/MM/DD HH:MM:SS
	return time.Now().Format("2006/01/02 15:04:05")
}

func NowYYYYMMDDHHMMSS() string {
	return NowFunc()
}

func SqToKIF(file, rank int) string {
	return fwDigits[file] + rankKanji[rank]
}

func SqToParen(file, rank int) string {
	return fmt.Sprintf("(%d%d)", file, rank)
}

var reSpaceBeforeParen = regexp.MustCompile(`\s+\(`)
var reSpaceAfterLParen = regexp.MustCompile(`\(\s+`)

func FinalizeLineSpacing(line string) string {
	// 指手と消費時間の間は半角スペース1つ
	line = reSpaceBeforeParen.ReplaceAllString(line, " (")
	// "(" の直後の余分なスペースを削除
	line = reSpaceAfterLParen.ReplaceAllString(line, "(")
	return line
}

func InvCountKanji(n int) string {
	inv := map[int]string{
		1: "", 2: "二", 3: "三", 4: "四", 5: "五", 6: "六", 7: "七", 8: "八", 9: "九",
		10: "十", 11: "十一", 12: "十二", 13: "十三", 14: "十四", 15: "十五", 16: "十六", 17: "十七", 18: "十八",
	}
	if v, ok := inv[n]; ok {
		return v
	}
	return fmt.Sprintf("%d", n)
}
