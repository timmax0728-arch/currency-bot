package rates

import(
	"context"
	"errors"
)

var ErrUnknownCurrency = errors.New("unknown currency")

type Provider interface{
	Rate(ctx context.Context, from, to string)(float64, error)
}