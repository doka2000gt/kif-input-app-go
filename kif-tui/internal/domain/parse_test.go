package domain

import "testing"

func TestParseNumeric_Move4(t *testing.T) {
	tag, from, to, prom, err := ParseNumeric("7776")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "move" || from == nil || to.File != 7 || to.Rank != 6 || prom {
		t.Fatalf("unexpected: tag=%s from=%v to=%v prom=%v", tag, from, to, prom)
	}
}

func TestParseNumeric_Move5Promote(t *testing.T) {
	tag, _, _, prom, err := ParseNumeric("77761")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "move" || !prom {
		t.Fatalf("unexpected: tag=%s prom=%v", tag, prom)
	}
}

func TestParseNumeric_Drop(t *testing.T) {
	tag, from, to, prom, err := ParseNumeric("076")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "drop_pick" || from != nil || to.File != 7 || to.Rank != 6 || prom {
		t.Fatalf("unexpected: tag=%s from=%v to=%v prom=%v", tag, from, to, prom)
	}
}
