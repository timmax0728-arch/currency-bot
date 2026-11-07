package converter

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"context"

	"github.com/timmax0728-arch/currency-bot/internal/rates"
)


const maxAmount = 1e12

var(
	ErrInvalidFormat = errors.New("invalid format")
	ErrInvalidAmount = errors.New("invalid amount")
)

type Request struct {
	Amount float64
	From string
	To string
}

type Result struct {
	Amount float64
	From string
	To string
	Rate float64
	Total float64
}

func Parse(text string) (Request, error) {
	fields := strings.Fields(text)

	if len(fields) == 4 && isSeparator(fields[2]) {
		fields = []string{fields[0], fields[1], fields[3]}
	}

	if len(fields) != 3 {
		return Request{}, fmt.Errorf("%w: expected \"<amount> <from> <to>\"", ErrInvalidFormat)
	}

	amount, err := parseAmount(fields[0])
	if err != nil{
		return Request{}, err
	}

	from, err := parseCode(fields[1])
	if err != nil {
		return Request{}, err
	}

	to, err := parseCode(fields[2])
	if err != nil {
		return Request{}, err
	}

	return Request{Amount: amount, From: from, To: to}, nil
}

func isSeparator(s string) bool {
	switch strings.ToLower(s) {
	case "to", "in", "в", "->":
		return true
	}
	return false
}

func parseAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not a number", ErrInvalidAmount, s)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > maxAmount {
		return 0, fmt.Errorf("%w: %q", ErrInvalidAmount, s)
	}

	return v, nil
}

func parseCode(s string) (string, error) {
	s = strings.ToUpper(s)
	if len(s) != 3 {
		return "", fmt.Errorf("%w: bad currency code %q", ErrInvalidFormat, s)
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("%w: bad currency code %q", ErrInvalidFormat, s)
		}
	}
	return s, nil
}

func Convert (ctx context.Context, p rates.Provider, req Request) (Result, error) {
	rate, err := p.Rate(ctx, req.From, req.To)
	if err != nil {
		return Result{}, fmt.Errorf("get rate %s/%s: %w", req.From, req.To, err)
	}
	return Result{
		Amount: req.Amount,
		From: req.From,
		To: req.To,
		Rate: rate,
		Total: req.Amount * rate,
	}, nil
}

func (r Result) String() string {
	return fmt.Sprintf("%.2f %s = %.2f %s\nКурс: 1 %s = %.6g %s",
		r.Amount, r.From, r.Total, r.To, r.From, r.Rate, r.To)
}