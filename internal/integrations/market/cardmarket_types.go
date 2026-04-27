package market

// singleRow is a subset of Cardmarket expansion singles JSON.
type singleRow struct {
	IDProduct int    `json:"idProduct"`
	Name      string `json:"name,omitempty"`
	EnName    string `json:"enName,omitempty"`
	Number    string `json:"number,omitempty"`
	// LocalID is sometimes used instead of number in API variants.
	LocalID string `json:"localId,omitempty"`
	Price   *struct {
		Trend *float64 `json:"trend,omitempty"`
	} `json:"price,omitempty"`
}

// productDetailJSON is a subset of Cardmarket product JSON used for EU pricing.
// We prefer trend price in EUR (Cardmarket lists trend in account currency; EU accounts use EUR).
type productDetailJSON struct {
	Price *struct {
		Trend *float64 `json:"trend,omitempty"`
	} `json:"price,omitempty"`
}
