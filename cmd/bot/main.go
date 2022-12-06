package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/cr00z/goSpendingBot/internal/cache/cache_lru"
	"github.com/cr00z/goSpendingBot/internal/clients/tg"
	"github.com/cr00z/goSpendingBot/internal/config"
	"github.com/cr00z/goSpendingBot/internal/currency/cbrcurrency"
	grpc_report "github.com/cr00z/goSpendingBot/internal/grpc/report/server"
	producer "github.com/cr00z/goSpendingBot/internal/kafka/producers"
	"github.com/cr00z/goSpendingBot/internal/model/messages"
	"github.com/cr00z/goSpendingBot/internal/observability"
	"github.com/cr00z/goSpendingBot/internal/repository/postgres_sql"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	develMode = flag.Bool("devel", false, "development mode")

	KafkaTopic  = "report-requests"
	BrokersList = []string{"kafka:9092"}
)

func main() {
	ctx, cancelFn := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	flag.Parse()

	logger := observability.InitLogger(*develMode)

	observability.InitTracing(logger)

	config, err := config.New()
	if err != nil {
		logger.Fatal("config init failed: ", zap.Error(err))
	}

	metricsSrv := observability.NewMetricsServer(logger)
	metricsSrv.Start()

	tgClient, err := tg.New(config, logger)
	if err != nil {
		logger.Fatal("tg client init failed: ", zap.Error(err))
	}

	db, err := postgres_sql.OpenAndConnect("build/bot/environment.dev")
	if err != nil {
		logger.Fatal(err.Error())
	}
	spRepository := postgres_sql.New(db)

	currencyCache := cache_lru.NewLRUCache("currency", config.CurrencyCacheSize())
	reportCache := cache_lru.NewLRUCache("report", config.ReportCacheSize())

	cbrCurrency, err := cbrcurrency.NewCbrCurrencyStorage(ctx, &wg, logger)
	if err != nil {
		logger.Warn(
			"currency storage temporary failed",
			zap.Error(err),
		)
	}

	reportService, err := producer.New(producer.ProducerOptions{
		KafkaTopic:  KafkaTopic,
		BrokersList: BrokersList,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	msgModel := messages.New(tgClient, spRepository, currencyCache, reportCache, cbrCurrency, reportService)

	go func() {
		err = grpc_report.NewServer(msgModel, tgClient)
		if err != nil {
			logger.Fatal(err.Error())
		}
	}()

	go tgClient.ListenUpdates(ctx, &wg, msgModel)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	cancelFn()

	metricsSrv.Stop()

	wg.Wait()
}
