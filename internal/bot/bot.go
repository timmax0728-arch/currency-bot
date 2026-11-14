package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	startText   = "Привет! Я бот-конвертер валют. Курсы берутся с сайта Центробанка РФ.\n\nКонвертация появится совсем скоро. Справка: /help"
	helpText    = "Я пока умею отвечать на команды /start  и /help. Конвертация в разработке."
	unknownText = "Не знаю такой команды. Напишите /help."
)

type Bot struct {
	api *tgbotapi.BotAPI
	token string
}

func New(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %s", redact(err, token))
	}
	return &Bot{api: api, token: token}, nil
}

func (b *Bot) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.api.GetUpdatesChan(u)
	defer b.api.StopReceivingUpdates()

	log.Printf("authorized as @%s", b.api.Self.UserName)

	for{
		select{
		case <- ctx.Done():
			return
		case upd, ok := <-updates:
			if !ok{
				return
			}
			if upd.Message == nil {
				continue
			}
			b.handle(upd.Message)
		}
	}
}

func (b *Bot) handle(m *tgbotapi.Message) {
	var text string
	if m.IsCommand() {
		switch m.Command() {
		case "start":
			text = startText
		case "help":
			text = helpText
		default:
			text = unknownText
		}
	} else {
		text = helpText
	}
	b.reply(m.Chat.ID, text)
}

func (b *Bot) reply(chatID int64, text string) {
	if _, err := b.api.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("send message: %s", redact(err, b.token))
	}
}

func redact (err error, token string) string {
	return strings.ReplaceAll(err.Error(), token, "<token>")
}