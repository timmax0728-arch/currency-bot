package converter

import (
	"context"
	"errors"
	"testing"

	"github.com/timmax0728-arch/currency-bot/internal/rates"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Request
		wantErr error
	}{
		{name: "basic", input: "100 usd eur", want: Request{100, "USD", "EUR"}},
		{name: "with to", input: "100 usd to eur", want: Request{100, "USD", "EUR"}},
		{name: "russian separator and comma", input: "1,5 eur в rub", want: Request{1.5, "EUR", "RUB"}},
		{name: "arrow", input: "100 USD -> EUR", want: Request{100, "USD", "EUR"}},
		{name: "extra spaces", input: "  2.5   gbp   usd  ", want: Request{2.5, "GBP", "USD"}},

		{name: "empty", input: "", wantErr: ErrInvalidFormat},
		{name: "too short", input: "100 usd", wantErr: ErrInvalidFormat},
		{name: "too long", input: "100 usd eur rub", wantErr: ErrInvalidFormat},
		{name: "short code", input: "100 us eur", wantErr: ErrInvalidFormat},
		{name: "digit in code", input: "100 usd1 eur", wantErr: ErrInvalidFormat},
		{name: "words instead of codes", input: "100 доллар евро", wantErr: ErrInvalidFormat},

		{name: "amount not first", input: "usd 100 eur", wantErr: ErrInvalidAmount},
		{name: "negative", input: "-5 usd eur", wantErr: ErrInvalidAmount},
		{name: "zero", input: "0 usd eur", wantErr: ErrInvalidAmount},
		{name: "not a number", input: "abc usd eur", wantErr: ErrInvalidAmount},
		{name: "NaN", input: "NaN usd eur", wantErr: ErrInvalidAmount},
		{name: "Inf", input: "Inf usd eur", wantErr: ErrInvalidAmount},
		{name: "too large", input: "1e20 usd eur", wantErr: ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got error %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

type fakeProvider struct {
	rate float64
	err  error
}

func (f fakeProvider) Rate(ctx context.Context, from, to string) (float64, error) {
	return f.rate, f.err
}


var _ rates.Provider = fakeProvider{}

func TestConvert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := fakeProvider{rate: 0.5}
		got, err := Convert(context.Background(), p, Request{Amount: 10, From: "USD", To: "EUR"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := Result{Amount: 10, From: "USD", To: "EUR", Rate: 0.5, Total: 5}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("provider error is passed on", func(t *testing.T) {
		p := fakeProvider{err: rates.ErrUnknownCurrency}
		_, err := Convert(context.Background(), p, Request{Amount: 10, From: "USD", To: "XXX"})
		if !errors.Is(err, rates.ErrUnknownCurrency) {
			t.Fatalf("got error %v, want ErrUnknownCurrency", err)
		}
	})
}

func TestResultString(t *testing.T) {
	r := Result{Amount: 100, From: "USD", To: "EUR", Rate: 0.9, Total: 90}
	want := "100.00 USD = 90.00 EUR\nКурс: 1 USD = 0.9 EUR"
	if got := r.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}