package kif

import "testing"

func TestFinalizeLineSpacing(t *testing.T) {
	in := "   1 ３三金(24)   ( 0:01/00:00:01)"
	want := "   1 ３三金(24) (0:01/00:00:01)"
	got := FinalizeLineSpacing(in)
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}
