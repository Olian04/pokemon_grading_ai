package market

import (
	"strings"

	"pokemon_ai/internal/integrations/pokemontcg"
)

// tcgSetAndLocal splits a Pokemon TCG API card id (e.g. "base1-4", "sv3pt5-GG70")
// into set code and collector number / local id.
func tcgSetAndLocal(tcgCardID string) (setCode, localID string) {
	i := strings.LastIndex(tcgCardID, "-")
	if i <= 0 || i == len(tcgCardID)-1 {
		return "", ""
	}
	return tcgCardID[:i], tcgCardID[i+1:]
}

func normalizeNumber(s string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.ToLower(s)), "0")
}

// pickSingleRow chooses the best matching product row from expansion singles.
func pickSingleRow(rows []singleRow, card pokemontcg.Card, setCodeHint, numberHint string) (singleRow, bool) {
	set, local := tcgSetAndLocal(card.ID)
	if set == "" {
		return singleRow{}, false
	}
	wantNum := local
	if strings.TrimSpace(numberHint) != "" {
		wantNum = strings.TrimSpace(numberHint)
	}
	wantNumNorm := normalizeNumber(wantNum)
	nameNeedle := strings.TrimSpace(strings.ToLower(card.Name))
	best := -1
	var bestRow singleRow
	for _, r := range rows {
		score := 0
		rowNum := r.Number
		if strings.TrimSpace(rowNum) == "" {
			rowNum = r.LocalID
		}
		rowNumNorm := normalizeNumber(rowNum)
		if wantNumNorm != "" && rowNumNorm == wantNumNorm {
			score += 80
		}
		rowName := strings.TrimSpace(strings.ToLower(r.EnName))
		if rowName == "" {
			rowName = strings.TrimSpace(strings.ToLower(r.Name))
		}
		if nameNeedle != "" && rowName == nameNeedle {
			score += 60
		} else if nameNeedle != "" && strings.Contains(rowName, nameNeedle) {
			score += 40
		}
		if strings.TrimSpace(setCodeHint) != "" && strings.EqualFold(set, strings.TrimSpace(setCodeHint)) {
			score += 20
		}
		if score > best {
			best = score
			bestRow = r
		}
	}
	if best < 40 {
		return singleRow{}, false
	}
	return bestRow, true
}

func scoreCardChoice(c pokemontcg.Card, setCodeHint, numberHint, nameHint string) int {
	set, num := tcgSetAndLocal(c.ID)
	score := 0
	if strings.TrimSpace(setCodeHint) != "" && strings.EqualFold(set, strings.TrimSpace(setCodeHint)) {
		score += 100
	}
	if strings.TrimSpace(numberHint) != "" && strings.EqualFold(strings.TrimSpace(num), strings.TrimSpace(numberHint)) {
		score += 50
	}
	n := strings.TrimSpace(strings.ToLower(nameHint))
	if n != "" && strings.Contains(strings.ToLower(strings.TrimSpace(c.Name)), n) {
		score += 10
	}
	return score
}

// PickBestCard selects the TCG search hit that best matches optional set/number/name hints.
func PickBestCard(cards []pokemontcg.Card, setCodeHint, numberHint, nameHint string) pokemontcg.Card {
	if len(cards) == 0 {
		return pokemontcg.Card{}
	}
	bestI := 0
	best := scoreCardChoice(cards[0], setCodeHint, numberHint, nameHint)
	for i := 1; i < len(cards); i++ {
		s := scoreCardChoice(cards[i], setCodeHint, numberHint, nameHint)
		if s > best {
			best = s
			bestI = i
		}
	}
	return cards[bestI]
}
