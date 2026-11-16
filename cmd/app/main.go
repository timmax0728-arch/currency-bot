package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/timmax0728-arch/currency-bot/internal/bot"
	"github.com/timmax0728-arch/currency-bot/internal/config"
	"github.com/timmax0728-arch/currency-bot/internal/rates"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	provider := rates.NewCBR(cfg.HTTPTimeout, cfg.CacheTTL)

	b, err := bot.New(cfg.TelegramToken, provider)
	if err != nil {
		return err
	}

	log.Println("bot started")
	b.Run(ctx)
	log.Println("bot stopped")
	return nil
}