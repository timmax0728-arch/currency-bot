package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/timmax0728-arch/currency-bot/internal/rates"
)

type fakeProvider struct {
	rate float64
	err  error
}

func (f fakeProvider) Rate(ctx context.Context, from, to string) (float64, error) {
	return f.rate, f.err
}

func TestBotConvert(t *testing.T) {
	tests := []struct {
		name     string
		provider fakeProvider
		input    string
		want     string
	}{
		{
			name:     "success",
			provider: fakeProvider{rate: 0.5},
			input:    "10 usd eur",
			want:     "10.00 USD = 5.00 EUR\nКурс: 1 USD = 0.5 EUR",
		},
		{name: "bad format", input: "привет", want: formatText},
		{name: "bad amount", input: "-5 usd eur", want: amountText},
		{
			name:     "unknown currency",
			provider: fakeProvider{err: rates.ErrUnknownCurrency},
			input:    "10 usd xxx",
			want:     unknownCurrencyText,
		},
		{
			name:     "provider failure",
			provider: fakeProvider{err: errors.New("boom")},
			input:    "10 usd eur",
			want:     failureText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bot{provider: tt.provider}
			if got := b.convert(context.Background(), tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBotRate(t *testing.T) {
	b := &Bot{provider: fakeProvider{rate: 2}}
	got := b.rate(context.Background(), "usd eur")
	want := "1.00 USD = 2.00 EUR\nКурс: 1 USD = 2 EUR"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}