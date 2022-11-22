package producer

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
)

var (
	RequestTimeout  = errors.New("report-requests request to consumer timeout")
	ResponseTimeout = errors.New("report-requests response from consumer timeout")
)

type ReportProducer interface {
	SendMessage(userID int64, period string, dateFirst time.Time, dateLast time.Time) error
}

type ProducerOptions struct {
	KafkaTopic  string
	BrokersList []string
}

type Producer struct {
	options  ProducerOptions
	producer sarama.AsyncProducer
}

func New(options ProducerOptions) (*Producer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	config.Producer.Return.Successes = true

	producer, err := sarama.NewAsyncProducer(options.BrokersList, config)
	if err != nil {
		return nil, errors.Wrap(err, "starting Sarama producer")
	}

	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write message:", err)
		}
	}()

	return &Producer{
		options:  options,
		producer: producer,
	}, nil
}

type ReportMessage struct {
	UserID    int64     `json:"user_id"`
	Period    string    `json:"period"`
	DateFirst time.Time `json:"date_first"`
	DateLast  time.Time `json:"date_last"`
}

func (p *Producer) SendMessage(userID int64, period string, dateFirst time.Time, dateLast time.Time) error {
	msg := ReportMessage{userID, period, dateFirst, dateLast}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Println(err.Error())
	}
	saramaMsg := &sarama.ProducerMessage{
		Topic: p.options.KafkaTopic,
		Value: sarama.ByteEncoder(msgBytes),
	}

	log.Println(saramaMsg)
	select {
	case p.producer.Input() <- saramaMsg:
	case <-time.After(5 * time.Second):
		return RequestTimeout
	}

	select {
	case successMsg := <-p.producer.Successes():
		log.Println("Successful to write message, offset:", successMsg.Offset)
	case <-time.After(5 * time.Second):
		return ResponseTimeout
	}

	return nil
}
