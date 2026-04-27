package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pokemon_ai/internal/integrations/pokemontcg"
)

func TestBuildMarketResultEUFromSinglesTrend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/output.json/expansion/") && strings.HasSuffix(r.URL.Path, "/singles"):
			_, _ = w.Write([]byte(`{"single":[{"idProduct":501,"enName":"Charizard","number":"4","price":{"trend":42.5}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	base := strings.TrimSuffix(srv.URL, "/") + "/ws/v2.0"
	svc, err := NewService(Config{
		Cardmarket: CardmarketOAuthConfig{
			BaseURL:      base,
			AppToken:     "app",
			AppSecret:    "secret",
			AccessToken:  "tok",
			AccessSecret: "toksec",
		},
		TcgSetToExpansion: map[string]int{"base1": 99},
	})
	if err != nil {
		t.Fatal(err)
	}
	v := 99.0
	res := svc.BuildMarketResult(context.Background(), BuildInput{
		US:   pokemontcg.PriceSummary{ID: "base1-4", Holofoil: &v},
		Card: pokemontcg.Card{ID: "base1-4", Name: "Charizard", Number: "4"},
	})
	if res.EU.UnavailableReason != "" {
		t.Fatalf("unexpected EU unavailable: %q", res.EU.UnavailableReason)
	}
	if res.EU.CurrentMarketValue == nil || *res.EU.CurrentMarketValue != 42.5 {
		t.Fatalf("EU trend: got %v", res.EU.CurrentMarketValue)
	}
}

func TestBuildMarketResultEUFromProductEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/singles"):
			_, _ = w.Write([]byte(`{"single":[{"idProduct":777,"enName":"Charizard","number":"4"}]}`))
		case strings.Contains(r.URL.Path, "/output.json/product/777"):
			_, _ = w.Write([]byte(`{"price":{"trend":9.99}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	base := strings.TrimSuffix(srv.URL, "/") + "/ws/v2.0"
	svc, err := NewService(Config{
		Cardmarket: CardmarketOAuthConfig{
			BaseURL:      base,
			AppToken:     "a",
			AppSecret:    "b",
			AccessToken:  "c",
			AccessSecret: "d",
		},
		TcgSetToExpansion: map[string]int{"base1": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	v := 1.0
	res := svc.BuildMarketResult(context.Background(), BuildInput{
		US:   pokemontcg.PriceSummary{ID: "base1-4", Normal: &v},
		Card: pokemontcg.Card{ID: "base1-4", Name: "Charizard"},
	})
	if res.EU.UnavailableReason != "" {
		t.Fatalf("unexpected: %q", res.EU.UnavailableReason)
	}
	if res.EU.CurrentMarketValue == nil || *res.EU.CurrentMarketValue != 9.99 {
		t.Fatalf("got %v", res.EU.CurrentMarketValue)
	}
}

func TestBuildMarketResultMissingOAuth(t *testing.T) {
	svc, err := NewService(Config{Cardmarket: CardmarketOAuthConfig{}})
	if err != nil {
		t.Fatal(err)
	}
	v := 1.0
	res := svc.BuildMarketResult(context.Background(), BuildInput{
		US:   pokemontcg.PriceSummary{Normal: &v},
		Card: pokemontcg.Card{ID: "x-1", Name: "X"},
	})
	if res.EU.UnavailableReason == "" {
		t.Fatal("expected unavailable EU")
	}
}
