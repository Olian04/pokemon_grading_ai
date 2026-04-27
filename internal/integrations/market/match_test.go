package market

import (
	"testing"

	"pokemon_ai/internal/integrations/pokemontcg"
)

func TestTcgSetAndLocal(t *testing.T) {
	for _, tc := range []struct {
		id, wantSet, wantLocal string
	}{
		{"base1-4", "base1", "4"},
		{"sv3pt5-GG70", "sv3pt5", "GG70"},
		{"nohyphen", "", ""},
		{"only-", "", ""},
	} {
		set, loc := tcgSetAndLocal(tc.id)
		if set != tc.wantSet || loc != tc.wantLocal {
			t.Fatalf("tcgSetAndLocal(%q) = (%q,%q) want (%q,%q)", tc.id, set, loc, tc.wantSet, tc.wantLocal)
		}
	}
}

func TestPickBestCard(t *testing.T) {
	cards := []pokemontcg.Card{
		{ID: "base1-4", Name: "Charizard"},
		{ID: "sv1-25", Name: "Pikachu"},
	}
	got := PickBestCard(cards, "sv1", "", "Pikachu")
	if got.ID != "sv1-25" {
		t.Fatalf("expected sv1-25, got %q", got.ID)
	}
	got = PickBestCard(cards, "", "4", "")
	if got.ID != "base1-4" {
		t.Fatalf("expected base1-4 for number hint, got %q", got.ID)
	}
}

func TestPickSingleRow(t *testing.T) {
	rows := []singleRow{
		{IDProduct: 1, EnName: "Charizard", Number: "4", Price: nil},
		{IDProduct: 2, EnName: "Charmander", Number: "3", Price: nil},
	}
	card := pokemontcg.Card{ID: "base1-4", Name: "Charizard"}
	row, ok := pickSingleRow(rows, card, "base1", "4")
	if !ok || row.IDProduct != 1 {
		t.Fatalf("expected product 1, ok=%v row=%+v", ok, row)
	}
}
