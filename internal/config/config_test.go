package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	testes := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "all values set",
			env: map[string]string{
				"TELEGRAM_TOKEN": "abc",
				"HTTP_TIMEOUT":   "3s",
				"CACHE_TTL":      "1h",
			},
			want: Config{TelegramToken: "abc", HTTPTimeout: 3 * time.Second, CacheTTL: time.Hour},
		},
		{
			name: "defaults",
			env:  map[string]string{"TELEGRAM_TOKEN": "abc"},
			want: Config{TelegramToken: "abc", HTTPTimeout: 10 * time.Second, CacheTTL: 10 * time.Minute},
		},
		{
			name:    "missing token",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name:    "invalid duration",
			env:     map[string]string{"TELEGRAM_TOKEN": "abc", "HTTP_TIMEOUT": "abc"},
			wantErr: true,
		},
		{
			name: "zero duration",
			env: map[string]string{"TELEGRAM_TOKEN": "abc", "CACHE_TTL": "0s"},
			wantErr : true,
		},
	}

	for _, tt := range testes {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range []string{"TELEGRAM_TOKEN", "HTTP_TIMEOUT", "CACHE_TTL"} {
				t.Setenv(k, "")
			}

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return

			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}
