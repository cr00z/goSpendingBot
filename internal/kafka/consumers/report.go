package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
	grpc_report "github.com/cr00z/goSpendingBot/internal/grpc/report/client"
	"github.com/cr00z/goSpendingBot/internal/report_service/model"
	"github.com/pkg/errors"
)

type ConsumerOptions struct {
	KafkaTopic         string
	KafkaConsumerGroup string
	BrokersList        []string
}

type Consumer struct {
	ctx     context.Context
	options ConsumerOptions
	group   sarama.ConsumerGroup
	model   *model.ReportServiceModel
}

func New(ctx context.Context, options ConsumerOptions, model *model.ReportServiceModel) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGroup, err := sarama.NewConsumerGroup(options.BrokersList, options.KafkaConsumerGroup, config)
	if err != nil {
		return nil, errors.Wrap(err, "starting consumer group")
	}

	return &Consumer{
		ctx:     ctx,
		options: options,
		group:   consumerGroup,
		model:   model,
	}, nil
}

func (consumer *Consumer) ListenRequests(ctx context.Context,
	reportServiceModel *model.ReportServiceModel) error {

	err := consumer.group.Consume(ctx, []string{consumer.options.KafkaTopic}, consumer)
	if err != nil {
		return errors.Wrap(err, "consuming via handler")
	}

	return nil
}

func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

type ReportMessage struct {
	UserID    int64     `json:"user_id"`
	Period    string    `json:"period"`
	DateFirst time.Time `json:"date_first"`
	DateLast  time.Time `json:"date_last"`
}

func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		fmt.Printf("New message received from topic:%s, offset:%d, partition:%d, key:%s, value:%s\n",
			msg.Topic, msg.Offset, msg.Partition, string(msg.Key), string(msg.Value))

		var request ReportMessage
		err := json.Unmarshal(msg.Value, &request)
		if err != nil {
			return errors.Wrap(err, "input message unmarshal error")
		}

		report, err := consumer.model.Store.ReportPeriod(consumer.ctx,
			request.UserID, request.DateFirst, request.DateLast)
		if err != nil {
			return errors.Wrap(err, "service error")
		}

		err = grpc_report.SendReport(request.UserID, request.Period, report)
		if err != nil {
			log.Println(err.Error())
			return errors.Wrap(err, "send to bot error")
		}

		session.MarkMessage(msg, "")
	}

	return nil
}
