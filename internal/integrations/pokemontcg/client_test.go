package pokemontcg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchCardsWithAPIKeyHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Api-Key")
		_, _ = w.Write([]byte(`{"data":[{"id":"xy7-54","name":"Gyarados","number":"54"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{
		BaseURL: srv.URL,
		APIKey:  "test-key",
	})
	_, err := c.SearchCards(context.Background(), "Gyarados")
	if err != nil {
		t.Fatalf("SearchCards returned error: %v", err)
	}
	if gotHeader != "test-key" {
		t.Fatalf("expected X-Api-Key header to be set, got %q", gotHeader)
	}
}

func TestGetCardPricingParsesMarkets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"data":{
				"id":"base1-4",
				"tcgplayer":{
					"prices":{
						"normal":{"market":1.23},
						"holofoil":{"market":5.67},
						"reverseHolofoil":{"market":2.34}
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient(Config{
		BaseURL: srv.URL,
		APIKey:  "test-key",
	})
	out, err := c.GetCardPricing(context.Background(), "base1-4")
	if err != nil {
		t.Fatalf("GetCardPricing returned error: %v", err)
	}
	if out.ID != "base1-4" {
		t.Fatalf("unexpected id: %s", out.ID)
	}
	if out.Normal == nil || *out.Normal != 1.23 {
		t.Fatalf("unexpected normal market value: %+v", out.Normal)
	}
	if out.Holofoil == nil || *out.Holofoil != 5.67 {
		t.Fatalf("unexpected holofoil market value: %+v", out.Holofoil)
	}
	if out.Reverse == nil || *out.Reverse != 2.34 {
		t.Fatalf("unexpected reverse market value: %+v", out.Reverse)
	}
}
