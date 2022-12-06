package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	consumer "github.com/cr00z/goSpendingBot/internal/kafka/consumers"
	"github.com/cr00z/goSpendingBot/internal/observability"
	"github.com/cr00z/goSpendingBot/internal/report_service/model"
	"github.com/cr00z/goSpendingBot/internal/repository/postgres_sql"
	_ "github.com/lib/pq"
)

var (
	develMode = flag.Bool("devel", false, "development mode")

	KafkaTopic         = "report-requests"
	KafkaConsumerGroup = "report-requests-group"
	BrokersList        = []string{"kafka:9092"}
)

func main() {
	ctx, cancelFn := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFn()
	flag.Parse()

	logger := observability.InitLogger(*develMode)

	db, err := postgres_sql.OpenAndConnect("build/bot/environment.dev")
	if err != nil {
		logger.Fatal(err.Error())
	}
	spRepository := postgres_sql.New(db)

	reportServiceModel := model.New(spRepository)

	consumer, err := consumer.New(ctx,
		consumer.ConsumerOptions{
			KafkaTopic:         KafkaTopic,
			KafkaConsumerGroup: KafkaConsumerGroup,
			BrokersList:        BrokersList,
		},
		reportServiceModel,
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	err = consumer.ListenRequests(ctx, reportServiceModel)
	if err != nil {
		logger.Fatal(err.Error())
	}

	<-ctx.Done()
}
