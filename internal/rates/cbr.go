package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const cbrURL = "https://www.cbr-xml-daily.ru/daily_json.js"

type CBR struct {
	client *http.Client
	url string
}

func NewCBR(timeout time.Duration) *CBR{
	return &CBR{
		client : &http.Client{Timeout: timeout},
		url : 	 cbrURL,
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
	
	data, err := c.fetch(ctx)
	if err != nil {
		return 0, err
	}

	fromRub, err := rubPerUint(data.Valute, from)
	if err != nil{
		return 0, err
	}

	toRub, err := rubPerUint(data.Valute, to)
	if err != nil{
		return 0, err
	}

	return fromRub / toRub, nil
}

func (c *CBR) fetch(ctx context.Context) (*dailyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil{
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

func rubPerUint(rates map[string]valute, code string) (float64, error) {
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