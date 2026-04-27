package market

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/tcg_set_to_expansion.json
var embeddedTcgSetToExpansionJSON []byte

func loadDefaultExpansionMap() (map[string]int, error) {
	var m map[string]int
	if len(embeddedTcgSetToExpansionJSON) == 0 {
		return map[string]int{}, nil
	}
	if err := json.Unmarshal(embeddedTcgSetToExpansionJSON, &m); err != nil {
		return nil, fmt.Errorf("embedded tcg_set_to_expansion: %w", err)
	}
	if m == nil {
		return map[string]int{}, nil
	}
	return m, nil
}

func mergeExpansionMaps(base map[string]int, overlay map[string]int) map[string]int {
	out := make(map[string]int, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}
