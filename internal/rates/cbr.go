package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const cbrURL = "https://www.cbr-xml-daily.ru/daily_json.js"

type CBR struct {
	client *http.Client
	url string
	ttl time.Duration
	now func() time.Time

	mu sync.RWMutex
	cached map[string]valute
	fetchedAt time.Time
}

func NewCBR(timeout, ttl time.Duration) *CBR{
	return &CBR{
		client : &http.Client{Timeout: timeout},
		url : 	 cbrURL,
		ttl: ttl,
		now : time.Now,
	}
}

type dailyResponse struct {
	Valute map[string]valute `json:"Valute"`
}

type valute struct {
	CharCode string `json:"CharCode"`
	Nominal int `json:"Nominal"`
	Value float64 `json:"Value"`
}

func (c *CBR) Rate(ctx context.Context, from, to string)(float64, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	
	table, err := c.table(ctx)
	if err != nil {
		return 0, err
	}

	fromRub, err := rubPerUnit(table, from)
	if err != nil{
		return 0, err
	}

	toRub, err := rubPerUnit(table, to)
	if err != nil{
		return 0, err
	}

	return fromRub / toRub, nil
}

func (c *CBR) table(ctx context.Context) (map[string]valute, error) {
	c.mu.RLock()
	if c.fresh() {
		t := c.cached
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.fresh() {
		return c.cached, nil
	}

	log.Printf("rates: refreshing from %s", c.url)
	data, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	c.cached = data.Valute
	c.fetchedAt = c.now()
	return c.cached, nil
}

func(c *CBR) fresh() bool {
	return c.cached != nil && c.now().Sub(c.fetchedAt) < c.ttl
}

func (c *CBR) fetch(ctx context.Context) (*dailyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "currency-bot/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rates API returned status %d", resp.StatusCode)
	}

	var data dailyResponse

	if err := json.NewDecoder(io.LimitReader(resp.Body, 5 << 20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode rates: %w", err)
	}

	if len(data.Valute) == 0 {
		return nil, fmt.Errorf("decode rates: no currencies in response")
	}
	return &data, nil
}

func rubPerUnit(rates map[string]valute, code string) (float64, error) {
	if code == "RUB" {
		return 1, nil
	}

	v, ok := rates[code]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownCurrency, code)
	}

	if v.Nominal <= 0 || v.Value <= 0 {
		return 0, fmt.Errorf("invalid data for currency %s", code)
	}

	return v.Value / float64(v.Nominal), nil
}