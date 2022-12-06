package messages

import (
	"context"
	"testing"

	mocks "github.com/cr00z/goSpendingBot/internal/mocks/messages"
	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
)

func Test_OnStartCommand_ShouldAnswerWithIntroMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)

	span, ctx := opentracing.StartSpanFromContext(context.TODO(), "incoming message")
	span.SetTag("command", "/start")
	defer span.Finish()

	sender.EXPECT().SendMessage(ctx, messageHello+"\n\n"+messageHelp, int64(123))

	model := New(sender, nil, nil, nil, nil, nil)
	err := model.IncomingMessage(context.TODO(), Message{
		Text:   "/start",
		UserID: 123,
	})

	assert.NoError(t, err)
}

func Test_OnUnknownCommand_ShouldAnswerWithHelpMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)

	span, ctx := opentracing.StartSpanFromContext(context.TODO(), "incoming message")
	span.SetTag("command", "/start")
	defer span.Finish()

	sender.EXPECT().SendMessage(ctx, "Я не знаю эту команду", int64(123))

	model := New(sender, nil, nil, nil, nil, nil)
	err := model.IncomingMessage(context.TODO(), Message{
		Text:   "some text",
		UserID: 123,
	})

	assert.NoError(t, err)
}
