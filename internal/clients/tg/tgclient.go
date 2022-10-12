package tg

import (
	"context"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/model/messages"
)

type TokenGetter interface {
	Token() string
}

type Client struct {
	client *tgbotapi.BotAPI
}

func New(tokenGetter TokenGetter) (*Client, error) {
	client, err := tgbotapi.NewBotAPI(tokenGetter.Token())
	if err != nil {
		return nil, errors.Wrap(err, "NewBotAPI")
	}

	return &Client{
		client: client,
	}, nil
}

func (c *Client) SendMessage(text string, userID int64) error {
	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "markdown"
	_, err := c.client.Send(msg)
	if err != nil {
		return errors.Wrap(err, "client.Send")
	}
	return nil
}

func (c *Client) ListenUpdates(ctx context.Context, wg *sync.WaitGroup, msgModel *messages.Model) {
	wg.Add(1)
	defer wg.Done()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.client.GetUpdatesChan(u)
	log.Println("listening for messages")

LOOP:
	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

				err := msgModel.IncomingMessage(messages.Message{
					Text:   update.Message.Text,
					UserID: update.Message.From.ID,
				})

				if err != nil {
					log.Println("error processing message:", err)
				}
			}
		case <-ctx.Done():
			log.Println("shutdown listen updates")
			break LOOP
		}
	}
}
