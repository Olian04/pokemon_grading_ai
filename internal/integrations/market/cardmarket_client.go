package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/oauth1"
)

// CardmarketClient performs OAuth1-signed GET requests against Cardmarket API 2.0.
type CardmarketClient struct {
	baseURL string
	mu      sync.Mutex
	oauth   *oauth1.Config
	http    *http.Client
}

// NewCardmarketClient returns a client for MKM API 2.0. All OAuth fields must be non-empty.
func NewCardmarketClient(cfg CardmarketOAuthConfig) (*CardmarketClient, error) {
	if strings.TrimSpace(cfg.AppToken) == "" ||
		strings.TrimSpace(cfg.AppSecret) == "" ||
		strings.TrimSpace(cfg.AccessToken) == "" ||
		strings.TrimSpace(cfg.AccessSecret) == "" {
		return nil, fmt.Errorf("%w: missing oauth token or secret", ErrCardmarketOAuth)
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("%w: base url is empty", ErrCardmarketOAuth)
	}
	oauthCfg := oauth1.NewConfig(cfg.AppToken, cfg.AppSecret)
	tok := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	inner := oauthCfg.Client(context.Background(), tok)
	inner.Transport = transport
	inner.Timeout = timeout
	return &CardmarketClient{
		baseURL: base,
		oauth:   oauthCfg,
		http:    inner,
	}, nil
}

func (c *CardmarketClient) get(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.oauth.Realm = req.URL.String()
	resp, err := c.http.Do(req)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrCardmarketHTTP, resp.StatusCode)
	}
	return body, nil
}

// ExpansionSinglesURL builds the documented output.json path for singles in an expansion.
func (c *CardmarketClient) ExpansionSinglesURL(idExpansion int) string {
	return fmt.Sprintf("%s/output.json/expansion/%d/singles", c.baseURL, idExpansion)
}

// ProductURL builds the output.json path for a single product.
func (c *CardmarketClient) ProductURL(idProduct int) string {
	return fmt.Sprintf("%s/output.json/product/%d", c.baseURL, idProduct)
}

// FetchExpansionSingles returns singles rows for idExpansion.
func (c *CardmarketClient) FetchExpansionSingles(ctx context.Context, idExpansion int) ([]singleRow, error) {
	raw, err := c.get(ctx, c.ExpansionSinglesURL(idExpansion))
	if err != nil {
		return nil, err
	}
	rows, err := decodeSinglesResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCardmarketDecode, err)
	}
	return rows, nil
}

// FetchProductTrendEUR returns the trend price from the product resource (EUR for EU-region accounts).
func (c *CardmarketClient) FetchProductTrendEUR(ctx context.Context, idProduct int) (float64, error) {
	raw, err := c.get(ctx, c.ProductURL(idProduct))
	if err != nil {
		return 0, err
	}
	var p productDetailJSON
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCardmarketDecode, err)
	}
	if p.Price == nil || p.Price.Trend == nil {
		return 0, fmt.Errorf("%w: missing trend price", ErrCardmarketDecode)
	}
	return *p.Price.Trend, nil
}

func decodeSinglesResponse(raw []byte) ([]singleRow, error) {
	var direct []singleRow
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	var wrap struct {
		Single []singleRow `json:"single"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	if wrap.Single == nil {
		return []singleRow{}, nil
	}
	return wrap.Single, nil
}
