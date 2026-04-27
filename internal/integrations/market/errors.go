package market

import "errors"

var (
	// ErrCardmarketHTTP indicates a non-success HTTP response from Cardmarket.
	// Wrap with fmt.Errorf("%w: status %d", ErrCardmarketHTTP, code).
	ErrCardmarketHTTP = errors.New("cardmarket: request failed")

	// ErrCardmarketDecode is returned when the response body is not valid JSON for the expected shape.
	ErrCardmarketDecode = errors.New("cardmarket: response decode error")

	// ErrCardmarketOAuth is returned when OAuth configuration is incomplete or signing fails.
	ErrCardmarketOAuth = errors.New("cardmarket: oauth configuration error")
)
