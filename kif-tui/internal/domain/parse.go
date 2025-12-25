package domain

import (
	"fmt"
	"regexp"
	"strconv"
)

var reNumeric = regexp.MustCompile(`^\d{3,5}$`)

// ParseNumeric replicates Python behavior:
//   - "7776"  => normal move from 77 to 76
//   - "77761" => normal move + promote flag
//   - "076"   => drop_pick to 76 (0 + file + rank)
func ParseNumeric(s string) (tag string, from *Square, to Square, promote bool, err error) {
	if !reNumeric.MatchString(s) {
		return "", nil, Square{}, false, fmt.Errorf("numeric input must be 3..5 digits")
	}

	switch len(s) {
	case 3:
		// drop: 0 f r
		if s[0] != '0' {
			return "", nil, Square{}, false, fmt.Errorf("3-digit input must start with 0 for drop")
		}
		f, _ := strconv.Atoi(string(s[1]))
		r, _ := strconv.Atoi(string(s[2]))
		if f < 1 || f > 9 || r < 1 || r > 9 {
			return "", nil, Square{}, false, fmt.Errorf("square out of range")
		}
		return "drop_pick", nil, Square{File: f, Rank: r}, false, nil

	case 4, 5:
		ff, _ := strconv.Atoi(string(s[0]))
		fr, _ := strconv.Atoi(string(s[1]))
		tf, _ := strconv.Atoi(string(s[2]))
		tr, _ := strconv.Atoi(string(s[3]))
		if ff < 1 || ff > 9 || fr < 1 || fr > 9 || tf < 1 || tf > 9 || tr < 1 || tr > 9 {
			return "", nil, Square{}, false, fmt.Errorf("square out of range")
		}
		prom := false
		if len(s) == 5 {
			// last digit "1" means promote (Python版と同じ想定)
			last := s[4]
			if last == '1' {
				prom = true
			} else if last == '0' {
				prom = false
			} else {
				// 互換性を優先して厳密に（必要なら緩められる）
				return "", nil, Square{}, false, fmt.Errorf("5th digit must be 0 or 1")
			}
		}
		fsq := Square{File: ff, Rank: fr}
		tsq := Square{File: tf, Rank: tr}
		return "move", &fsq, tsq, prom, nil
	default:
		return "", nil, Square{}, false, fmt.Errorf("unexpected length")
	}
}
