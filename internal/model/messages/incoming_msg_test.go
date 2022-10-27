package messages

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mocks "gitlab.ozon.dev/netrebinr/netrebin-roman/internal/mocks/messages"
)

func Test_OnStartCommand_ShouldAnswerWithIntroMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	sender.EXPECT().SendMessage(messageHello+"\n\n"+messageHelp, int64(123))

	model := New(sender, nil, nil)
	err := model.IncomingMessage(context.TODO(), Message{
		Text:   "/start",
		UserID: 123,
	})

	assert.NoError(t, err)
}

func Test_OnUnknownCommand_ShouldAnswerWithHelpMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	sender.EXPECT().SendMessage("Я не знаю эту команду", int64(123))

	model := New(sender, nil, nil)
	err := model.IncomingMessage(context.TODO(), Message{
		Text:   "some text",
		UserID: 123,
	})

	assert.NoError(t, err)
}
