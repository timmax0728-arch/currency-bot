package rates

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testJSON = `{"Valute":{
"USD":{"CharCode":"USD", "Nominal":1, "Value":90},
"EUR":{"CharCode":"EUR", "Nominal":1, "Value":100},
"JPY":{"CharCode":"JPY", "Nominal":100, "Value":60}
}}`

func newTestProvider(t *testing.T, handler http.HandlerFunc) *CBR {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := NewCBR(2*time.Second, time.Minute)
	c.url = srv.URL
	return c
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(testJSON))
}

func TestRate(t *testing.T) {
	testes := []struct {
		name    string
		from    string
		to      string
		want    float64
		wantErr error
	}{
		{name: "currency to rub", from: "USD", to: "RUB", want: 90},
		{name: "rub to currency", from: "RUB", to: "USD", want: 1.0 / 90},
		{name: "cross rate", from: "USD", to: "EUR", want: 0.9},
		{name: "nominal 100", from: "JPY", to: "RUB", want: 0.6},
		{name: "cross rate with nominal", from: "EUR", to: "JPY", want: 100 / 0.6},
		{name: "lowercase and spaces", from: " usd ", to: "rub", want: 90},
		{name: "same currency", from: "USD", to: "USD", want: 1},
		{name: "unknown target", from: "USD", to: "XXX", wantErr: ErrUnknownCurrency},
		{name: "unknown source", from: "XXX", to: "USD", wantErr: ErrUnknownCurrency},
	}

	for _, tt := range testes {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProvider(t, okHandler)

			got, err := p.Rate(context.Background(), tt.from, tt.to)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got error %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error : %v", err)
			}

			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRate_APIErrors(t *testing.T) {
	testes := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("not json"))
			},
		},
		{
			name: "no currencies",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"Valute":{}}`))
			},
		},
	}

	for _, tt := range testes {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProvider(t, tt.handler)

			if _, err := p.Rate(context.Background(), "USD", "RUB"); err == nil {
				t.Fatal("unexpected error, got nil")
			}
		})
	}
}

func TestRate_Timeout(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(testJSON))
	})
	p.client.Timeout = 50 * time.Millisecond

	if _, err := p.Rate(context.Background(), "USD", "RUB"); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// Сколько раз клиент ходил на сервер, считаем счётчиком в обработчике.

func TestRate_Cache(t *testing.T) {
	var calls atomic.Int32
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		okHandler(w, r)
	})

	// Подменяем часы, чтобы «перемотать» время без ожидания.
	now := time.Now()
	p.now = func() time.Time { return now }
	p.ttl = time.Minute
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := p.Rate(ctx, "USD", "RUB"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("after 3 calls: server hit %d times, want 1", got)
	}

	// Другая пара валют берётся из того же кэша.
	if _, err := p.Rate(ctx, "USD", "EUR"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("different pair: server hit %d times, want 1", got)
	}

	// Прошло больше TTL: кэш устарел, нужно скачать заново.
	now = now.Add(2 * time.Minute)
	if _, err := p.Rate(ctx, "USD", "RUB"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("after expiry: server hit %d times, want 2", got)
	}
}

func TestRate_ErrorIsNotCached(t *testing.T) {
	var calls atomic.Int32
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 { // первый раз сервер «ломается»
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		okHandler(w, r)
	})
	ctx := context.Background()

	if _, err := p.Rate(ctx, "USD", "RUB"); err == nil {
		t.Fatal("expected error on first call")
	}
	if _, err := p.Rate(ctx, "USD", "RUB"); err != nil {
		t.Fatalf("second call should succeed, got: %v", err)
	}
}

func TestRate_ConcurrentCalls(t *testing.T) {
	var calls atomic.Int32
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		okHandler(w, r)
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.Rate(context.Background(), "USD", "RUB"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Errorf("20 parallel calls: server hit %d times, want 1", got)
	}
}
