package pokemontcg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL                string
	APIKey                 string
	FallbackRequestsPerMin int
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	limiter    <-chan time.Time
}

type Card struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number string `json:"number"`
}

type PriceSummary struct {
	ID       string   `json:"id"`
	Normal   *float64 `json:"normal,omitempty"`
	Holofoil *float64 `json:"holofoil,omitempty"`
	Reverse  *float64 `json:"reverse_holofoil,omitempty"`
}

func NewClient(cfg Config) *Client {
	rpm := cfg.FallbackRequestsPerMin
	if rpm <= 0 {
		rpm = 15
	}
	interval := time.Minute / time.Duration(rpm)
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		limiter: time.Tick(interval),
	}
}

func (c *Client) SearchCards(ctx context.Context, query string) ([]Card, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	path := fmt.Sprintf("%s/cards?q=name:%s&pageSize=15", c.baseURL, url.QueryEscape("*"+query+"*"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	out := struct {
		Data []Card `json:"data"`
	}{}
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) GetCardPricing(ctx context.Context, id string) (PriceSummary, error) {
	if strings.TrimSpace(id) == "" {
		return PriceSummary{}, errors.New("card id cannot be empty")
	}
	path := fmt.Sprintf("%s/cards/%s", c.baseURL, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return PriceSummary{}, err
	}
	out := struct {
		Data struct {
			ID        string `json:"id"`
			TCGPLayer struct {
				Prices struct {
					Normal struct {
						Market *float64 `json:"market"`
					} `json:"normal"`
					Holofoil struct {
						Market *float64 `json:"market"`
					} `json:"holofoil"`
					ReverseHolofoil struct {
						Market *float64 `json:"market"`
					} `json:"reverseHolofoil"`
				} `json:"prices"`
			} `json:"tcgplayer"`
		} `json:"data"`
	}{}
	if err := c.do(req, &out); err != nil {
		return PriceSummary{}, err
	}
	return PriceSummary{
		ID:       out.Data.ID,
		Normal:   out.Data.TCGPLayer.Prices.Normal.Market,
		Holofoil: out.Data.TCGPLayer.Prices.Holofoil.Market,
		Reverse:  out.Data.TCGPLayer.Prices.ReverseHolofoil.Market,
	}, nil
}

func (c *Client) do(req *http.Request, out any) error {
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	} else {
		<-c.limiter
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pokemontcg request failed with status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
