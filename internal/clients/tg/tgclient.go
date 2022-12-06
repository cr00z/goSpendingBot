package tg

import (
	"context"
	"sync"

	"github.com/cr00z/goSpendingBot/internal/model/messages"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type TokenGetter interface {
	Token() string
}

type Client struct {
	client *tgbotapi.BotAPI
	logger *zap.Logger
}

func New(tokenGetter TokenGetter, logger *zap.Logger) (*Client, error) {
	client, err := tgbotapi.NewBotAPI(tokenGetter.Token())
	if err != nil {
		return nil, errors.Wrap(err, "NewBotAPI")
	}

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

func (c *Client) SendMessage(ctx context.Context, text string, userID int64) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "send message")
	defer span.Finish()

	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "markdown"
	_, err := c.client.Send(msg)

	ext.Error.Set(span, err != nil)

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
	c.logger.Info("listening for messages")

LOOP:
	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				c.logger.Info(
					"command received",
					zap.String("username", update.Message.From.UserName),
					zap.String("text", update.Message.Text),
				)

				err := msgModel.IncomingMessage(ctx, messages.Message{
					Text:   update.Message.Text,
					UserID: update.Message.From.ID,
				})

				if err != nil {
					c.logger.Warn(
						"error processing message:",
						zap.Error(err),
					)
				}
			}
		case <-ctx.Done():
			c.logger.Info("shutdown listen updates")
			break LOOP
		}
	}
}
