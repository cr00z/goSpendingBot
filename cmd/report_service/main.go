package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	_ "github.com/lib/pq"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/kafka/consumers"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/observability"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/report_service/model"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository/postgres_sql"
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
