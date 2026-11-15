package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/timmax0728-arch/currency-bot/internal/converter"
	"github.com/timmax0728-arch/currency-bot/internal/rates"
)

const (
	startText = "Привет! Я конвертирую валюты по курсу Центробанка РФ.\n\n" +
		"Напишите, например: 100 usd eur\n\n" +
		"Все команды: /help"
	helpText = "Как пользоваться:\n\n" +
		"• 100 usd eur — перевести 100 долларов в евро (можно и командой: /convert 100 usd eur)\n" +
		"• /rate usd eur — курс одной единицы\n\n" +
		"Коды валют из трёх букв: USD, EUR, CNY, RUB и другие, которые публикует ЦБ РФ. " +
		"Курсы публикуются в рабочие дни."
	unknownText         = "Не знаю такой команды. Напишите /help."
	formatText          = "Не понял запрос. Напишите так: 100 usd eur."
	amountText          = "Сумма должна быть положительным числом, например: 100 или 2.5."
	unknownCurrencyText = "Не знаю такой валюты. Используйте трёхбуквенные коды, например USD, EUR, CNY, RUB."
	failureText         = "Не удалось получить курсы валют. Попробуйте позже."
)

type Bot struct {
	api      *tgbotapi.BotAPI
	token    string
	provider rates.Provider
}

func New(token string, provider rates.Provider) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %s", redact(err, token))
	}
	return &Bot{api: api, token: token, provider: provider}, nil
}

func (b *Bot) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.api.GetUpdatesChan(u)
	defer b.api.StopReceivingUpdates()

	log.Printf("authorized as @%s", b.api.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			return
		case upd, ok := <-updates:
			if !ok {
				return
			}
			if upd.Message == nil {
				continue
			}
			b.handle(ctx, upd.Message)
		}
	}
}

func (b *Bot) handle(ctx context.Context, m *tgbotapi.Message) {
	var text string
	if m.IsCommand() {
		switch m.Command() {
		case "start":
			text = startText
		case "help":
			text = helpText
		case "convert":
			text = b.convert(ctx, m.CommandArguments())
		case "rate":
			text = b.rate(ctx, m.CommandArguments())
		default:
			text = unknownText
		}
	} else {
		text = b.convert(ctx, m.Text)
	}
	b.reply(m.Chat.ID, text)
}

func (b *Bot) convert(ctx context.Context, input string) string {
	req, err := converter.Parse(input)
	if err != nil {
		return b.errorText(err)
	}
	res, err := converter.Convert(ctx, b.provider, req)
	if err != nil {
		return b.errorText(err)
	}
	return res.String()
}

func (b *Bot) rate(ctx context.Context, input string) string {
	return b.convert(ctx, "1 "+input)
}

func (b *Bot) errorText(err error) string {
	switch {
	case errors.Is(err, converter.ErrInvalidFormat):
		return formatText
	case errors.Is(err, converter.ErrInvalidAmount):
		return amountText
	case errors.Is(err, rates.ErrUnknownCurrency):
		return unknownCurrencyText
	default:
		log.Printf("convert: %v", err)
		return failureText
	}
}

func (b *Bot) reply(chatID int64, text string) {
	if _, err := b.api.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("send message: %s", redact(err, b.token))
	}
}

func redact(err error, token string) string {
	return strings.ReplaceAll(err.Error(), token, "<token>")
}
